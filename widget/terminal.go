package widget

import (
	"io"
	"os"
	"os/exec"
	"sync"
	"unicode/utf8"

	"github.com/creack/pty"
	"github.com/gdamore/tcell/v2"
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

	// ANSI parser state
	parseState int // 0=Normal, 1=Escape, 2=CSI
	csiBuf     []byte
}

const (
	parseNormal = 0
	parseEscape = 1
	parseCSI    = 2
)

func NewTerminal(name string, args ...string) *Terminal {
	return &Terminal{
		Base: Base{
			Style: style.Style{FG: tcell.ColorWhite, BG: tcell.ColorDefault, Border: style.BorderSingle},
			Flex:  FlexProps{Basis: -1, Grow: 1, Shrink: 1, MinHeight: 10, MinWidth: 20},
		},
		Cmd:      exec.Command(name, args...),
		curStyle: style.Style{FG: tcell.ColorWhite, BG: tcell.ColorDefault},
		done:     make(chan struct{}),
	}
}

func (t *Terminal) Focusable() bool { return true }

func (t *Terminal) Start(rows, cols int) error {
	t.mu.Lock()
	t.rows = rows
	t.cols = cols
	t.allocBuf()
	t.mu.Unlock()

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
		switch t.parseState {
		case parseNormal:
			switch b {
			case 0x1b: // ESC
				t.parseState = parseEscape
			case '\n':
				t.curRow++
				if t.curRow >= t.rows {
					t.scrollUp()
					t.curRow = t.rows - 1
				}
			case '\r':
				t.curCol = 0
			case '\b':
				if t.curCol > 0 {
					t.curCol--
				}
			case '\t':
				t.curCol = (t.curCol + 8) &^ 7
				if t.curCol >= t.cols {
					t.curCol = t.cols - 1
				}
			default:
				if b >= 0x20 {
					t.putChar(rune(b))
				}
			}
		case parseEscape:
			if b == '[' {
				t.parseState = parseCSI
				t.csiBuf = t.csiBuf[:0]
			} else {
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
		}
	}
}

func (t *Terminal) putChar(ch rune) {
	if t.curRow >= 0 && t.curRow < t.rows && t.curCol >= 0 && t.curCol < t.cols {
		t.buf[t.curRow][t.curCol] = termCell{Ch: ch, Style: t.curStyle}
		t.curCol++
		if t.curCol >= t.cols {
			t.curCol = 0
			t.curRow++
			if t.curRow >= t.rows {
				t.scrollUp()
				t.curRow = t.rows - 1
			}
		}
	}
}

func (t *Terminal) scrollUp() {
	if t.rows <= 1 {
		return
	}
	copy(t.buf, t.buf[1:])
	t.buf[t.rows-1] = make([]termCell, t.cols)
	for j := range t.buf[t.rows-1] {
		t.buf[t.rows-1][j] = termCell{Ch: ' ', Style: t.curStyle}
	}
}

func (t *Terminal) handleCSI(final byte) {
	params := parseCSIParams(t.csiBuf)

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
			t.curStyle = style.Style{FG: tcell.ColorWhite, BG: tcell.ColorDefault}
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
			t.curStyle.FG = tcell.ColorWhite
		case p >= 40 && p <= 47: // BG color
			t.curStyle.BG = ansi4Color(p - 40)
		case p == 48: // Extended BG
			i += t.parseExtendedColor(params[i:], false)
		case p == 49: // Default BG
			t.curStyle.BG = tcell.ColorDefault
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
	if t.ptmx == nil {
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
		data = []byte("\x1b[A")
	case tcell.KeyDown:
		data = []byte("\x1b[B")
	case tcell.KeyRight:
		data = []byte("\x1b[C")
	case tcell.KeyLeft:
		data = []byte("\x1b[D")
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
	if st.Border != style.BorderNone {
		r.DrawBorder(x, y, w, h, st.Border, st)
	}

	ix, iy, iw, ih := st.InnerRect(x, y, w, h)
	if iw <= 0 || ih <= 0 {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Resize if needed
	if iw != t.cols || ih != t.rows {
		t.Resize(ih, iw)
	}

	// Blit buffer
	for row := 0; row < t.rows && row < ih; row++ {
		for col := 0; col < t.cols && col < iw; col++ {
			cell := t.buf[row][col]
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
	if t.ptmx != nil {
		_ = t.ptmx.Close()
	}
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
