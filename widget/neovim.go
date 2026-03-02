package widget

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/signal"
	"github.com/satyam/reactive-tui/style"
)

// NeovimEditor embeds neovim inside a terminal widget as a rich text editor.
// When neovim is not running it shows a read-only preview of the content.
// Pressing 'i' (entering edit mode) launches neovim with the user's full config.
// When neovim exits (:wq, ZZ, :q!) the buffer content is read back via a temp file.
type NeovimEditor struct {
	Base
	term     *Terminal // nil when neovim is not running
	content  string
	filetype string // e.g. "go", "python", "markdown"
	tmpFile  string
	tmpDir   string
	running  bool
	mu       sync.Mutex

	// OnChange is called with the file content when neovim exits after writing.
	OnChange func(string)

	// ZenMode hides line numbers, statusline, and signcolumn in neovim.
	ZenMode bool

	// scrollY for preview mode
	scrollY int
}

// NewNeovimEditor creates a new NeovimEditor for the given filetype.
// The filetype is used as the temp file extension so neovim/LSP detect it automatically.
func NewNeovimEditor(filetype string, onChange func(string)) *NeovimEditor {
	label := filetype
	if label == "" {
		label = "nvim"
	}
	return &NeovimEditor{
		Base: Base{
			Style: style.Style{FG: style.CurrentTheme.FG, BG: style.CurrentTheme.BG, Border: style.BorderSingle},
			Flex:  FlexProps{Basis: -1, Grow: 1, Shrink: 1, MinHeight: 10, MinWidth: 20},
			Label: label,
		},
		filetype: filetype,
		OnChange: onChange,
	}
}

func (ne *NeovimEditor) Focusable() bool    { return true }
func (ne *NeovimEditor) IsEditable() bool   { return true }
func (ne *NeovimEditor) EscapesToExit() int { return 3 }

// Text returns the current content.
func (ne *NeovimEditor) Text() string {
	ne.mu.Lock()
	defer ne.mu.Unlock()
	return ne.content
}

// SetText sets the content. Takes effect on the next Start() (not while neovim is running).
func (ne *NeovimEditor) SetText(s string) {
	ne.mu.Lock()
	defer ne.mu.Unlock()
	ne.content = s
}

// Running returns whether neovim is currently active.
func (ne *NeovimEditor) Running() bool {
	ne.mu.Lock()
	defer ne.mu.Unlock()
	return ne.running
}

// SetEditing overrides Base to auto-start neovim when entering edit mode.
func (ne *NeovimEditor) SetEditing(editing bool) {
	ne.Base.SetEditing(editing)
	if editing {
		ne.mu.Lock()
		running := ne.running
		ne.mu.Unlock()
		if !running {
			ne.Start()
		}
	}
}

// Start launches neovim with the current content in a temp file.
func (ne *NeovimEditor) Start() {
	ne.mu.Lock()
	defer ne.mu.Unlock()

	if ne.running {
		return
	}

	if err := ne.createTempFile(); err != nil {
		return
	}

	// Build nvim arguments
	var args []string

	// Zen mode: hide UI chrome
	if ne.ZenMode {
		args = append(args, "-c", "set laststatus=0 | set nonumber | set norelativenumber | set signcolumn=no")
	}

	// Set filetype explicitly as backup (extension handles most cases)
	if ne.filetype != "" {
		args = append(args, "-c", "set filetype="+ne.filetype)
	}

	args = append(args, ne.tmpFile)

	// Create a fresh terminal for this session
	ne.term = NewTerminal("nvim", args...)
	ne.term.Start()
	ne.running = true

	go ne.waitForExit()
}

