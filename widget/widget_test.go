package widget

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/sattfatt/reactive-tui/render"
	"github.com/sattfatt/reactive-tui/style"
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
		lines := WrapText(tt.text, tt.width)
		if len(lines) != tt.want {
			t.Errorf("WrapText(%q, %d): expected %d lines, got %d: %v",
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
	b.Render(r, 0, 0, 10, 1)

	// Should have label with brackets: "[ OK ]" centered in 10 = at x=2
	if getCell(s, 2, 0) != '[' {
		t.Errorf("expected '[', got %c", getCell(s, 2, 0))
	}
}

func TestButtonFocusedRender(t *testing.T) {
	r, s := newTestRenderer(80, 24)
	defer s.Fini()

	b := NewButton("OK", nil)
	b.SetFocused(true)
	b.Render(r, 0, 0, 10, 1)

	// Should render without panic
	if getCell(s, 2, 0) != '[' {
		t.Errorf("expected '[' on focused button, got %c", getCell(s, 2, 0))
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
	inp.Editing = true
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
	inp.Editing = true
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
	inp.Editing = true
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
	inp.Editing = true
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
	inp.Editing = true
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
	inp.Editing = true
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

	// With auto-sizing, texts should each get 1 row
	if getCell(s, 0, 0) != 'l' {
		t.Errorf("expected 'l' at (0,0), got %c", getCell(s, 0, 0))
	}
	if getCell(s, 0, 1) != 'l' {
		t.Errorf("expected 'l' at (0,1), got %c", getCell(s, 0, 1))
	}
}

func TestVBoxAutoSizesChildren(t *testing.T) {
	r, s := newTestRenderer(80, 24)
	defer s.Fini()

	box := VBox(
		StaticText("AAA"),
		NewButton("B", nil),
		StaticText("CCC"),
	)
	box.Render(r, 0, 0, 80, 24)

	// Text gets 1 row, button gets 1 row (no border), text gets 1 row
	// Row 0: "AAA"
	if getCell(s, 0, 0) != 'A' {
		t.Errorf("expected 'A' at (0,0), got %c", getCell(s, 0, 0))
	}
	// Row 1: button label centered - "[ B ]" in 80 cols, left pad = 37
	if getCell(s, 37, 1) != '[' {
		t.Errorf("expected button label at (37,1), got %c", getCell(s, 37, 1))
	}
	// Row 2: "CCC"
	if getCell(s, 0, 2) != 'C' {
		t.Errorf("expected 'C' at (0,2), got %c", getCell(s, 0, 2))
	}
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

func TestBoxFlexPropsComputesMinHeight(t *testing.T) {
	// VBox with 2 text (1 row each) + 1 button (1 row) = 3 min height
	box := VBox(
		StaticText("a"),
		NewButton("b", nil),
		StaticText("c"),
	)
	fp := box.FlexProps()
	if fp.MinHeight < 3 {
		t.Errorf("expected MinHeight >= 3, got %d", fp.MinHeight)
	}
}

func TestBoxFlexPropsWithGap(t *testing.T) {
	box := VBox(StaticText("a"), StaticText("b"))
	box.Gap = 2
	fp := box.FlexProps()
	// 2 texts (1 each) + 1 gap * 2 = 4
	if fp.MinHeight < 4 {
		t.Errorf("expected MinHeight >= 4, got %d", fp.MinHeight)
	}
}

func TestHBoxFlexPropsComputesMinWidth(t *testing.T) {
	box := HBox(
		StaticText("a"),
		StaticText("b"),
	)
	fp := box.FlexProps()
	// 2 texts with MinWidth=1 each = 2
	if fp.MinWidth < 2 {
		t.Errorf("expected MinWidth >= 2, got %d", fp.MinWidth)
	}
	// Cross-axis: max child MinHeight = 1
	if fp.MinHeight < 1 {
		t.Errorf("expected MinHeight >= 1, got %d", fp.MinHeight)
	}
}

func TestBoxFlexPropsWithExplicitBasis(t *testing.T) {
	box := VBox(StaticText("a"))
	box.Flex.Basis = 10 // explicit, should skip auto-compute
	fp := box.FlexProps()
	if fp.Basis != 10 {
		t.Errorf("expected Basis=10, got %d", fp.Basis)
	}
}

func TestTextConstructorDefaults(t *testing.T) {
	txt := StaticText("hi")
	fp := txt.FlexProps()
	if fp.Basis != -1 {
		t.Errorf("expected Basis=-1, got %d", fp.Basis)
	}
	if fp.MinHeight != 1 {
		t.Errorf("expected MinHeight=1, got %d", fp.MinHeight)
	}
}

func TestButtonConstructorDefaults(t *testing.T) {
	b := NewButton("OK", nil)
	fp := b.FlexProps()
	if fp.Basis != -1 {
		t.Errorf("expected Basis=-1, got %d", fp.Basis)
	}
	if fp.MinHeight != 1 {
		t.Errorf("expected MinHeight=1, got %d", fp.MinHeight)
	}
}

func TestInputConstructorDefaults(t *testing.T) {
	inp := NewInput("", nil)
	fp := inp.FlexProps()
	if fp.Basis != -1 {
		t.Errorf("expected Basis=-1, got %d", fp.Basis)
	}
	if fp.MinHeight != 3 {
		t.Errorf("expected MinHeight=3, got %d", fp.MinHeight)
	}
}

func TestListConstructorDefaults(t *testing.T) {
	l := NewList(func() []string { return nil }, nil)
	fp := l.FlexProps()
	if fp.Grow != 1 {
		t.Errorf("expected Grow=1, got %f", fp.Grow)
	}
}

// --- ProgressBar tests ---

func TestProgressBarDefaults(t *testing.T) {
	pb := NewProgressBar(func() float64 { return 0.5 })
	fp := pb.FlexProps()
	if fp.Grow != 1 {
		t.Errorf("expected Grow=1, got %f", fp.Grow)
	}
	if fp.MaxHeight != 1 {
		t.Errorf("expected MaxHeight=1, got %d", fp.MaxHeight)
	}
	if pb.Focusable() {
		t.Error("progress bar should not be focusable")
	}
}

func TestProgressBarRenderEmpty(t *testing.T) {
	r, s := newTestRenderer(30, 1)
	defer s.Fini()

	pb := NewProgressBar(func() float64 { return 0 })
	pb.ShowLabel = false
	pb.Render(r, 0, 0, 30, 1)

	// All cells should be empty char
	if getCell(s, 0, 0) != '░' {
		t.Errorf("expected empty char, got %c", getCell(s, 0, 0))
	}
}

func TestProgressBarRenderFull(t *testing.T) {
	r, s := newTestRenderer(30, 1)
	defer s.Fini()

	pb := NewProgressBar(func() float64 { return 1.0 })
	pb.ShowLabel = false
	pb.Render(r, 0, 0, 30, 1)

	// First cell should be filled char
	if getCell(s, 0, 0) != '█' {
		t.Errorf("expected filled char, got %c", getCell(s, 0, 0))
	}
}

func TestProgressBarRenderWithLabel(t *testing.T) {
	r, s := newTestRenderer(30, 1)
	defer s.Fini()

	pb := NewProgressBar(func() float64 { return 0.5 })
	pb.Render(r, 0, 0, 30, 1)

	// Should have " 50%" somewhere near the end
	found := false
	for x := 20; x < 30; x++ {
		if getCell(s, x, 0) == '%' {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected label with % sign")
	}
}

func TestProgressBarClampsValues(t *testing.T) {
	r, s := newTestRenderer(20, 1)
	defer s.Fini()

	// Values outside 0-1 should be clamped
	pb := NewProgressBar(func() float64 { return 1.5 })
	pb.ShowLabel = false
	pb.Render(r, 0, 0, 20, 1)
	// All should be filled
	if getCell(s, 0, 0) != '█' {
		t.Errorf("expected filled, got %c", getCell(s, 0, 0))
	}

	r2, s2 := newTestRenderer(20, 1)
	defer s2.Fini()
	pb2 := NewProgressBar(func() float64 { return -0.5 })
	pb2.ShowLabel = false
	pb2.Render(r2, 0, 0, 20, 1)
	if getCell(s2, 0, 0) != '░' {
		t.Errorf("expected empty, got %c", getCell(s2, 0, 0))
	}
}

// --- TextArea tests ---

func TestTextAreaDefaults(t *testing.T) {
	ta := NewTextArea(nil)
	if !ta.Focusable() {
		t.Error("textarea should be focusable")
	}
	if ta.Text() != "" {
		t.Errorf("expected empty text, got %q", ta.Text())
	}
	fp := ta.FlexProps()
	if fp.MinHeight != 5 {
		t.Errorf("expected MinHeight=5, got %d", fp.MinHeight)
	}
}

func TestTextAreaSetText(t *testing.T) {
	ta := NewTextArea(nil)
	ta.SetText("hello\nworld")
	if ta.Text() != "hello\nworld" {
		t.Errorf("expected 'hello\\nworld', got %q", ta.Text())
	}
	if len(ta.Lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(ta.Lines))
	}
}

func TestTextAreaInsertRune(t *testing.T) {
	var changed string
	ta := NewTextArea(func(s string) { changed = s })
	ta.Editing = true
	ta.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'h'})
	ta.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'i'})
	if ta.Text() != "hi" {
		t.Errorf("expected 'hi', got %q", ta.Text())
	}
	if changed != "hi" {
		t.Errorf("OnChange should report 'hi', got %q", changed)
	}
}

