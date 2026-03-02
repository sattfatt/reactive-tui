package widget

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"unicode/utf8"

	"github.com/creack/pty"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/signal"
	"github.com/satyam/reactive-tui/style"
)

type termCell struct {
	Ch    rune
	Style style.Style
}

type Terminal struct {
	Base
	Cmd      *exec.Cmd
	ptmx     *os.File
	buf      [][]termCell
	rows     int
	cols     int
	curRow   int
	curCol   int
	curStyle style.Style
	mu       sync.Mutex
	done     chan struct{}
	started  bool

	// ANSI parser state
	parseState int // 0=Normal, 1=Escape, 2=CSI, 3=ESC-(
	csiBuf     []byte

	// Character set
	charset int // 0=normal (B), 1=line drawing (0)

	// Scroll region
	scrollTop    int
	scrollBottom int

	// Saved cursor
	savedRow   int
	savedCol   int
	savedStyle style.Style

	// UTF-8 accumulation buffer
	utf8Buf  [utf8.UTFMax]byte
	utf8Len  int // bytes accumulated so far
	utf8Need int // total bytes needed for current rune

	// DEC private modes
	autoWrap      bool // DECAWM — auto-wrap at right margin (CSI ? 7 h/l)
	cursorKeysApp bool // DECCKM — cursor keys send application sequences (CSI ? 1 h/l)
	wrapPending   bool // deferred wrap: cursor hit right margin, wrap on next printable char

	// Alternate screen buffer
	altBuf      [][]termCell
	altRows     int
	altCols     int
	altCurRow   int
	altCurCol   int
	altCurStyle style.Style
	altActive   bool
}

const (
	parseNormal   = 0
	parseEscape   = 1
	parseCSI      = 2
	parseCharset  = 3 // after ESC (
	parseOSC      = 4 // after ESC ]
)

// ACS (Alternate Character Set) line-drawing map
var acsMap = map[rune]rune{
	'j': '┘', 'k': '┐', 'l': '┌', 'm': '└',
	'n': '┼', 'q': '─', 't': '├', 'u': '┤',
	'v': '┴', 'w': '┬', 'x': '│',
	'a': '▒', 'f': '°', 'g': '±', 'h': '#',
	'o': '⎺', 'p': '⎻', 'r': '⎼', 's': '⎽',
	'~': '·', 'y': '≤', 'z': '≥', '{': 'π',
	'|': '≠', '}': '£', '0': '▮',
}

func NewTerminal(name string, args ...string) *Terminal {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	return &Terminal{
		Base: Base{
			Style: style.Style{FG: style.CurrentTheme.FG, BG: style.CurrentTheme.BG, Border: style.BorderSingle},
			Flex:  FlexProps{Basis: -1, Grow: 1, Shrink: 1, MinHeight: 10, MinWidth: 20},
		},
		Cmd:      cmd,
		curStyle: style.Style{FG: style.CurrentTheme.FG, BG: style.CurrentTheme.BG},
		done:     make(chan struct{}),
		autoWrap: true,
	}
}

func (t *Terminal) Focusable() bool   { return true }
func (t *Terminal) IsEditable() bool  { return true }
func (t *Terminal) EscapesToExit() int { return 3 }

// Start marks the terminal as ready to launch. The PTY is started lazily
// on the first Render call, when the actual container size is known.
func (t *Terminal) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.started = true
}