// waitForExit blocks until neovim exits, then reads the temp file content back.
func (ne *NeovimEditor) waitForExit() {
	// Block until the terminal's readLoop exits (neovim process ended)
	<-ne.term.done

	ne.mu.Lock()
	newContent, err := ne.readTempFile()
	if err == nil {
		ne.content = newContent
	}
	ne.running = false
	ne.cleanupTempFile()
	ne.mu.Unlock()

	// Fire callback outside the lock
	if err == nil && ne.OnChange != nil {
		ne.OnChange(newContent)
	}

	// Trigger re-render so the preview shows
	if signal.OnChange != nil {
		signal.OnChange()
	}
}

func (ne *NeovimEditor) createTempFile() error {
	dir, err := os.MkdirTemp("", "reactive-tui-nvim-*")
	if err != nil {
		return err
	}
	ne.tmpDir = dir

	ext := ne.filetype
	if ext == "" {
		ext = "txt"
	}
	path := filepath.Join(dir, "edit."+ext)

	if err := os.WriteFile(path, []byte(ne.content), 0600); err != nil {
		os.RemoveAll(dir)
		ne.tmpDir = ""
		return err
	}
	ne.tmpFile = path
	return nil
}

func (ne *NeovimEditor) readTempFile() (string, error) {
	data, err := os.ReadFile(ne.tmpFile)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (ne *NeovimEditor) cleanupTempFile() {
	if ne.tmpDir != "" {
		os.RemoveAll(ne.tmpDir)
		ne.tmpDir = ""
		ne.tmpFile = ""
	}
}

// Stop cleans up the terminal and temp files.
func (ne *NeovimEditor) Stop() {
	ne.mu.Lock()
	defer ne.mu.Unlock()
	if ne.term != nil && ne.running {
		ne.term.Stop()
	}
	ne.cleanupTempFile()
}

// HandleKey delegates to the inner terminal when neovim is running.
func (ne *NeovimEditor) HandleKey(ev KeyEvent) bool {
	ne.mu.Lock()
	running := ne.running
	term := ne.term
	ne.mu.Unlock()

	if running && term != nil {
		return term.HandleKey(ev)
	}
	return false
}

// Render draws the neovim terminal when running, or a content preview when idle.
func (ne *NeovimEditor) Render(r *render.Renderer, x, y, w, h int) {
	ne.Base.SetRect(x, y, w, h)

	ne.mu.Lock()
	running := ne.running
	term := ne.term
	content := ne.content
	ne.mu.Unlock()

	if running && term != nil {
		// Propagate focus/editing state to inner terminal
		term.Focused = ne.Focused
		term.Editing = ne.Editing
		term.Style = ne.Style
		term.Render(r, x, y, w, h)
		return
	}

	// Preview mode: render content as static text with themed border
	st := ne.Style

	borderSt := st
	if ne.Focused {
		borderSt.FG = style.CurrentTheme.NavFocusFG
	} else {
		borderSt.FG = style.CurrentTheme.BorderFG
	}
	if st.Border != style.BorderNone {
		r.DrawBorder(x, y, w, h, st.Border, borderSt)
		ne.Base.RenderLabel(r, x, y, w)
	}

	ix, iy, iw, ih := st.InnerRect(x, y, w, h)
	if iw <= 0 || ih <= 0 {
		return
	}

	// Show content or placeholder
	var lines []string
	placeholderMode := false
	if content == "" {
		lines = []string{"(press i to open neovim)"}
		placeholderMode = true
	} else {
		lines = strings.Split(content, "\n")
	}

	// Clamp scroll
	if ne.scrollY > len(lines)-ih {
		ne.scrollY = len(lines) - ih
	}
	if ne.scrollY < 0 {
		ne.scrollY = 0
	}

	drawSt := st
	if placeholderMode {
		drawSt.FG = style.CurrentTheme.MutedFG
	}

	for row := range ih {
		lineIdx := ne.scrollY + row
		if lineIdx >= len(lines) {
			break
		}
		line := lines[lineIdx]
		if len(line) > iw {
			line = line[:iw]
		}
		r.DrawText(ix, iy+row, line, drawSt, iw)
	}
}
