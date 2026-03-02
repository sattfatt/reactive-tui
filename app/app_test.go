package app

import (
	"testing"

	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/style"
	"github.com/satyam/reactive-tui/widget"

	"github.com/gdamore/tcell/v2"
)

func newTestApp(root widget.Node) (*App, tcell.SimulationScreen) {
	s := tcell.NewSimulationScreen("")
	_ = s.Init()
	s.SetSize(80, 24)

	a := New(root)
	a.screen = s
	a.renderer = render.New(s)
	return a, s
}

func TestNew(t *testing.T) {
	txt := widget.StaticText("hi")
	a := New(txt)
	if a.Root != txt {
		t.Error("Root should be set")
	}
	if a.rerender == nil {
		t.Error("rerender channel should be initialized")
	}
}

func TestCollectFocusables(t *testing.T) {
	btn1 := widget.NewButton("A", nil)
	btn2 := widget.NewButton("B", nil)
	txt := widget.StaticText("text")

	root := widget.VBox(btn1, txt, btn2)

	a, s := newTestApp(root)
	defer s.Fini()

	a.collectFocusables()

	if len(a.focusables) != 2 {
		t.Errorf("expected 2 focusables, got %d", len(a.focusables))
	}
}

func TestCollectFocusablesNested(t *testing.T) {
	btn := widget.NewButton("A", nil)
	inp := widget.NewInput("", nil)

	root := widget.VBox(
		widget.HBox(btn),
		widget.VBox(inp),
	)

	a, s := newTestApp(root)
	defer s.Fini()

	a.collectFocusables()

	if len(a.focusables) != 2 {
		t.Errorf("expected 2 focusables, got %d", len(a.focusables))
	}
}

func TestCollectFocusablesNone(t *testing.T) {
	root := widget.VBox(widget.StaticText("a"), widget.StaticText("b"))

	a, s := newTestApp(root)
	defer s.Fini()

	a.collectFocusables()

	if len(a.focusables) != 0 {
		t.Errorf("expected 0 focusables, got %d", len(a.focusables))
	}
}

func TestCycleFocusForward(t *testing.T) {
	btn1 := widget.NewButton("A", nil)
	btn2 := widget.NewButton("B", nil)
	btn3 := widget.NewButton("C", nil)

	root := widget.VBox(btn1, btn2, btn3)

	a, s := newTestApp(root)
	defer s.Fini()

	a.collectFocusables()
	a.focusables[0].SetFocused(true)

	// Cycle forward
	a.cycleFocus(false)
	if a.focusIndex != 1 {
		t.Errorf("expected focusIndex=1, got %d", a.focusIndex)
	}
	if !btn2.Focused {
		t.Error("btn2 should be focused")
	}
	if btn1.Focused {
		t.Error("btn1 should not be focused")
	}

	// Cycle to end and wrap
	a.cycleFocus(false)
	a.cycleFocus(false)
	if a.focusIndex != 0 {
		t.Errorf("expected wrap to 0, got %d", a.focusIndex)
	}
}

func TestCycleFocusReverse(t *testing.T) {
	btn1 := widget.NewButton("A", nil)
	btn2 := widget.NewButton("B", nil)

	root := widget.VBox(btn1, btn2)

	a, s := newTestApp(root)
	defer s.Fini()

	a.collectFocusables()
	a.focusables[0].SetFocused(true)

	// Reverse from 0 should wrap to last
	a.cycleFocus(true)
	if a.focusIndex != 1 {
		t.Errorf("expected wrap to 1, got %d", a.focusIndex)
	}
	if !btn2.Focused {
		t.Error("btn2 should be focused after reverse wrap")
	}
}

func TestCycleFocusEmpty(t *testing.T) {
	root := widget.VBox(widget.StaticText("a"))

	a, s := newTestApp(root)
	defer s.Fini()

	a.collectFocusables()
	// Should not panic with no focusables
	a.cycleFocus(false)
	a.cycleFocus(true)
}

func TestHandleEventQuit(t *testing.T) {
	root := widget.StaticText("hi")
	a, s := newTestApp(root)
	defer s.Fini()

	ev := tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)
	if !a.handleEvent(ev) {
		t.Error("Ctrl+C should return true (quit)")
	}

	ev = tcell.NewEventKey(tcell.KeyCtrlQ, 0, tcell.ModNone)
	if !a.handleEvent(ev) {
		t.Error("Ctrl+Q should return true (quit)")
	}
}

