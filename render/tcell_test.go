package render

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/sattfatt/reactive-tui/style"
)

func newTestScreen(w, h int) tcell.SimulationScreen {
	s := tcell.NewSimulationScreen("")
	_ = s.Init()
	s.SetSize(w, h)
	return s
}

func getCell(s tcell.SimulationScreen, x, y int) (rune, tcell.Style) {
	r, _, st, _ := s.GetContent(x, y)
	return r, st
}

func TestNew(t *testing.T) {
	s := newTestScreen(80, 24)
	defer s.Fini()
	r := New(s)
	if r.Screen != s {
		t.Error("renderer should hold the screen")
	}
}

func TestSize(t *testing.T) {
	s := newTestScreen(80, 24)
	defer s.Fini()
	r := New(s)
	w, h := r.Size()
	if w != 80 || h != 24 {
		t.Errorf("expected 80x24, got %dx%d", w, h)
	}
}

func TestDrawText(t *testing.T) {
	s := newTestScreen(80, 24)
	defer s.Fini()
	r := New(s)

	st := style.DefaultStyle()
	r.DrawText(0, 0, "hello", st, 80)

	expected := "hello"
	for i, ch := range expected {
		got, _ := getCell(s, i, 0)
		if got != ch {
			t.Errorf("pos %d: expected %c, got %c", i, ch, got)
		}
	}
}

func TestDrawTextClipped(t *testing.T) {
	s := newTestScreen(80, 24)
	defer s.Fini()
	r := New(s)

	st := style.DefaultStyle()
	r.DrawText(0, 0, "hello world", st, 5)

	// Should only render "hello"
	got, _ := getCell(s, 4, 0)
	if got != 'o' {
		t.Errorf("expected 'o' at pos 4, got %c", got)
	}
	got, _ = getCell(s, 5, 0)
	if got == 'w' {
		t.Error("text should be clipped at maxWidth=5")
	}
}

func TestDrawTextWithStyle(t *testing.T) {
	s := newTestScreen(80, 24)
	defer s.Fini()
	r := New(s)

	st := style.Style{FG: tcell.ColorRed, BG: tcell.ColorBlue}
	r.DrawText(2, 3, "X", st, 10)

	_, cellStyle := getCell(s, 2, 3)
	fg, bg, _ := cellStyle.Decompose()
	if fg != tcell.ColorRed {
		t.Errorf("expected red fg, got %v", fg)
	}
	if bg != tcell.ColorBlue {
		t.Errorf("expected blue bg, got %v", bg)
	}
}

func TestFillRect(t *testing.T) {
	s := newTestScreen(80, 24)
	defer s.Fini()
	r := New(s)

	st := style.DefaultStyle()
	r.FillRect(1, 1, 3, 2, '#', st)

	for y := 1; y <= 2; y++ {
		for x := 1; x <= 3; x++ {
			got, _ := getCell(s, x, y)
			if got != '#' {
				t.Errorf("(%d,%d): expected '#', got %c", x, y, got)
			}
		}
	}
	// Outside should be empty
	got, _ := getCell(s, 0, 0)
	if got == '#' {
		t.Error("(0,0) should not be filled")
	}
}

func TestDrawBorder(t *testing.T) {
	s := newTestScreen(80, 24)
	defer s.Fini()
	r := New(s)

	st := style.DefaultStyle()
	r.DrawBorder(0, 0, 5, 3, style.BorderSingle, st)

	tl, _ := getCell(s, 0, 0)
	tr, _ := getCell(s, 4, 0)
	bl, _ := getCell(s, 0, 2)
	br, _ := getCell(s, 4, 2)

	if tl != '┌' {
		t.Errorf("expected ┌ at TL, got %c", tl)
	}
	if tr != '┐' {
		t.Errorf("expected ┐ at TR, got %c", tr)
	}
	if bl != '└' {
		t.Errorf("expected └ at BL, got %c", bl)
	}
	if br != '┘' {
		t.Errorf("expected ┘ at BR, got %c", br)
	}

	// Horizontal edges
	for x := 1; x < 4; x++ {
		top, _ := getCell(s, x, 0)
		bottom, _ := getCell(s, x, 2)
		if top != '─' {
			t.Errorf("expected ─ at (%d,0), got %c", x, top)
		}
		if bottom != '─' {
			t.Errorf("expected ─ at (%d,2), got %c", x, bottom)
		}
	}

	// Vertical edges
	left, _ := getCell(s, 0, 1)
	right, _ := getCell(s, 4, 1)
	if left != '│' {
		t.Errorf("expected │ at left, got %c", left)
	}
	if right != '│' {
		t.Errorf("expected │ at right, got %c", right)
	}
}

func TestDrawBorderRounded(t *testing.T) {
	s := newTestScreen(80, 24)
	defer s.Fini()
	r := New(s)

	st := style.DefaultStyle()
	r.DrawBorder(0, 0, 4, 3, style.BorderRounded, st)

	tl, _ := getCell(s, 0, 0)
	if tl != '╭' {
		t.Errorf("expected ╭ for rounded border, got %c", tl)
	}
}

func TestDrawBorderNone(t *testing.T) {
	s := newTestScreen(80, 24)
	defer s.Fini()
	r := New(s)

	st := style.DefaultStyle()
	r.DrawBorder(0, 0, 5, 3, style.BorderNone, st)

	// Should not draw anything
	ch, _ := getCell(s, 0, 0)
	if ch == '┌' || ch == '╭' || ch == '╔' {
		t.Error("BorderNone should not draw corners")
	}
}

func TestDrawBorderTooSmall(t *testing.T) {
	s := newTestScreen(80, 24)
	defer s.Fini()
	r := New(s)

	st := style.DefaultStyle()
	// 1x1 is too small for a border
	r.DrawBorder(0, 0, 1, 1, style.BorderSingle, st)

	ch, _ := getCell(s, 0, 0)
	if ch == '┌' {
		t.Error("border should not be drawn for w<2 or h<2")
	}
}

func TestClearAndShow(t *testing.T) {
	s := newTestScreen(80, 24)
	defer s.Fini()
	r := New(s)

	st := style.DefaultStyle()
	r.DrawText(0, 0, "test", st, 10)
	r.Clear()
	r.Show()

	ch, _ := getCell(s, 0, 0)
	if ch == 't' {
		t.Error("Clear should have removed text")
	}
}
