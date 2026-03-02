package widget

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/style"
)

func newTestRenderer(w, h int) (*render.Renderer, tcell.SimulationScreen) {
	s := tcell.NewSimulationScreen("")
	_ = s.Init()
	s.SetSize(w, h)
	return render.New(s), s
}

func getCell(s tcell.SimulationScreen, x, y int) rune {
	r, _, _, _ := s.GetContent(x, y)
	return r
}

// --- Base tests ---

func TestBaseDefaults(t *testing.T) {
	b := &Base{}
	if b.Focusable() {
		t.Error("Base should not be focusable")
	}
	if b.Children() != nil {
		t.Error("Base should have nil children")
	}
	if b.HandleKey(KeyEvent{}) {
		t.Error("Base should not handle keys")
	}
}

func TestBaseSetFocused(t *testing.T) {
	b := &Base{}
	b.SetFocused(true)
	if !b.Focused {
		t.Error("expected focused=true")
	}
	b.SetFocused(false)
	if b.Focused {
		t.Error("expected focused=false")
	}
}

func TestBaseFlexProps(t *testing.T) {
	b := &Base{Flex: FlexProps{Grow: 2, Shrink: 0.5, Basis: 30}}
	fp := b.FlexProps()
	if fp.Grow != 2 || fp.Shrink != 0.5 || fp.Basis != 30 {
		t.Errorf("unexpected flex props: %+v", fp)
	}
}

func TestDefaultFlexProps(t *testing.T) {
	fp := DefaultFlexProps()
	if fp.Grow != 0 || fp.Shrink != 1 || fp.Basis != -1 {
		t.Errorf("unexpected default flex props: %+v", fp)
	}
}

// --- Text widget tests ---

func TestStaticText(t *testing.T) {
	txt := StaticText("hello")
	if txt.Content() != "hello" {
		t.Errorf("expected hello, got %s", txt.Content())
	}
}

func TestBoundText(t *testing.T) {
	val := "initial"
	txt := BoundText(func() string { return val })
	if txt.Content() != "initial" {
		t.Error("should return initial value")
	}
	val = "updated"
	if txt.Content() != "updated" {
		t.Error("should reflect updated value")
	}
}

func TestTextRender(t *testing.T) {
	r, s := newTestRenderer(80, 24)
	defer s.Fini()

	txt := StaticText("hi")
	txt.Render(r, 0, 0, 80, 1)

	if getCell(s, 0, 0) != 'h' || getCell(s, 1, 0) != 'i' {
		t.Error("text should render 'hi'")
	}
}

func TestTextRenderWithBorder(t *testing.T) {
	r, s := newTestRenderer(80, 24)
	defer s.Fini()

	txt := StaticText("X")
	txt.Style.Border = style.BorderSingle
	txt.Render(r, 0, 0, 10, 3)

	// Border at corners
	if getCell(s, 0, 0) != '┌' {
		t.Errorf("expected border corner, got %c", getCell(s, 0, 0))
	}
	// Text inside border
	if getCell(s, 1, 1) != 'X' {
		t.Errorf("expected X inside border, got %c", getCell(s, 1, 1))
	}
}

func TestTextNotFocusable(t *testing.T) {
	txt := StaticText("hi")
	if txt.Focusable() {
		t.Error("text should not be focusable")
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		text  string
		width int
		want  int // number of lines
	}{
		{"hello", 10, 1},
		{"hello world", 5, 2},
		{"a\nb", 10, 2},
		{"", 10, 1},
		{"hello", 0, 0},
	}
	for _, tt := range tests {
		lines := wrapText(tt.text, tt.width)
		if len(lines) != tt.want {
			t.Errorf("wrapText(%q, %d): expected %d lines, got %d: %v",
				tt.text, tt.width, tt.want, len(lines), lines)
		}
	}
}

// --- Button tests ---

func TestButtonFocusable(t *testing.T) {
	b := NewButton("OK", nil)
	if !b.Focusable() {
		t.Error("button should be focusable")
	}
}

func TestButtonHandleEnter(t *testing.T) {
	clicked := false
	b := NewButton("OK", func() { clicked = true })
	consumed := b.HandleKey(KeyEvent{Key: int(tcell.KeyEnter)})
	if !consumed {
		t.Error("button should consume Enter")
	}
	if !clicked {
		t.Error("OnClick should have been called")
	}
}

func TestButtonHandleSpace(t *testing.T) {
	clicked := false
	b := NewButton("OK", func() { clicked = true })
	consumed := b.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: ' '})
	if !consumed {
		t.Error("button should consume space")
	}
	if !clicked {
		t.Error("OnClick should have been called")
	}
}

