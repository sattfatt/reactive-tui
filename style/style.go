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

// Theme defines semantic color roles for the entire UI.
type Theme struct {
	FG          tcell.Color // default foreground
	BG          tcell.Color // default background
	NavFocusFG  tcell.Color // border/fg in nav-mode focus
	EditFocusFG tcell.Color // border/fg in edit-mode focus
	SelectionFG tcell.Color // selected row/tab foreground
	SelectionBG tcell.Color // selected row/tab background
	CursorFG    tcell.Color // block cursor foreground
	CursorBG    tcell.Color // block cursor background
	PlaceholderFG    tcell.Color // input placeholder text
	ProgressFilledFG tcell.Color // progress bar filled portion
	ProgressEmptyFG  tcell.Color // progress bar empty portion
	ButtonFG    tcell.Color // inactive button text
	ButtonBG    tcell.Color // inactive button background
	BorderFG    tcell.Color // unfocused border accent
	MutedFG     tcell.Color // help text, comments
}

// CurrentTheme is the active theme. Widgets read from this at render time.
var CurrentTheme = TokyoNight()

// TokyoNight returns the Tokyo Night color theme.
func TokyoNight() Theme {
	return Theme{
		FG: tcell.NewRGBColor(0xc0, 0xca, 0xf5), // #c0caf5
		BG: tcell.NewRGBColor(0x1a, 0x1b, 0x26), // #1a1b26

		NavFocusFG:  tcell.NewRGBColor(0xe0, 0xaf, 0x68), // #e0af68
		EditFocusFG: tcell.NewRGBColor(0xf7, 0x76, 0x8e), // #f7768e

		SelectionFG: tcell.NewRGBColor(0xc0, 0xca, 0xf5), // #c0caf5
		SelectionBG: tcell.NewRGBColor(0x28, 0x34, 0x57), // #283457

		CursorFG: tcell.NewRGBColor(0x1a, 0x1b, 0x26), // #1a1b26
		CursorBG: tcell.NewRGBColor(0xc0, 0xca, 0xf5), // #c0caf5

		PlaceholderFG:    tcell.NewRGBColor(0x56, 0x5f, 0x89), // #565f89
		ProgressFilledFG: tcell.NewRGBColor(0x9e, 0xce, 0x6a), // #9ece6a
		ProgressEmptyFG:  tcell.NewRGBColor(0x56, 0x5f, 0x89), // #565f89

		ButtonFG: tcell.NewRGBColor(0x7a, 0xa2, 0xf7), // #7aa2f7 (blue)
		ButtonBG: tcell.NewRGBColor(0x24, 0x28, 0x3b), // #24283b (surface)
		BorderFG: tcell.NewRGBColor(0x27, 0xa1, 0xb9), // #27a1b9
		MutedFG:  tcell.NewRGBColor(0x56, 0x5f, 0x89), // #565f89
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