func TestTextAreaEnterSplitsLine(t *testing.T) {
	ta := NewTextArea(nil)
	ta.Editing = true
	ta.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'a'})
	ta.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'b'})
	ta.HandleKey(KeyEvent{Key: int(tcell.KeyLeft)})
	ta.HandleKey(KeyEvent{Key: int(tcell.KeyEnter)})

	if len(ta.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(ta.Lines))
	}
	if ta.Lines[0] != "a" || ta.Lines[1] != "b" {
		t.Errorf("expected [a, b], got %v", ta.Lines)
	}
	if ta.CursorRow != 1 || ta.CursorCol != 0 {
		t.Errorf("expected cursor at (1,0), got (%d,%d)", ta.CursorRow, ta.CursorCol)
	}
}

func TestTextAreaBackspaceMergesLines(t *testing.T) {
	ta := NewTextArea(nil)
	ta.SetText("ab\ncd")
	ta.CursorRow = 1
	ta.CursorCol = 0
	ta.HandleKey(KeyEvent{Key: int(tcell.KeyBackspace2)})

	if len(ta.Lines) != 1 {
		t.Fatalf("expected 1 line after merge, got %d", len(ta.Lines))
	}
	if ta.Lines[0] != "abcd" {
		t.Errorf("expected 'abcd', got %q", ta.Lines[0])
	}
	if ta.CursorRow != 0 || ta.CursorCol != 2 {
		t.Errorf("expected cursor at (0,2), got (%d,%d)", ta.CursorRow, ta.CursorCol)
	}
}

