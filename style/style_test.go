package style

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestBorderChars(t *testing.T) {
	tests := []struct {
		border BorderStyle
		tl     rune
	}{
		{BorderSingle, '┌'},
		{BorderDouble, '╔'},
		{BorderRounded, '╭'},
		{BorderNone, 0},
	}
	for _, tt := range tests {
		chars := tt.border.Chars()
		if chars[0] != tt.tl {
			t.Errorf("border %d: expected tl=%c, got %c", tt.border, tt.tl, chars[0])
		}
	}
}

func TestSpacingHorizontalVertical(t *testing.T) {
	s := Spacing{Top: 1, Right: 2, Bottom: 3, Left: 4}
	if s.Horizontal() != 6 {
		t.Errorf("expected horizontal=6, got %d", s.Horizontal())
	}
	if s.Vertical() != 4 {
		t.Errorf("expected vertical=4, got %d", s.Vertical())
	}
}

func TestPad(t *testing.T) {
	p := Pad(3)
	if p.Top != 3 || p.Right != 3 || p.Bottom != 3 || p.Left != 3 {
		t.Errorf("Pad(3) should set all sides to 3: %+v", p)
	}
}

func TestPadXY(t *testing.T) {
	p := PadXY(2, 4)
	if p.Left != 2 || p.Right != 2 || p.Top != 4 || p.Bottom != 4 {
		t.Errorf("PadXY(2,4): expected L=R=2, T=B=4, got %+v", p)
	}
}

func TestDefaultStyle(t *testing.T) {
	s := DefaultStyle()
	if s.FG != tcell.ColorDefault || s.BG != tcell.ColorDefault {
		t.Error("DefaultStyle should use default colors")
	}
	if s.Border != BorderNone {
		t.Error("DefaultStyle should have no border")
	}
}

func TestTcellStyle(t *testing.T) {
	s := Style{
		FG:     tcell.ColorRed,
		BG:     tcell.ColorBlue,
		Bold:   true,
		Italic: true,
	}
	ts := s.TcellStyle()
	fg, bg, attrs := ts.Decompose()
	if fg != tcell.ColorRed {
		t.Errorf("expected fg red, got %v", fg)
	}
	if bg != tcell.ColorBlue {
		t.Errorf("expected bg blue, got %v", bg)
	}
	if attrs&tcell.AttrBold == 0 {
		t.Error("expected bold")
	}
	if attrs&tcell.AttrItalic == 0 {
		t.Error("expected italic")
	}
}

func TestTcellStyleNoBoldItalic(t *testing.T) {
	s := Style{FG: tcell.ColorDefault, BG: tcell.ColorDefault}
	ts := s.TcellStyle()
	_, _, attrs := ts.Decompose()
	if attrs&tcell.AttrBold != 0 {
		t.Error("should not be bold")
	}
	if attrs&tcell.AttrItalic != 0 {
		t.Error("should not be italic")
	}
}

func TestInnerRectNoBorder(t *testing.T) {
	s := Style{Padding: Spacing{Top: 1, Right: 2, Bottom: 1, Left: 2}}
	ix, iy, iw, ih := s.InnerRect(0, 0, 20, 10)
	if ix != 2 || iy != 1 || iw != 16 || ih != 8 {
		t.Errorf("expected (2,1,16,8), got (%d,%d,%d,%d)", ix, iy, iw, ih)
	}
}

func TestInnerRectWithBorder(t *testing.T) {
	s := Style{Border: BorderSingle, Padding: Pad(1)}
	ix, iy, iw, ih := s.InnerRect(0, 0, 20, 10)
	// border=1 each side + padding=1 each side = 2 each side
	if ix != 2 || iy != 2 || iw != 16 || ih != 6 {
		t.Errorf("expected (2,2,16,6), got (%d,%d,%d,%d)", ix, iy, iw, ih)
	}
}

func TestInnerRectTooSmall(t *testing.T) {
	s := Style{Border: BorderSingle, Padding: Pad(5)}
	_, _, iw, ih := s.InnerRect(0, 0, 4, 4)
	if iw != 0 || ih != 0 {
		t.Errorf("expected (0,0) for too-small rect, got (%d,%d)", iw, ih)
	}
}

func TestInnerRectWithOffset(t *testing.T) {
	s := Style{Border: BorderSingle}
	ix, iy, iw, ih := s.InnerRect(5, 3, 20, 10)
	if ix != 6 || iy != 4 || iw != 18 || ih != 8 {
		t.Errorf("expected (6,4,18,8), got (%d,%d,%d,%d)", ix, iy, iw, ih)
	}
}

func TestChromeWidth(t *testing.T) {
	tests := []struct {
		s    Style
		want int
	}{
		{Style{}, 0},
		{Style{Padding: Pad(2)}, 4},
		{Style{Border: BorderSingle}, 2},
		{Style{Border: BorderSingle, Padding: Pad(1)}, 4},
	}
	for i, tt := range tests {
		got := tt.s.ChromeWidth()
		if got != tt.want {
			t.Errorf("case %d: expected %d, got %d", i, tt.want, got)
		}
	}
}

func TestChromeHeight(t *testing.T) {
	tests := []struct {
		s    Style
		want int
	}{
		{Style{}, 0},
		{Style{Padding: PadXY(0, 3)}, 6},
		{Style{Border: BorderDouble}, 2},
		{Style{Border: BorderDouble, Padding: PadXY(0, 1)}, 4},
	}
	for i, tt := range tests {
		got := tt.s.ChromeHeight()
		if got != tt.want {
			t.Errorf("case %d: expected %d, got %d", i, tt.want, got)
		}
	}
}
