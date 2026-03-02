package style

import "github.com/gdamore/tcell/v2"

type BorderStyle int

const (
	BorderNone BorderStyle = iota
	BorderSingle
	BorderDouble
	BorderRounded
)

// BorderChars returns the runes for a border style:
// [topLeft, topRight, bottomLeft, bottomRight, horizontal, vertical]
func (b BorderStyle) Chars() [6]rune {
	switch b {
	case BorderSingle:
		return [6]rune{'┌', '┐', '└', '┘', '─', '│'}
	case BorderDouble:
		return [6]rune{'╔', '╗', '╚', '╝', '═', '║'}
	case BorderRounded:
		return [6]rune{'╭', '╮', '╰', '╯', '─', '│'}
	default:
		return [6]rune{}
	}
}

type Align int

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
	AlignStretch
)

type Justify int

const (
	JustifyStart Justify = iota
	JustifyCenter
	JustifyEnd
	JustifySpaceBetween
)

type Style struct {
	Border  BorderStyle
	Padding Spacing
	Margin  Spacing
	FG      tcell.Color
	BG      tcell.Color
	Bold    bool
	Italic  bool
}

type Spacing struct {
	Top, Right, Bottom, Left int
}

func (s Spacing) Horizontal() int { return s.Left + s.Right }
func (s Spacing) Vertical() int   { return s.Top + s.Bottom }

func Pad(all int) Spacing {
	return Spacing{all, all, all, all}
}

func PadXY(x, y int) Spacing {
	return Spacing{y, x, y, x}
}

func DefaultStyle() Style {
	return Style{
		FG: tcell.ColorDefault,
		BG: tcell.ColorDefault,
	}
}

func (s Style) TcellStyle() tcell.Style {
	st := tcell.StyleDefault.Foreground(s.FG).Background(s.BG)
	if s.Bold {
		st = st.Bold(true)
	}
	if s.Italic {
		st = st.Italic(true)
	}
	return st
}

// InnerRect returns the rect after applying border and padding.
func (s Style) InnerRect(x, y, w, h int) (int, int, int, int) {
	bw := 0
	if s.Border != BorderNone {
		bw = 1
	}
	ix := x + bw + s.Padding.Left
	iy := y + bw + s.Padding.Top
	iw := w - 2*bw - s.Padding.Horizontal()
	ih := h - 2*bw - s.Padding.Vertical()
	if iw < 0 {
		iw = 0
	}
	if ih < 0 {
		ih = 0
	}
	return ix, iy, iw, ih
}

// OuterSize returns how much space border+padding consume.
func (s Style) ChromeWidth() int {
	bw := 0
	if s.Border != BorderNone {
		bw = 2
	}
	return bw + s.Padding.Horizontal()
}

func (s Style) ChromeHeight() int {
	bw := 0
	if s.Border != BorderNone {
		bw = 2
	}
	return bw + s.Padding.Vertical()
}