func TestTextAreaDeleteMergesLines(t *testing.T) {
	ta := NewTextArea(nil)
	ta.SetText("ab\ncd")
	ta.CursorRow = 0
	ta.CursorCol = 2
	ta.HandleKey(KeyEvent{Key: int(tcell.KeyDelete)})

	if len(ta.Lines) != 1 || ta.Lines[0] != "abcd" {
		t.Errorf("expected 'abcd', got %v", ta.Lines)
	}
}

func TestTextAreaCursorMovement(t *testing.T) {
	ta := NewTextArea(nil)
	ta.SetText("abc\ndef")
	ta.Editing = true

	// Move down
	ta.CursorRow = 0
	ta.CursorCol = 2
	ta.HandleKey(KeyEvent{Key: int(tcell.KeyDown)})
	if ta.CursorRow != 1 || ta.CursorCol != 2 {
		t.Errorf("expected (1,2), got (%d,%d)", ta.CursorRow, ta.CursorCol)
	}

	// Move up
	ta.HandleKey(KeyEvent{Key: int(tcell.KeyUp)})
	if ta.CursorRow != 0 {
		t.Errorf("expected row 0, got %d", ta.CursorRow)
	}

	// Right wraps to next line
	ta.CursorRow = 0
	ta.CursorCol = 3
	ta.HandleKey(KeyEvent{Key: int(tcell.KeyRight)})
	if ta.CursorRow != 1 || ta.CursorCol != 0 {
		t.Errorf("expected (1,0), got (%d,%d)", ta.CursorRow, ta.CursorCol)
	}

	// Left wraps to previous line
	ta.HandleKey(KeyEvent{Key: int(tcell.KeyLeft)})
	if ta.CursorRow != 0 || ta.CursorCol != 3 {
		t.Errorf("expected (0,3), got (%d,%d)", ta.CursorRow, ta.CursorCol)
	}
}

