package app

import (
	"math"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/signal"
	"github.com/satyam/reactive-tui/style"
	"github.com/satyam/reactive-tui/widget"
)

// AppMode determines how key events are routed.
type AppMode int

const (
	ModeNavigation AppMode = iota // hjkl moves focus, i enters edit
	ModeEdit                      // all input goes to widget, Esc exits
)

type App struct {
	Root       widget.Node
	overlay    widget.Node // when set, rendered full-screen instead of Root
	screen     tcell.Screen
	renderer   *render.Renderer
	focusIndex int
	focusables []widget.Node
	rerender   chan struct{}
	mode       AppMode
	escCount   int       // consecutive Escape presses
	lastEscAt  time.Time // time of last Escape press
}

func New(root widget.Node) *App {
	return &App{
		Root:     root,
		rerender: make(chan struct{}, 1),
		mode:     ModeNavigation,
	}
}

// Mode returns the current app mode.
func (a *App) Mode() AppMode { return a.mode }

func (a *App) Run() error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()

	screen.SetStyle(tcell.StyleDefault.
		Foreground(style.CurrentTheme.FG).
		Background(style.CurrentTheme.BG))
	screen.EnableMouse()
	screen.Clear()

	a.screen = screen
	a.renderer = render.New(screen)

	// Wire signal changes to re-render
	signal.OnChange = func() {
		select {
		case a.rerender <- struct{}{}:
		default:
		}
	}

	// Initial render
	a.collectFocusables()
	if len(a.focusables) > 0 {
		a.focusables[0].SetFocused(true)
	}
	a.render()

	// Event loop
	eventCh := make(chan tcell.Event, 32)
	go func() {
		for {
			ev := screen.PollEvent()
			if ev == nil {
				return
			}
			eventCh <- ev
		}
	}()

	for {
		select {
		case ev := <-eventCh:
			if a.handleEvent(ev) {
				return nil // quit
			}
			a.render()
		case <-a.rerender:
			a.collectFocusables()
			a.render()
		}
	}
}

func (a *App) render() {
	a.renderer.Clear()
	w, h := a.renderer.Size()
	if a.overlay != nil {
		a.overlay.Render(a.renderer, 0, 0, w, h)
	} else {
		a.Root.Render(a.renderer, 0, 0, w, h)
	}
	a.renderer.Show()
}

func (a *App) handleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventResize:
		a.screen.Sync()
		return false
	case *tcell.EventKey:
		// Global quit: Ctrl+C or Ctrl+Q
		if e.Key() == tcell.KeyCtrlC || e.Key() == tcell.KeyCtrlQ {
			return true
		}

		// Ctrl+N → Down, Ctrl+P → Up
		if e.Key() == tcell.KeyCtrlN {
			e = tcell.NewEventKey(tcell.KeyDown, 0, e.Modifiers()&^tcell.ModCtrl)
		} else if e.Key() == tcell.KeyCtrlP {
			e = tcell.NewEventKey(tcell.KeyUp, 0, e.Modifiers()&^tcell.ModCtrl)
		}

		switch a.mode {
		case ModeNavigation:
			return a.handleNavMode(e)
		case ModeEdit:
			return a.handleEditMode(e)
		}
	}
	return false
}

func (a *App) handleNavMode(e *tcell.EventKey) bool {
	// 'i' enters edit mode
	if e.Key() == tcell.KeyRune && e.Rune() == 'i' {
		if len(a.focusables) > 0 && a.focusIndex < len(a.focusables) {
			if _, ok := a.focusables[a.focusIndex].(widget.Editable); ok {
				a.setEditing(true)
			}
		}
		return false
	}

	// Tab/Shift+Tab to cycle focus
	if e.Key() == tcell.KeyTab || e.Key() == tcell.KeyBacktab {
		a.cycleFocus(e.Key() == tcell.KeyBacktab)
		return false
	}

	// hjkl: offer to widget first, fall back to directional focus
	if e.Key() == tcell.KeyRune {
		dx, dy := 0, 0
		switch e.Rune() {
		case 'h':
			dx = -1
		case 'j':
			dy = 1
		case 'k':
			dy = -1
		case 'l':
			dx = 1
		}
		if dx != 0 || dy != 0 {
			consumed := false
			if len(a.focusables) > 0 && a.focusIndex < len(a.focusables) {
				kev := widget.KeyEvent{Key: int(e.Key()), Rune: e.Rune(), Mod: int(e.Modifiers())}
				consumed = a.focusables[a.focusIndex].HandleKey(kev)
			}
			if !consumed {
				a.focusDirection(dx, dy)
			}
			a.collectFocusables()
			return false
		}
	}

	// Forward all other keys to focused widget (Enter, arrows, etc.)
	if len(a.focusables) > 0 && a.focusIndex < len(a.focusables) {
		kev := widget.KeyEvent{Key: int(e.Key()), Rune: e.Rune(), Mod: int(e.Modifiers())}
		a.focusables[a.focusIndex].HandleKey(kev)
	}
	a.collectFocusables()
	return false
}