func TestButtonIgnoresOtherKeys(t *testing.T) {
	b := NewButton("OK", func() {})
	consumed := b.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'a'})
	if consumed {
		t.Error("button should not consume 'a'")
	}
}

func TestButtonNilOnClick(t *testing.T) {
	b := NewButton("OK", nil)
	// Should not panic
	b.HandleKey(KeyEvent{Key: int(tcell.KeyEnter)})
}

func TestButtonRender(t *testing.T) {
	r, s := newTestRenderer(80, 24)
	defer s.Fini()

	b := NewButton("OK", nil)
	b.Render(r, 0, 0, 10, 3)

	// Should have a border
	if getCell(s, 0, 0) != '┌' {
		t.Errorf("expected border, got %c", getCell(s, 0, 0))
	}
}

func TestButtonFocusedRender(t *testing.T) {
	r, s := newTestRenderer(80, 24)
	defer s.Fini()

	b := NewButton("OK", nil)
	b.SetFocused(true)
	b.Render(r, 0, 0, 10, 3)

	// Check that it renders (colors inverted is hard to test, but we can check it doesn't panic)
	if getCell(s, 0, 0) != '┌' {
		t.Errorf("expected border on focused button, got %c", getCell(s, 0, 0))
	}
}

// --- Input tests ---

func TestInputFocusable(t *testing.T) {
	inp := NewInput("type here", nil)
	if !inp.Focusable() {
		t.Error("input should be focusable")
	}
}

func TestInputTypeRune(t *testing.T) {
	var lastValue string
	inp := NewInput("", func(v string) { lastValue = v })
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'a'})
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'b'})

	if inp.Value != "ab" {
		t.Errorf("expected 'ab', got %q", inp.Value)
	}
	if inp.Cursor != 2 {
		t.Errorf("expected cursor at 2, got %d", inp.Cursor)
	}
	if lastValue != "ab" {
		t.Errorf("OnChange should report 'ab', got %q", lastValue)
	}
}

func TestInputBackspace(t *testing.T) {
	inp := NewInput("", nil)
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'a'})
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'b'})
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyBackspace2)})

	if inp.Value != "a" {
		t.Errorf("expected 'a', got %q", inp.Value)
	}
	if inp.Cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", inp.Cursor)
	}
}

func TestInputBackspaceAtStart(t *testing.T) {
	inp := NewInput("", nil)
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyBackspace2)})
	if inp.Value != "" || inp.Cursor != 0 {
		t.Error("backspace at start should do nothing")
	}
}

func TestInputDelete(t *testing.T) {
	inp := NewInput("", nil)
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'a'})
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'b'})
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyLeft)})
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyDelete)})

	if inp.Value != "a" {
		t.Errorf("expected 'a', got %q", inp.Value)
	}
}

func TestInputCursorMovement(t *testing.T) {
	inp := NewInput("", nil)
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'a'})
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'b'})

	inp.HandleKey(KeyEvent{Key: int(tcell.KeyLeft)})
	if inp.Cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", inp.Cursor)
	}

	inp.HandleKey(KeyEvent{Key: int(tcell.KeyLeft)})
	if inp.Cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", inp.Cursor)
	}

	// Left at 0 should stay at 0
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyLeft)})
	if inp.Cursor != 0 {
		t.Errorf("cursor should stay at 0")
	}

	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRight)})
	if inp.Cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", inp.Cursor)
	}

	// Right past end should stay at end
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRight)})
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRight)})
	if inp.Cursor != 2 {
		t.Errorf("cursor should stay at 2")
	}
}

func TestInputInsertAtCursor(t *testing.T) {
	inp := NewInput("", nil)
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'a'})
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'c'})
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyLeft)})
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'b'})

	if inp.Value != "abc" {
		t.Errorf("expected 'abc', got %q", inp.Value)
	}
}

func TestInputRender(t *testing.T) {
	r, s := newTestRenderer(80, 24)
	defer s.Fini()

	inp := NewInput("hint", nil)
	inp.Render(r, 0, 0, 20, 3)

	// Should show placeholder when unfocused and empty
	if getCell(s, 1, 1) != 'h' {
		t.Errorf("expected placeholder 'h', got %c", getCell(s, 1, 1))
	}
}

func TestInputNilOnChange(t *testing.T) {
	inp := NewInput("", nil)
	// Should not panic
	inp.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'x'})
}

// --- List tests ---

func TestListFocusable(t *testing.T) {
	l := NewList(func() []string { return nil }, nil)
	if !l.Focusable() {
		t.Error("list should be focusable")
	}
}