func TestTextAreaRender(t *testing.T) {
	r, s := newTestRenderer(20, 10)
	defer s.Fini()

	ta := NewTextArea(nil)
	ta.SetText("line1\nline2")
	ta.Render(r, 0, 0, 20, 10)

	// Border at (0,0), text starts at (1,1)
	if getCell(s, 1, 1) != 'l' {
		t.Errorf("expected 'l' at (1,1), got %c", getCell(s, 1, 1))
	}
}

// --- Table tests ---

func TestTableDefaults(t *testing.T) {
	cols := []TableColumn{{Header: "Name"}, {Header: "Age"}}
	tbl := NewTable(cols, func() [][]string { return nil }, nil)
	if !tbl.Focusable() {
		t.Error("table should be focusable")
	}
	fp := tbl.FlexProps()
	if fp.MinHeight != 4 {
		t.Errorf("expected MinHeight=4, got %d", fp.MinHeight)
	}
}

func TestTableColumnWidths(t *testing.T) {
	cols := []TableColumn{
		{Header: "ID", Width: 5},
		{Header: "Name", Grow: 2},
		{Header: "Age", Grow: 1},
	}
	tbl := NewTable(cols, func() [][]string { return nil }, nil)
	widths := tbl.resolveColumnWidths(30)

	if widths[0] != 5 {
		t.Errorf("expected fixed col width=5, got %d", widths[0])
	}
	// Remaining: 30 - 5 - 2(separators) = 23, split 2:1
	if widths[1] < widths[2] {
		t.Errorf("grow=2 col should be wider than grow=1, got %d vs %d", widths[1], widths[2])
	}
}

func TestTableNavigation(t *testing.T) {
	rows := [][]string{{"a"}, {"b"}, {"c"}}
	tbl := NewTable(nil, func() [][]string { return rows }, nil)

	tbl.HandleKey(KeyEvent{Key: int(tcell.KeyDown)})
	if tbl.Selected != 1 {
		t.Errorf("expected 1, got %d", tbl.Selected)
	}
	tbl.HandleKey(KeyEvent{Key: int(tcell.KeyDown)})
	tbl.HandleKey(KeyEvent{Key: int(tcell.KeyDown)}) // should clamp
	if tbl.Selected != 2 {
		t.Errorf("expected 2, got %d", tbl.Selected)
	}
	tbl.HandleKey(KeyEvent{Key: int(tcell.KeyUp)})
	if tbl.Selected != 1 {
		t.Errorf("expected 1, got %d", tbl.Selected)
	}
}

func TestTableSelect(t *testing.T) {
	selected := -1
	rows := [][]string{{"a"}, {"b"}}
	tbl := NewTable(nil, func() [][]string { return rows }, func(i int) { selected = i })
	tbl.HandleKey(KeyEvent{Key: int(tcell.KeyDown)})
	tbl.HandleKey(KeyEvent{Key: int(tcell.KeyEnter)})
	if selected != 1 {
		t.Errorf("expected 1, got %d", selected)
	}
}

