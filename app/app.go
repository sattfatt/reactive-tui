package app

import (
	"math"

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

		// Directional focus: Ctrl+H/J/K/L
		switch e.Key() {
		case tcell.KeyCtrlH:
			a.focusDirection(-1, 0)
			return false
		case tcell.KeyCtrlJ:
			a.focusDirection(0, 1)
			return false
		case tcell.KeyCtrlK:
			a.focusDirection(0, -1)
			return false
		case tcell.KeyCtrlL:
			a.focusDirection(1, 0)
			return false
		}

		// Forward to focused widget
		if len(a.focusables) > 0 && a.focusIndex < len(a.focusables) {
			kev := widget.KeyEvent{Key: int(e.Key()), Rune: e.Rune(), Mod: int(e.Modifiers())}
			a.focusables[a.focusIndex].HandleKey(kev)
		}

		// Recollect focusables in case the tree changed (e.g., tab switch)
		a.collectFocusables()
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
	var currentFocused widget.Node
	if a.focusIndex < len(a.focusables) {
		currentFocused = a.focusables[a.focusIndex]
	}

	a.focusables = nil
	collectFocusable(a.Root, &a.focusables)

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