func TestHandleEventTab(t *testing.T) {
	btn1 := widget.NewButton("A", nil)
	btn2 := widget.NewButton("B", nil)
	root := widget.VBox(btn1, btn2)

	a, s := newTestApp(root)
	defer s.Fini()

	a.collectFocusables()
	a.focusables[0].SetFocused(true)

	ev := tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
	quit := a.handleEvent(ev)
	if quit {
		t.Error("Tab should not quit")
	}
	if a.focusIndex != 1 {
		t.Errorf("Tab should advance focus, got %d", a.focusIndex)
	}
}

func TestHandleEventBacktab(t *testing.T) {
	btn1 := widget.NewButton("A", nil)
	btn2 := widget.NewButton("B", nil)
	root := widget.VBox(btn1, btn2)

	a, s := newTestApp(root)
	defer s.Fini()

	a.collectFocusables()
	a.focusables[0].SetFocused(true)

	ev := tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
	a.handleEvent(ev)
	if a.focusIndex != 1 {
		t.Errorf("Backtab should wrap to last, got %d", a.focusIndex)
	}
}

func TestHandleEventForwardsToFocused(t *testing.T) {
	clicked := false
	btn := widget.NewButton("OK", func() { clicked = true })
	root := widget.VBox(btn)

	a, s := newTestApp(root)
	defer s.Fini()

	a.collectFocusables()
	a.focusables[0].SetFocused(true)

	ev := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	a.handleEvent(ev)

	if !clicked {
		t.Error("Enter should have triggered button click")
	}
}

func TestHandleEventResize(t *testing.T) {
	root := widget.StaticText("hi")
	a, s := newTestApp(root)
	defer s.Fini()

	ev := tcell.NewEventResize(100, 50)
	quit := a.handleEvent(ev)
	if quit {
		t.Error("resize should not quit")
	}
}

func TestHandleEventNonKeyNonResize(t *testing.T) {
	root := widget.StaticText("hi")
	a, s := newTestApp(root)
	defer s.Fini()

	// Mouse event should not quit
	ev := tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone)
	quit := a.handleEvent(ev)
	if quit {
		t.Error("mouse event should not quit")
	}
}

func TestRender(t *testing.T) {
	txt := widget.StaticText("hello")
	a, s := newTestApp(txt)
	defer s.Fini()

	// Should not panic
	a.render()

	r, _, _, _ := s.GetContent(0, 0)
	if r != 'h' {
		t.Errorf("expected 'h' at (0,0), got %c", r)
	}
}

func TestRenderWithBox(t *testing.T) {
	root := &widget.Box{
		Base: widget.Base{
			Style: style.Style{Border: style.BorderRounded},
		},
		Dir:   widget.Column,
		Items: []widget.Node{widget.StaticText("inside")},
	}

	a, s := newTestApp(root)
	defer s.Fini()

	a.render()

	r, _, _, _ := s.GetContent(0, 0)
	if r != '╭' {
		t.Errorf("expected ╭, got %c", r)
	}
}

// --- Stale focus fix tests ---

func TestTabsSwitchRecollectsFocusables(t *testing.T) {
	btn1 := widget.NewButton("Tab1Btn", nil)
	btn2 := widget.NewButton("Tab2Btn", nil)

	tabs := widget.NewTabs(
		widget.Tab{Label: "One", Content: btn1},
		widget.Tab{Label: "Two", Content: btn2},
	)
	root := widget.VBox(tabs)

	a, s := newTestApp(root)
	defer s.Fini()

	a.collectFocusables()
	// Initially: tabs + btn1 are focusable
	hasTabs := false
	hasBtn1 := false
	for _, f := range a.focusables {
		if f == tabs {
			hasTabs = true
		}
		if f == btn1 {
			hasBtn1 = true
		}
	}
	if !hasTabs || !hasBtn1 {
		t.Errorf("expected tabs and btn1 in focusables, got %d items", len(a.focusables))
	}

	// Focus on tabs, switch to tab 2
	for i, f := range a.focusables {
		if f == tabs {
			a.focusIndex = i
			f.SetFocused(true)
			break
		}
	}

	// Simulate pressing Right to switch to tab 2
	ev := tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
	a.handleEvent(ev)

	// After handleEvent, focusables should be recollected
	hasBtn2 := false
	hasBtn1After := false
	for _, f := range a.focusables {
		if f == btn2 {
			hasBtn2 = true
		}
		if f == btn1 {
			hasBtn1After = true
		}
	}
	if !hasBtn2 {
		t.Error("btn2 should be in focusables after switching to tab 2")
	}
	if hasBtn1After {
		t.Error("btn1 should NOT be in focusables after switching to tab 2")
	}
}