func TestTableRender(t *testing.T) {
	r, s := newTestRenderer(40, 10)
	defer s.Fini()

	cols := []TableColumn{{Header: "Name", Grow: 1}}
	rows := [][]string{{"Alice"}, {"Bob"}}
	tbl := NewTable(cols, func() [][]string { return rows }, nil)
	tbl.Render(r, 0, 0, 40, 10)

	// Border at (0,0), header "Name" at (1,1)
	if getCell(s, 1, 1) != 'N' {
		t.Errorf("expected 'N' at (1,1), got %c", getCell(s, 1, 1))
	}
	// Separator at row 2
	if getCell(s, 1, 2) != '─' {
		t.Errorf("expected separator at (1,2), got %c", getCell(s, 1, 2))
	}
	// First data row at row 3
	if getCell(s, 1, 3) != 'A' {
		t.Errorf("expected 'A' at (1,3), got %c", getCell(s, 1, 3))
	}
}

// --- Tabs tests ---

func TestTabsDefaults(t *testing.T) {
	tabs := NewTabs(
		Tab{Label: "One", Content: StaticText("content1")},
		Tab{Label: "Two", Content: StaticText("content2")},
	)
	if !tabs.Focusable() {
		t.Error("tabs should be focusable")
	}
	if tabs.Active != 0 {
		t.Errorf("expected Active=0, got %d", tabs.Active)
	}
}

func TestTabsChildrenReturnsOnlyActive(t *testing.T) {
	c1 := StaticText("one")
	c2 := StaticText("two")
	tabs := NewTabs(Tab{Label: "A", Content: c1}, Tab{Label: "B", Content: c2})

	children := tabs.Children()
	if len(children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(children))
	}
	// Should be c1 (active=0)
	if children[0] != c1 {
		t.Error("expected active tab content")
	}

	tabs.Active = 1
	children = tabs.Children()
	if children[0] != c2 {
		t.Error("expected second tab content after switch")
	}
}

func TestTabsSwitchKeys(t *testing.T) {
	tabs := NewTabs(
		Tab{Label: "A"},
		Tab{Label: "B"},
		Tab{Label: "C"},
	)

	tabs.HandleKey(KeyEvent{Key: int(tcell.KeyRight)})
	if tabs.Active != 1 {
		t.Errorf("expected 1, got %d", tabs.Active)
	}
	tabs.HandleKey(KeyEvent{Key: int(tcell.KeyRight)})
	if tabs.Active != 2 {
		t.Errorf("expected 2, got %d", tabs.Active)
	}
	tabs.HandleKey(KeyEvent{Key: int(tcell.KeyRight)}) // clamp
	if tabs.Active != 2 {
		t.Errorf("expected 2, got %d", tabs.Active)
	}
	tabs.HandleKey(KeyEvent{Key: int(tcell.KeyLeft)})
	if tabs.Active != 1 {
		t.Errorf("expected 1, got %d", tabs.Active)
	}
}

func TestTabsRender(t *testing.T) {
	r, s := newTestRenderer(40, 10)
	defer s.Fini()

	tabs := NewTabs(
		Tab{Label: "Tab1", Content: StaticText("Hello")},
		Tab{Label: "Tab2", Content: StaticText("World")},
	)
	tabs.Render(r, 0, 0, 40, 10)

	// Tab bar should show " Tab1 " starting at (0,0)
	if getCell(s, 1, 0) != 'T' {
		t.Errorf("expected 'T' at (1,0), got %c", getCell(s, 1, 0))
	}
	// Separator at row 1
	if getCell(s, 0, 1) != '─' {
		t.Errorf("expected separator at (0,1), got %c", getCell(s, 0, 1))
	}
	// Content at row 2
	if getCell(s, 0, 2) != 'H' {
		t.Errorf("expected 'H' at (0,2), got %c", getCell(s, 0, 2))
	}
}

// --- FuzzyFinder tests ---

func TestFuzzyMatchAll(t *testing.T) {
	items := []string{"foo", "bar", "foobar", "baz"}
	matches := FuzzyMatchAll("fo", items)

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	// Both "foo" and "foobar" should match
	texts := map[string]bool{}
	for _, m := range matches {
		texts[m.Text] = true
	}
	if !texts["foo"] || !texts["foobar"] {
		t.Errorf("expected foo and foobar, got %v", texts)
	}
}

func TestFuzzyMatchAllEmpty(t *testing.T) {
	items := []string{"foo", "bar"}
	matches := FuzzyMatchAll("", items)
	// Empty query matches everything with score 0
	if len(matches) != 2 {
		t.Errorf("empty query should match all items, got %d", len(matches))
	}
}

