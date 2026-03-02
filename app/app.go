package app

import (
	"github.com/gdamore/tcell/v2"
	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/signal"
	"github.com/satyam/reactive-tui/widget"
)

type App struct {
	Root       widget.Node
	screen     tcell.Screen
	renderer   *render.Renderer
	focusIndex int
	focusables []widget.Node
	rerender   chan struct{}
}

func New(root widget.Node) *App {
	return &App{
		Root:     root,
		rerender: make(chan struct{}, 1),
	}
}

func (a *App) Run() error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()

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
	a.Root.Render(a.renderer, 0, 0, w, h)
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

		// Tab to cycle focus
		if e.Key() == tcell.KeyTab || e.Key() == tcell.KeyBacktab {
			a.cycleFocus(e.Key() == tcell.KeyBacktab)
			return false
		}

		// Forward to focused widget
		if len(a.focusables) > 0 && a.focusIndex < len(a.focusables) {
			kev := widget.KeyEvent{Key: int(e.Key()), Rune: e.Rune()}
			a.focusables[a.focusIndex].HandleKey(kev)
		}
	}
	return false
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
	a.focusables = nil
	collectFocusable(a.Root, &a.focusables)
	// Ensure focusIndex is valid
	if a.focusIndex >= len(a.focusables) {
		a.focusIndex = 0
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