func (a *App) handleEditMode(e *tcell.EventKey) bool {
	// Determine how many consecutive Escapes are needed to exit edit mode.
	// Terminal needs 3 (since Escape is used heavily inside it),
	// other widgets need 1.
	escThreshold := 1
	if len(a.focusables) > 0 && a.focusIndex < len(a.focusables) {
		if et, ok := a.focusables[a.focusIndex].(widget.EscapeThreshold); ok {
			escThreshold = et.EscapesToExit()
		}
	}

	// Escape handling: count consecutive presses within 500ms
	if e.Key() == tcell.KeyEscape {
		now := time.Now()
		if now.Sub(a.lastEscAt) < 500*time.Millisecond {
			a.escCount++
		} else {
			a.escCount = 1
		}
		a.lastEscAt = now

		if a.escCount >= escThreshold {
			a.setEditing(false)
			return false
		}

		// For single-escape widgets, we already exited above.
		// For multi-escape widgets, forward the escape to the widget.
		if escThreshold > 1 {
			if len(a.focusables) > 0 && a.focusIndex < len(a.focusables) {
				kev := widget.KeyEvent{Key: int(e.Key()), Rune: e.Rune(), Mod: int(e.Modifiers())}
				a.focusables[a.focusIndex].HandleKey(kev)
			}
		}
		return false
	}

	// Non-escape key resets the counter
	a.escCount = 0

	// Forward everything to focused widget
	if len(a.focusables) > 0 && a.focusIndex < len(a.focusables) {
		kev := widget.KeyEvent{Key: int(e.Key()), Rune: e.Rune(), Mod: int(e.Modifiers())}
		a.focusables[a.focusIndex].HandleKey(kev)
	}

	// Recollect focusables in case the tree changed (e.g., tab switch)
	a.collectFocusables()
	return false
}

type editingSetter interface {
	SetEditing(bool)
}

func (a *App) setEditing(editing bool) {
	if editing {
		a.mode = ModeEdit
		a.escCount = 0
	} else {
		a.mode = ModeNavigation
		a.escCount = 0
	}
	if len(a.focusables) > 0 && a.focusIndex < len(a.focusables) {
		if es, ok := a.focusables[a.focusIndex].(editingSetter); ok {
			es.SetEditing(editing)
		}
	}
}

func (a *App) cycleFocus(reverse bool) {
	if len(a.focusables) == 0 {
		return
	}
	a.focusables[a.focusIndex].SetFocused(false)
	if reverse {
		a.focusIndex--
		if a.focusIndex < 0 {
			a.focusIndex = len(a.focusables) - 1
		}
	} else {
		a.focusIndex++
		if a.focusIndex >= len(a.focusables) {
			a.focusIndex = 0
		}
	}
	a.focusables[a.focusIndex].SetFocused(true)
}

func (a *App) collectFocusables() {
	var currentFocused widget.Node
	if a.focusIndex < len(a.focusables) {
		currentFocused = a.focusables[a.focusIndex]
	}

	a.focusables = nil
	root := a.Root
	if a.overlay != nil {
		root = a.overlay
	}
	collectFocusable(root, &a.focusables)

	// Try to preserve focus on the same widget by pointer identity
	a.focusIndex = 0
	if currentFocused != nil {
		for i, n := range a.focusables {
			if n == currentFocused {
				a.focusIndex = i
				return
			}
		}
	}
	if a.focusIndex >= len(a.focusables) {
		a.focusIndex = 0
	}
}

type rectGetter interface {
	GetRect() widget.Rect
}

func (a *App) focusDirection(dx, dy int) {
	if len(a.focusables) <= 1 {
		return
	}

	current, ok := a.focusables[a.focusIndex].(rectGetter)
	if !ok {
		return
	}
	cr := current.GetRect()
	cx := cr.X + cr.W/2
	cy := cr.Y + cr.H/2

	bestIdx := -1
	bestDist := math.MaxFloat64

	for i, node := range a.focusables {
		if i == a.focusIndex {
			continue
		}
		rg, ok := node.(rectGetter)
		if !ok {
			continue
		}
		nr := rg.GetRect()
		nx := nr.X + nr.W/2
		ny := nr.Y + nr.H/2

		relX := nx - cx
		relY := ny - cy

		// Candidate must be in the correct directional half-plane
		dot := relX*dx + relY*dy
		if dot <= 0 {
			continue
		}

		dist := math.Sqrt(float64(relX*relX + relY*relY))
		if dist < bestDist {
			bestDist = dist
			bestIdx = i
		}
	}

	if bestIdx >= 0 {
		a.focusables[a.focusIndex].SetFocused(false)
		a.focusIndex = bestIdx
		a.focusables[a.focusIndex].SetFocused(true)
	}
}

func collectFocusable(node widget.Node, out *[]widget.Node) {
	if node.Focusable() {
		*out = append(*out, node)
	}
	for _, child := range node.Children() {
		collectFocusable(child, out)
	}
}

// SetMode changes the app mode (Navigation or Edit).
func (a *App) SetMode(m AppMode) {
	a.mode = m
	a.escCount = 0
}

// OpenNeovimPopup opens a full-screen neovim editor overlay.
// When neovim exits, onDone is called with the edited content and the overlay is dismissed.
func (a *App) OpenNeovimPopup(content, filetype string, onDone func(string)) {
	editor := widget.NewNeovimEditor(filetype, nil)
	editor.SetText(content)
	editor.ZenMode = true
	editor.Style.Border = style.BorderNone // full screen, no border

	editor.OnChange = func(s string) {
		if onDone != nil {
			onDone(s)
		}
		// Dismiss overlay and restore nav mode
		a.overlay = nil
		a.SetMode(ModeNavigation)
	}

	a.overlay = editor
	a.collectFocusables()
	if len(a.focusables) > 0 {
		a.focusables[0].SetFocused(true)
	}
	a.SetMode(ModeEdit)

	// SetEditing triggers neovim Start()
	editor.SetEditing(true)

	if signal.OnChange != nil {
		signal.OnChange()
	}
}