func TestFuzzyMatchNoMatch(t *testing.T) {
	items := []string{"foo", "bar"}
	matches := FuzzyMatchAll("xyz", items)
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestFuzzyMatchCaseInsensitive(t *testing.T) {
	items := []string{"FooBar", "hello"}
	matches := FuzzyMatchAll("fb", items)
	if len(matches) != 1 || matches[0].Text != "FooBar" {
		t.Errorf("expected FooBar match, got %v", matches)
	}
}

func TestFuzzyFinderDefaults(t *testing.T) {
	ff := NewFuzzyFinder([]string{"a", "b"}, nil)
	if !ff.Focusable() {
		t.Error("fuzzyfinder should be focusable")
	}
	// Initially all items shown
	if len(ff.Filtered()) != 2 {
		t.Errorf("expected 2 filtered items, got %d", len(ff.Filtered()))
	}
}

func TestFuzzyFinderFilters(t *testing.T) {
	ff := NewFuzzyFinder([]string{"apple", "banana", "avocado"}, nil)
	ff.Editing = true
	ff.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'a'})

	// "apple" and "avocado" and "banana" all contain 'a'
	filtered := ff.Filtered()
	if len(filtered) != 3 {
		t.Errorf("expected 3 items with 'a', got %d", len(filtered))
	}

	ff.HandleKey(KeyEvent{Key: int(tcell.KeyRune), Rune: 'p'})
	filtered = ff.Filtered()
	if len(filtered) != 1 || filtered[0].Text != "apple" {
		t.Errorf("expected only apple, got %v", filtered)
	}
}

func TestFuzzyFinderSelect(t *testing.T) {
	selected := -1
	ff := NewFuzzyFinder([]string{"a", "b", "c"}, func(i int) { selected = i })
	ff.HandleKey(KeyEvent{Key: int(tcell.KeyDown)})
	ff.HandleKey(KeyEvent{Key: int(tcell.KeyEnter)})
	if selected != 1 {
		t.Errorf("expected original index 1, got %d", selected)
	}
}

func TestFuzzyFinderRender(t *testing.T) {
	r, s := newTestRenderer(30, 10)
	defer s.Fini()

	ff := NewFuzzyFinder([]string{"alpha", "beta"}, nil)
	ff.Render(r, 0, 0, 30, 10)

	// Should show "> " prompt at (1,1) inside border
	if getCell(s, 1, 1) != '>' {
		t.Errorf("expected '>' at (1,1), got %c", getCell(s, 1, 1))
	}
	// Items listed below
	if getCell(s, 1, 2) != 'a' {
		t.Errorf("expected 'a' at (1,2), got %c", getCell(s, 1, 2))
	}
}

// --- Terminal tests ---

func TestTerminalANSIParser(t *testing.T) {
	term := NewTerminal("echo") // won't start, just testing parser
	term.rows = 24
	term.cols = 80
	term.allocBuf()

	// Write "Hello"
	term.Write([]byte("Hello"))
	ch, _ := term.GetCell(0, 0)
	if ch != 'H' {
		t.Errorf("expected 'H', got %c", ch)
	}
	ch, _ = term.GetCell(0, 4)
	if ch != 'o' {
		t.Errorf("expected 'o', got %c", ch)
	}

	row, col := term.CursorPos()
	if row != 0 || col != 5 {
		t.Errorf("expected cursor at (0,5), got (%d,%d)", row, col)
	}
}

func TestTerminalNewline(t *testing.T) {
	term := NewTerminal("echo")
	term.rows = 24
	term.cols = 80
	term.allocBuf()

	term.Write([]byte("A\r\nB"))
	ch, _ := term.GetCell(0, 0)
	if ch != 'A' {
		t.Errorf("expected 'A', got %c", ch)
	}
	ch, _ = term.GetCell(1, 0)
	if ch != 'B' {
		t.Errorf("expected 'B' on second row, got %c", ch)
	}
}