func TestListNavigation(t *testing.T) {
	items := []string{"a", "b", "c"}
	l := NewList(func() []string { return items }, nil)

	l.HandleKey(KeyEvent{Key: int(tcell.KeyDown)})
	if l.Selected != 1 {
		t.Errorf("expected selected=1, got %d", l.Selected)
	}

	l.HandleKey(KeyEvent{Key: int(tcell.KeyDown)})
	if l.Selected != 2 {
		t.Errorf("expected selected=2, got %d", l.Selected)
	}

	// Can't go past end
	l.HandleKey(KeyEvent{Key: int(tcell.KeyDown)})
	if l.Selected != 2 {
		t.Error("should not go past last item")
	}

	l.HandleKey(KeyEvent{Key: int(tcell.KeyUp)})
	if l.Selected != 1 {
		t.Errorf("expected selected=1, got %d", l.Selected)
	}

	l.HandleKey(KeyEvent{Key: int(tcell.KeyUp)})
	l.HandleKey(KeyEvent{Key: int(tcell.KeyUp)})
	if l.Selected != 0 {
		t.Error("should not go below 0")
	}
}

func TestListSelect(t *testing.T) {
	selected := -1
	items := []string{"a", "b"}
	l := NewList(func() []string { return items }, func(i int) { selected = i })

	l.HandleKey(KeyEvent{Key: int(tcell.KeyDown)})
	l.HandleKey(KeyEvent{Key: int(tcell.KeyEnter)})

	if selected != 1 {
		t.Errorf("expected selected=1, got %d", selected)
	}
}

func TestListSelectNilCallback(t *testing.T) {
	items := []string{"a"}
	l := NewList(func() []string { return items }, nil)
	// Should not panic
	l.HandleKey(KeyEvent{Key: int(tcell.KeyEnter)})
}

func TestListRender(t *testing.T) {
	r, s := newTestRenderer(80, 24)
	defer s.Fini()

	items := []string{"alpha", "beta"}
	l := NewList(func() []string { return items }, nil)
	l.Render(r, 0, 0, 20, 5)

	if getCell(s, 0, 0) != 'a' {
		t.Errorf("expected 'a' at (0,0), got %c", getCell(s, 0, 0))
	}
	if getCell(s, 0, 1) != 'b' {
		t.Errorf("expected 'b' at (0,1), got %c", getCell(s, 0, 1))
	}
}

func TestListIgnoresOtherKeys(t *testing.T) {
	l := NewList(func() []string { return []string{"a"} }, nil)
	consumed := l.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'x'})
	if consumed {
		t.Error("list should not consume random runes")
	}
}

// --- Box tests ---

func TestVBoxConstructor(t *testing.T) {
	t1 := StaticText("a")
	t2 := StaticText("b")
	box := VBox(t1, t2)
	if box.Dir != Column {
		t.Error("VBox should have Column direction")
	}
	if len(box.Children()) != 2 {
		t.Errorf("expected 2 children, got %d", len(box.Children()))
	}
}

func TestHBoxConstructor(t *testing.T) {
	box := HBox(StaticText("a"))
	if box.Dir != Row {
		t.Error("HBox should have Row direction")
	}
}

func TestBoxRender(t *testing.T) {
	r, s := newTestRenderer(80, 24)
	defer s.Fini()

	box := VBox(
		StaticText("line1"),
		StaticText("line2"),
	)
	box.Render(r, 0, 0, 80, 24)

	// Both texts should be present (exact positioning depends on flex)
	_ = s // rendering shouldn't panic
}

func TestBoxWithBorder(t *testing.T) {
	r, s := newTestRenderer(80, 24)
	defer s.Fini()

	box := VBox(StaticText("x"))
	box.Style.Border = style.BorderRounded
	box.Render(r, 0, 0, 20, 10)

	if getCell(s, 0, 0) != '╭' {
		t.Errorf("expected rounded border, got %c", getCell(s, 0, 0))
	}
}

func TestBoxEmptyChildren(t *testing.T) {
	r, s := newTestRenderer(80, 24)
	defer s.Fini()

	box := VBox()
	// Should not panic
	box.Render(r, 0, 0, 80, 24)
	_ = s
}

func TestHBoxRender(t *testing.T) {
	r, s := newTestRenderer(80, 24)
	defer s.Fini()

	t1 := StaticText("A")
	t1.Flex = FlexProps{Basis: 5}
	t2 := StaticText("B")
	t2.Flex = FlexProps{Basis: 5}

	box := HBox(t1, t2)
	box.Render(r, 0, 0, 80, 1)

	if getCell(s, 0, 0) != 'A' {
		t.Errorf("expected 'A' at (0,0), got %c", getCell(s, 0, 0))
	}
	if getCell(s, 5, 0) != 'B' {
		t.Errorf("expected 'B' at (5,0), got %c", getCell(s, 5, 0))
	}
}