func (t *Terminal) startPTY(rows, cols int) error {
	t.rows = rows
	t.cols = cols
	t.allocBuf()

	ptmx, err := pty.StartWithSize(t.Cmd, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
	if err != nil {
		return err
	}
	t.ptmx = ptmx

	go t.readLoop()
	return nil
}

func (t *Terminal) allocBuf() {
	t.scrollTop = 0
	t.scrollBottom = t.rows - 1
	t.buf = make([][]termCell, t.rows)
	for i := range t.buf {
		t.buf[i] = make([]termCell, t.cols)
		for j := range t.buf[i] {
			t.buf[i][j] = termCell{Ch: ' ', Style: t.curStyle}
		}
	}
}

func (t *Terminal) readLoop() {
	defer close(t.done)
	buf := make([]byte, 4096)
	for {
		n, err := t.ptmx.Read(buf)
		if n > 0 {
			t.mu.Lock()
			t.processBuf(buf[:n])
			t.mu.Unlock()
			if signal.OnChange != nil {
				signal.OnChange()
			}
		}
		if err != nil {
			return
		}
	}
}

func (t *Terminal) processBuf(data []byte) {
	for _, b := range data {
		// If we're accumulating a multi-byte UTF-8 sequence, continue it
		if t.utf8Len > 0 {
			if b&0xC0 == 0x80 { // continuation byte
				t.utf8Buf[t.utf8Len] = b
				t.utf8Len++
				if t.utf8Len >= t.utf8Need {
					r, _ := utf8.DecodeRune(t.utf8Buf[:t.utf8Len])
					t.utf8Len = 0
					t.putChar(r)
				}
				continue
			}
			// Not a valid continuation — discard partial and reprocess this byte
			t.utf8Len = 0
		}

		switch t.parseState {
		case parseNormal:
			switch b {
			case 0x1b: // ESC
				t.parseState = parseEscape
			case '\n':
				t.wrapPending = false
				if t.curRow == t.scrollBottom {
					t.scrollUpRegion()
				} else if t.curRow < t.rows-1 {
					t.curRow++
				}
			case '\r':
				t.wrapPending = false
				t.curCol = 0
			case '\b':
				t.wrapPending = false
				if t.curCol > 0 {
					t.curCol--
				}
			case '\t':
				t.wrapPending = false
				t.curCol = (t.curCol + 8) &^ 7
				if t.curCol >= t.cols {
					t.curCol = t.cols - 1
				}
			default:
				if b >= 0xC0 {
					// Start of multi-byte UTF-8 sequence
					t.utf8Buf[0] = b
					t.utf8Len = 1
					switch {
					case b < 0xE0:
						t.utf8Need = 2
					case b < 0xF0:
						t.utf8Need = 3
					default:
						t.utf8Need = 4
					}
				} else if b >= 0x20 && b < 0x80 {
					t.putChar(rune(b))
				}
			}
		case parseEscape:
			t.wrapPending = false
			switch b {
			case '[':
				t.parseState = parseCSI
				t.csiBuf = t.csiBuf[:0]
			case '(':
				t.parseState = parseCharset
			case ']':
				t.parseState = parseOSC
				t.csiBuf = t.csiBuf[:0]
			case '7': // Save cursor
				t.savedRow = t.curRow
				t.savedCol = t.curCol
				t.savedStyle = t.curStyle
				t.parseState = parseNormal
			case '8': // Restore cursor
				t.curRow = t.savedRow
				t.curCol = t.savedCol
				t.curStyle = t.savedStyle
				t.parseState = parseNormal
			case 'M': // Reverse Index (scroll down)
				if t.curRow == t.scrollTop {
					t.scrollDown()
				} else if t.curRow > 0 {
					t.curRow--
				}
				t.parseState = parseNormal
			case 'D': // Index (scroll up)
				if t.curRow == t.scrollBottom {
					t.scrollUpRegion()
				} else if t.curRow < t.rows-1 {
					t.curRow++
				}
				t.parseState = parseNormal
			case 'E': // Next Line
				t.curCol = 0
				if t.curRow == t.scrollBottom {
					t.scrollUpRegion()
				} else if t.curRow < t.rows-1 {
					t.curRow++
				}
				t.parseState = parseNormal
			default:
				t.parseState = parseNormal
			}
		case parseCSI:
			if b >= 0x40 && b <= 0x7e {
				// Final byte
				t.handleCSI(b)
				t.parseState = parseNormal
			} else {
				t.csiBuf = append(t.csiBuf, b)
			}
		case parseCharset:
			switch b {
			case '0':
				t.charset = 1 // line drawing
			default:
				t.charset = 0 // normal (B or anything else)
			}
			t.parseState = parseNormal
		case parseOSC:
			// Consume until ST (ESC \ or BEL)
			if b == 0x07 { // BEL terminates OSC
				t.parseState = parseNormal
			} else if b == 0x1b {
				// Next byte should be '\' for ST, but just exit
				t.parseState = parseNormal
			}
		}
	}
}

func (t *Terminal) putChar(ch rune) {
	// Apply ACS mapping if in line-drawing charset
	if t.charset == 1 {
		if mapped, ok := acsMap[ch]; ok {
			ch = mapped
		}
	}
	if t.curRow < 0 || t.curRow >= t.rows {
		return
	}

	w := runewidth.RuneWidth(ch)
	if w <= 0 {
		w = 1
	}

	// Execute pending wrap from previous character that hit the right margin
	if t.wrapPending {
		t.wrapPending = false
		if t.autoWrap {
			t.curCol = 0
			if t.curRow == t.scrollBottom {
				t.scrollUpRegion()
			} else if t.curRow < t.rows-1 {
				t.curRow++
			}
		}
	}

	// Clamp column (safety)
	if t.curCol < 0 || t.curCol >= t.cols {
		return
	}

	// If wide char won't fit on this line, pad and wrap
	if w > 1 && t.curCol+w > t.cols {
		for c := t.curCol; c < t.cols; c++ {
			t.buf[t.curRow][c] = termCell{Ch: ' ', Style: t.curStyle}
		}
		if t.autoWrap {
			t.curCol = 0
			if t.curRow == t.scrollBottom {
				t.scrollUpRegion()
			} else if t.curRow < t.rows-1 {
				t.curRow++
			}
		} else {
			return // can't fit, nowhere to go
		}
	}

	t.buf[t.curRow][t.curCol] = termCell{Ch: ch, Style: t.curStyle}
	// For wide characters, fill the second cell with a zero-width placeholder
	if w > 1 && t.curCol+1 < t.cols {
		t.buf[t.curRow][t.curCol+1] = termCell{Ch: 0, Style: t.curStyle}
	}
	t.curCol += w

	// If we've hit (or passed) the right margin, set pending wrap
	// instead of immediately wrapping — the wrap happens on the NEXT printable char
	if t.curCol >= t.cols {
		t.curCol = t.cols - 1
		if t.autoWrap {
			t.wrapPending = true
		}
	}
}

// scrollUpRegion scrolls the scroll region up by one line.
func (t *Terminal) scrollUpRegion() {
	top := t.scrollTop
	bot := t.scrollBottom
	if top >= bot || bot >= t.rows {
		return
	}
	copy(t.buf[top:], t.buf[top+1:bot+1])
	t.buf[bot] = make([]termCell, t.cols)
	for j := range t.buf[bot] {
		t.buf[bot][j] = termCell{Ch: ' ', Style: t.curStyle}
	}
}

// scrollDown scrolls the scroll region down by one line (insert line at top).
func (t *Terminal) scrollDown() {
	top := t.scrollTop
	bot := t.scrollBottom
	if top >= bot || bot >= t.rows {
		return
	}
	copy(t.buf[top+1:bot+1], t.buf[top:bot])
	t.buf[top] = make([]termCell, t.cols)
	for j := range t.buf[top] {
		t.buf[top][j] = termCell{Ch: ' ', Style: t.curStyle}
	}
}

func (t *Terminal) enterAltScreen() {
	// Save main screen state
	t.altBuf = t.buf
	t.altRows = t.rows
	t.altCols = t.cols
	t.altCurRow = t.curRow
	t.altCurCol = t.curCol
	t.altCurStyle = t.curStyle
	t.altActive = true

	// Create fresh buffer for alt screen
	t.allocBuf()
	t.curRow = 0
	t.curCol = 0
}

func (t *Terminal) exitAltScreen() {
	if !t.altActive {
		return
	}
	// Restore main screen
	t.buf = t.altBuf
	t.rows = t.altRows
	t.cols = t.altCols
	t.curRow = t.altCurRow
	t.curCol = t.altCurCol
	t.curStyle = t.altCurStyle
	t.scrollTop = 0
	t.scrollBottom = t.rows - 1
	t.altActive = false
}

func (t *Terminal) handleCSI(final byte) {
	params := parseCSIParams(t.csiBuf)

	// Any cursor movement cancels pending wrap
	switch final {
	case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'd', 'f', 'r':
		t.wrapPending = false
	}

	switch final {
	case 'A': // Cursor Up
		n := paramOrDefault(params, 0, 1)
		t.curRow -= n
		if t.curRow < 0 {
			t.curRow = 0
		}
	case 'B': // Cursor Down
		n := paramOrDefault(params, 0, 1)
		t.curRow += n
		if t.curRow >= t.rows {
			t.curRow = t.rows - 1
		}
	case 'C': // Cursor Forward
		n := paramOrDefault(params, 0, 1)
		t.curCol += n
		if t.curCol >= t.cols {
			t.curCol = t.cols - 1
		}
	case 'D': // Cursor Back
		n := paramOrDefault(params, 0, 1)
		t.curCol -= n
		if t.curCol < 0 {
			t.curCol = 0
		}
	case 'H', 'f': // Cursor Position
		row := paramOrDefault(params, 0, 1) - 1
		col := paramOrDefault(params, 1, 1) - 1
		if row < 0 {
			row = 0
		}
		if row >= t.rows {
			row = t.rows - 1
		}
		if col < 0 {
			col = 0
		}
		if col >= t.cols {
			col = t.cols - 1
		}
		t.curRow = row
		t.curCol = col
	case 'J': // Erase in Display
		n := paramOrDefault(params, 0, 0)
		switch n {
		case 0: // Clear from cursor to end
			t.clearRange(t.curRow, t.curCol, t.rows-1, t.cols-1)
		case 1: // Clear from start to cursor
			t.clearRange(0, 0, t.curRow, t.curCol)
		case 2: // Clear entire screen
			t.clearRange(0, 0, t.rows-1, t.cols-1)
		}
	case 'K': // Erase in Line
		n := paramOrDefault(params, 0, 0)
		switch n {
		case 0: // Clear from cursor to end of line
			t.clearRange(t.curRow, t.curCol, t.curRow, t.cols-1)
		case 1: // Clear from start of line to cursor
			t.clearRange(t.curRow, 0, t.curRow, t.curCol)
		case 2: // Clear entire line
			t.clearRange(t.curRow, 0, t.curRow, t.cols-1)
		}
	case 'm': // SGR - Select Graphic Rendition
		t.handleSGR(params)
	case 'r': // DECSTBM - Set Scrolling Region
		top := paramOrDefault(params, 0, 1) - 1
		bot := paramOrDefault(params, 1, t.rows) - 1
		if top < 0 {
			top = 0
		}
		if bot >= t.rows {
			bot = t.rows - 1
		}
		if top < bot {
			t.scrollTop = top
			t.scrollBottom = bot
		}
		t.curRow = 0
		t.curCol = 0
	case 'L': // Insert Lines
		n := paramOrDefault(params, 0, 1)
		for i := 0; i < n; i++ {
			if t.curRow <= t.scrollBottom {
				copy(t.buf[t.curRow+1:t.scrollBottom+1], t.buf[t.curRow:t.scrollBottom])
				t.buf[t.curRow] = make([]termCell, t.cols)
				for j := range t.buf[t.curRow] {
					t.buf[t.curRow][j] = termCell{Ch: ' ', Style: t.curStyle}
				}
			}
		}
	case 'M': // Delete Lines
		n := paramOrDefault(params, 0, 1)
		for i := 0; i < n; i++ {
			if t.curRow <= t.scrollBottom {
				copy(t.buf[t.curRow:t.scrollBottom], t.buf[t.curRow+1:t.scrollBottom+1])
				t.buf[t.scrollBottom] = make([]termCell, t.cols)
				for j := range t.buf[t.scrollBottom] {
					t.buf[t.scrollBottom][j] = termCell{Ch: ' ', Style: t.curStyle}
				}
			}
		}
	case '@': // Insert Characters
		n := paramOrDefault(params, 0, 1)
		row := t.curRow
		if row >= 0 && row < t.rows {
			for i := t.cols - 1; i >= t.curCol+n; i-- {
				t.buf[row][i] = t.buf[row][i-n]
			}
			for i := t.curCol; i < t.curCol+n && i < t.cols; i++ {
				t.buf[row][i] = termCell{Ch: ' ', Style: t.curStyle}
			}
		}
	case 'P': // Delete Characters
		n := paramOrDefault(params, 0, 1)
		row := t.curRow
		if row >= 0 && row < t.rows {
			for i := t.curCol; i < t.cols-n; i++ {
				t.buf[row][i] = t.buf[row][i+n]
			}
			for i := t.cols - n; i < t.cols; i++ {
				if i >= 0 {
					t.buf[row][i] = termCell{Ch: ' ', Style: t.curStyle}
				}
			}
		}
	case 'X': // Erase Characters
		n := paramOrDefault(params, 0, 1)
		row := t.curRow
		if row >= 0 && row < t.rows {
			for i := t.curCol; i < t.curCol+n && i < t.cols; i++ {
				t.buf[row][i] = termCell{Ch: ' ', Style: t.curStyle}
			}
		}
	case 'S': // Scroll Up
		n := paramOrDefault(params, 0, 1)
		for i := 0; i < n; i++ {
			t.scrollUpRegion()
		}
	case 'T': // Scroll Down
		n := paramOrDefault(params, 0, 1)
		for i := 0; i < n; i++ {
			t.scrollDown()
		}
	case 'G': // Cursor Horizontal Absolute
		col := paramOrDefault(params, 0, 1) - 1
		if col < 0 {
			col = 0
		}
		if col >= t.cols {
			col = t.cols - 1
		}
		t.curCol = col
	case 'd': // Vertical Position Absolute
		row := paramOrDefault(params, 0, 1) - 1
		if row < 0 {
			row = 0
		}
		if row >= t.rows {
			row = t.rows - 1
		}
		t.curRow = row
	case 'h', 'l': // Set/Reset Mode
		set := final == 'h'
		if len(t.csiBuf) > 0 && t.csiBuf[0] == '?' {
			// DEC private modes
			for _, p := range params {
				switch p {
				case 1: // DECCKM — cursor keys
					t.cursorKeysApp = set
				case 7: // DECAWM — auto-wrap
					t.autoWrap = set
				case 25: // DECTCEM — cursor visibility (ignore for now)
				case 47, 1047: // Alt screen buffer (without save/restore cursor)
					if set && !t.altActive {
						t.enterAltScreen()
					} else if !set && t.altActive {
						t.exitAltScreen()
					}
				case 1049: // Alt screen buffer with save/restore cursor
					if set && !t.altActive {
						t.savedRow = t.curRow
						t.savedCol = t.curCol
						t.savedStyle = t.curStyle
						t.enterAltScreen()
					} else if !set && t.altActive {
						t.exitAltScreen()
						t.curRow = t.savedRow
						t.curCol = t.savedCol
						t.curStyle = t.savedStyle
					}
				}
			}
		}
	case 'c': // Device Attributes — respond as VT220
		if t.ptmx != nil {
			_, _ = t.ptmx.Write([]byte("\x1b[?62;22c"))
		}
	case 'n': // Device Status Report
		n := paramOrDefault(params, 0, 0)
		if n == 6 && t.ptmx != nil {
			// Report cursor position (1-indexed)
			resp := fmt.Sprintf("\x1b[%d;%dR", t.curRow+1, t.curCol+1)
			_, _ = t.ptmx.Write([]byte(resp))
		}
	}
}

func (t *Terminal) clearRange(r1, c1, r2, c2 int) {
	for r := r1; r <= r2 && r < t.rows; r++ {
		startC := 0
		endC := t.cols - 1
		if r == r1 {
			startC = c1
		}
		if r == r2 {
			endC = c2
		}
		for c := startC; c <= endC && c < t.cols; c++ {
			t.buf[r][c] = termCell{Ch: ' ', Style: t.curStyle}
		}
	}
}

func (t *Terminal) handleSGR(params []int) {
	if len(params) == 0 {
		params = []int{0}
	}
	i := 0
	for i < len(params) {
		p := params[i]
		switch {
		case p == 0: // Reset
			t.curStyle = style.Style{FG: style.CurrentTheme.FG, BG: style.CurrentTheme.BG}
		case p == 1: // Bold
			t.curStyle.Bold = true
		case p == 3: // Italic
			t.curStyle.Italic = true
		case p == 22: // Not bold
			t.curStyle.Bold = false
		case p == 23: // Not italic
			t.curStyle.Italic = false
		case p >= 30 && p <= 37: // FG color
			t.curStyle.FG = ansi4Color(p - 30)
		case p == 38: // Extended FG
			i += t.parseExtendedColor(params[i:], true)
		case p == 39: // Default FG
			t.curStyle.FG = style.CurrentTheme.FG
		case p >= 40 && p <= 47: // BG color
			t.curStyle.BG = ansi4Color(p - 40)
		case p == 48: // Extended BG
			i += t.parseExtendedColor(params[i:], false)
		case p == 49: // Default BG
			t.curStyle.BG = style.CurrentTheme.BG
		case p >= 90 && p <= 97: // Bright FG
			t.curStyle.FG = ansi4Color(p - 90 + 8)
		case p >= 100 && p <= 107: // Bright BG
			t.curStyle.BG = ansi4Color(p - 100 + 8)
		}
		i++
	}
}

func (t *Terminal) parseExtendedColor(params []int, fg bool) int {
	if len(params) < 2 {
		return 0
	}
	switch params[1] {
	case 5: // 256-color
		if len(params) < 3 {
			return 1
		}
		color := tcell.PaletteColor(params[2])
		if fg {
			t.curStyle.FG = color
		} else {
			t.curStyle.BG = color
		}
		return 2
	case 2: // Truecolor
		if len(params) < 5 {
			return 1
		}
		color := tcell.NewRGBColor(int32(params[2]), int32(params[3]), int32(params[4]))
		if fg {
			t.curStyle.FG = color
		} else {
			t.curStyle.BG = color
		}
		return 4
	}
	return 0
}

func ansi4Color(idx int) tcell.Color {
	colors := [16]tcell.Color{
		tcell.ColorBlack, tcell.ColorMaroon, tcell.ColorGreen, tcell.ColorOlive,
		tcell.ColorNavy, tcell.ColorPurple, tcell.ColorTeal, tcell.ColorSilver,
		tcell.ColorGray, tcell.ColorRed, tcell.ColorLime, tcell.ColorYellow,
		tcell.ColorBlue, tcell.ColorFuchsia, tcell.ColorAqua, tcell.ColorWhite,
	}
	if idx >= 0 && idx < 16 {
		return colors[idx]
	}
	return tcell.ColorWhite
}

func parseCSIParams(buf []byte) []int {
	if len(buf) == 0 {
		return nil
	}
	// Skip private mode prefixes (?, >, =)
	start := 0
	if len(buf) > 0 && (buf[0] == '?' || buf[0] == '>' || buf[0] == '=') {
		start = 1
	}
	buf = buf[start:]
	if len(buf) == 0 {
		return nil
	}
	var params []int
	n := 0
	hasDigit := false
	for _, b := range buf {
		if b >= '0' && b <= '9' {
			n = n*10 + int(b-'0')
			hasDigit = true
		} else if b == ';' {
			params = append(params, n)
			n = 0
			hasDigit = false
		}
	}
	if hasDigit {
		params = append(params, n)
	}
	return params
}

func paramOrDefault(params []int, idx, def int) int {
	if idx < len(params) && params[idx] > 0 {
		return params[idx]
	}
	return def
}

func (t *Terminal) HandleKey(ev KeyEvent) bool {
	if t.ptmx == nil || !t.Editing {
		return false
	}

	var data []byte
	switch tcell.Key(ev.Key) {
	case tcell.KeyRune:
		buf := make([]byte, utf8.UTFMax)
		n := utf8.EncodeRune(buf, ev.Rune)
		data = buf[:n]
	case tcell.KeyEnter:
		data = []byte{'\r'}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		data = []byte{0x7f}
	case tcell.KeyTab:
		data = []byte{'\t'}
	case tcell.KeyEscape:
		data = []byte{0x1b}
	case tcell.KeyUp:
		if t.cursorKeysApp {
			data = []byte("\x1bOA")
		} else {
			data = []byte("\x1b[A")
		}
	case tcell.KeyDown:
		if t.cursorKeysApp {
			data = []byte("\x1bOB")
		} else {
			data = []byte("\x1b[B")
		}
	case tcell.KeyRight:
		if t.cursorKeysApp {
			data = []byte("\x1bOC")
		} else {
			data = []byte("\x1b[C")
		}
	case tcell.KeyLeft:
		if t.cursorKeysApp {
			data = []byte("\x1bOD")
		} else {
			data = []byte("\x1b[D")
		}
	case tcell.KeyHome:
		data = []byte("\x1b[H")
	case tcell.KeyEnd:
		data = []byte("\x1b[F")
	case tcell.KeyDelete:
		data = []byte("\x1b[3~")
	case tcell.KeyPgUp:
		data = []byte("\x1b[5~")
	case tcell.KeyPgDn:
		data = []byte("\x1b[6~")
	default:
		// Ctrl keys
		key := tcell.Key(ev.Key)
		if key >= tcell.KeyCtrlA && key <= tcell.KeyCtrlZ {
			data = []byte{byte(key - tcell.KeyCtrlA + 1)}
		}
	}

	if data != nil {
		_, _ = t.ptmx.Write(data)
		return true
	}
	return false
}

func (t *Terminal) Render(r *render.Renderer, x, y, w, h int) {
	t.Base.SetRect(x, y, w, h)
	st := t.Style

	borderSt := st
	if t.Focused {
		if t.Editing {
			borderSt.FG = style.CurrentTheme.EditFocusFG
		} else {
			borderSt.FG = style.CurrentTheme.NavFocusFG
		}
	} else {
		borderSt.FG = style.CurrentTheme.BorderFG
	}
	if st.Border != style.BorderNone {
		r.DrawBorder(x, y, w, h, st.Border, borderSt)
		t.Base.RenderLabel(r, x, y, w)
	}

	ix, iy, iw, ih := st.InnerRect(x, y, w, h)
	if iw <= 0 || ih <= 0 {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Lazy-start PTY at the correct size on first render
	if t.ptmx == nil && t.started {
		_ = t.startPTY(ih, iw)
	}

	// Resize if needed
	if t.ptmx != nil && (iw != t.cols || ih != t.rows) {
		t.Resize(ih, iw)
	}

	// Blit buffer
	for row := 0; row < t.rows && row < ih; row++ {
		for col := 0; col < t.cols && col < iw; col++ {
			cell := t.buf[row][col]
			if cell.Ch == 0 {
				// Wide char placeholder — skip (tcell handles it via the primary cell)
				continue
			}
			r.Screen.SetContent(ix+col, iy+row, cell.Ch, nil, cell.Style.TcellStyle())
		}
	}

	// Draw cursor
	if t.Focused && t.curRow < ih && t.curCol < iw {
		cell := t.buf[t.curRow][t.curCol]
		cursorStyle := tcell.StyleDefault.
			Foreground(cell.Style.BG).
			Background(cell.Style.FG)
		r.Screen.SetContent(ix+t.curCol, iy+t.curRow, cell.Ch, nil, cursorStyle)
	}
}

func (t *Terminal) Resize(rows, cols int) {
	if rows <= 0 || cols <= 0 {
		return
	}
	oldBuf := t.buf
	oldRows := t.rows
	oldCols := t.cols

	t.rows = rows
	t.cols = cols
	t.allocBuf()

	// Copy old content
	for r := 0; r < min(oldRows, rows); r++ {
		for c := 0; c < min(oldCols, cols); c++ {
			if r < len(oldBuf) && c < len(oldBuf[r]) {
				t.buf[r][c] = oldBuf[r][c]
			}
		}
	}

	// Clamp cursor
	if t.curRow >= t.rows {
		t.curRow = t.rows - 1
	}
	if t.curCol >= t.cols {
		t.curCol = t.cols - 1
	}

	// Resize PTY if running
	if t.ptmx != nil {
		_ = pty.Setsize(t.ptmx, &pty.Winsize{
			Rows: uint16(rows),
			Cols: uint16(cols),
		})
	}
}

func (t *Terminal) Stop() {
	if t.ptmx == nil {
		return
	}
	_ = t.ptmx.Close()
	if t.Cmd != nil && t.Cmd.Process != nil {
		_ = t.Cmd.Wait()
	}
	<-t.done
}

// Write sends raw bytes to the terminal for processing (useful for testing).
func (t *Terminal) Write(data []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.processBuf(data)
}

// GetCell returns the cell at the given position (for testing).
func (t *Terminal) GetCell(row, col int) (rune, style.Style) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if row >= 0 && row < t.rows && col >= 0 && col < t.cols {
		c := t.buf[row][col]
		return c.Ch, c.Style
	}
	return ' ', style.Style{}
}

// CursorPos returns the current cursor position.
func (t *Terminal) CursorPos() (int, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.curRow, t.curCol
}

// unexported helper for io.Writer — used only in readLoop
var _ io.Reader = (*os.File)(nil)