func TestTerminalCursorMovement(t *testing.T) {
	term := NewTerminal("echo")
	term.rows = 24
	term.cols = 80
	term.allocBuf()

	// Move cursor to row 3, col 5 (1-indexed in ANSI)
	term.Write([]byte("\x1b[4;6H"))
	row, col := term.CursorPos()
	if row != 3 || col != 5 {
		t.Errorf("expected cursor at (3,5), got (%d,%d)", row, col)
	}

	// Move up 2
	term.Write([]byte("\x1b[2A"))
	row, _ = term.CursorPos()
	if row != 1 {
		t.Errorf("expected row 1, got %d", row)
	}
}

func TestTerminalEraseDisplay(t *testing.T) {
	term := NewTerminal("echo")
	term.rows = 5
	term.cols = 10
	term.allocBuf()

	term.Write([]byte("ABCDE\nFGHIJ"))
	// Erase entire display
	term.Write([]byte("\x1b[2J"))

	ch, _ := term.GetCell(0, 0)
	if ch != ' ' {
		t.Errorf("expected space after erase, got %c", ch)
	}
}

func TestTerminalSGRColors(t *testing.T) {
	term := NewTerminal("echo")
	term.rows = 24
	term.cols = 80
	term.allocBuf()

	// Set red foreground (31), then write
	term.Write([]byte("\x1b[31mR"))
	_, st := term.GetCell(0, 0)
	if st.FG != tcell.ColorMaroon {
		t.Errorf("expected red foreground, got %v", st.FG)
	}

	// Reset
	term.Write([]byte("\x1b[0mN"))
	_, st = term.GetCell(0, 1)
	if st.FG != style.CurrentTheme.FG {
		t.Errorf("expected theme FG after reset, got %v", st.FG)
	}
}

func TestTerminalScrollUp(t *testing.T) {
	term := NewTerminal("echo")
	term.rows = 3
	term.cols = 5
	term.allocBuf()

	term.Write([]byte("A\r\nB\r\nC\r\nD"))

	// After writing 4 lines in a 3-row terminal, scroll should have happened.
	// Row 0 should be "B", row 1 "C", row 2 "D"
	ch, _ := term.GetCell(0, 0)
	if ch != 'B' {
		t.Errorf("expected 'B' at row 0 after scroll, got %c", ch)
	}
	ch, _ = term.GetCell(2, 0)
	if ch != 'D' {
		t.Errorf("expected 'D' at row 2, got %c", ch)
	}
}

func TestTerminalResize(t *testing.T) {
	term := NewTerminal("echo")
	term.rows = 5
	term.cols = 10
	term.allocBuf()

	term.Write([]byte("Hello"))
	term.Resize(3, 8)

	if term.rows != 3 || term.cols != 8 {
		t.Errorf("expected 3x8, got %dx%d", term.rows, term.cols)
	}
	// Content should be preserved
	ch, _ := term.GetCell(0, 0)
	if ch != 'H' {
		t.Errorf("expected 'H' preserved after resize, got %c", ch)
	}
}

// --- Rect storage tests ---

func TestRectStoredAfterRender(t *testing.T) {
	r, s := newTestRenderer(80, 24)
	defer s.Fini()

	btn := NewButton("OK", nil)
	btn.Render(r, 5, 10, 20, 3)
	rect := btn.GetRect()
	if rect.X != 5 || rect.Y != 10 || rect.W != 20 || rect.H != 3 {
		t.Errorf("expected rect (5,10,20,3), got %+v", rect)
	}
}

func TestBoxChildRectsPopulated(t *testing.T) {
	r, s := newTestRenderer(80, 24)
	defer s.Fini()

	btn1 := NewButton("A", nil)
	btn1.Flex = FlexProps{Basis: 10}
	btn2 := NewButton("B", nil)
	btn2.Flex = FlexProps{Basis: 10}

	box := HBox(btn1, btn2)
	box.Render(r, 0, 0, 80, 3)

	r1 := btn1.GetRect()
	r2 := btn2.GetRect()
	if r1.X != 0 {
		t.Errorf("btn1 should start at x=0, got %d", r1.X)
	}
	if r2.X <= r1.X {
		t.Errorf("btn2 should be to the right of btn1, got btn1.X=%d btn2.X=%d", r1.X, r2.X)
	}
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