func TestFocusPreservedAfterRecollect(t *testing.T) {
	btn1 := widget.NewButton("A", nil)
	btn2 := widget.NewButton("B", nil)
	root := widget.VBox(btn1, btn2)

	a, s := newTestApp(root)
	defer s.Fini()

	a.collectFocusables()
	a.focusIndex = 1
	a.focusables[1].SetFocused(true)

	// Recollect
	a.collectFocusables()

	// btn2 should still be focused (index preserved by pointer identity)
	if a.focusIndex != 1 {
		t.Errorf("expected focusIndex=1 preserved, got %d", a.focusIndex)
	}
	if a.focusables[a.focusIndex] != btn2 {
		t.Error("focus should still be on btn2")
	}
}

// --- Directional focus tests ---

func TestFocusDirectionHorizontal(t *testing.T) {
	btn1 := widget.NewButton("L", nil)
	btn1.Flex = widget.FlexProps{Basis: 10}
	btn2 := widget.NewButton("R", nil)
	btn2.Flex = widget.FlexProps{Basis: 10}

	root := widget.HBox(btn1, btn2)
	a, s := newTestApp(root)
	defer s.Fini()

	// Render to populate rects
	a.render()
	a.collectFocusables()
	a.focusIndex = 0
	a.focusables[0].SetFocused(true)

	// Move right
	a.focusDirection(1, 0)
	if a.focusIndex != 1 {
		t.Errorf("expected focus to move right to index 1, got %d", a.focusIndex)
	}

	// Move left
	a.focusDirection(-1, 0)
	if a.focusIndex != 0 {
		t.Errorf("expected focus to move left to index 0, got %d", a.focusIndex)
	}
}

func TestFocusDirectionVertical(t *testing.T) {
	btn1 := widget.NewButton("Top", nil)
	btn2 := widget.NewButton("Bot", nil)

	root := widget.VBox(btn1, btn2)
	a, s := newTestApp(root)
	defer s.Fini()

	a.render()
	a.collectFocusables()
	a.focusIndex = 0
	a.focusables[0].SetFocused(true)

	// Move down
	a.focusDirection(0, 1)
	if a.focusIndex != 1 {
		t.Errorf("expected focus to move down to index 1, got %d", a.focusIndex)
	}

	// Move up
	a.focusDirection(0, -1)
	if a.focusIndex != 0 {
		t.Errorf("expected focus to move up to index 0, got %d", a.focusIndex)
	}
}

func TestFocusDirectionNoWrap(t *testing.T) {
	btn1 := widget.NewButton("A", nil)
	btn2 := widget.NewButton("B", nil)

	root := widget.HBox(btn1, btn2)
	a, s := newTestApp(root)
	defer s.Fini()

	a.render()
	a.collectFocusables()
	a.focusIndex = 0
	a.focusables[0].SetFocused(true)

	// Move left from leftmost — should stay
	a.focusDirection(-1, 0)
	if a.focusIndex != 0 {
		t.Errorf("expected no wrap, focus should stay at 0, got %d", a.focusIndex)
	}
}

func TestFocusDirectionSingleWidget(t *testing.T) {
	btn := widget.NewButton("A", nil)
	root := widget.VBox(btn)
	a, s := newTestApp(root)
	defer s.Fini()

	a.render()
	a.collectFocusables()
	a.focusIndex = 0

	// Should not panic with single widget
	a.focusDirection(1, 0)
	if a.focusIndex != 0 {
		t.Errorf("expected index 0 with single widget, got %d", a.focusIndex)
	}
}

func TestFocusIndexClamped(t *testing.T) {
	btn := widget.NewButton("A", nil)
	root := widget.VBox(btn)

	a, s := newTestApp(root)
	defer s.Fini()

	a.focusIndex = 99
	a.collectFocusables()

	if a.focusIndex != 0 {
		t.Errorf("focus index should be clamped to 0, got %d", a.focusIndex)
	}
}
