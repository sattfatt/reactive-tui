package widget

import (
	"github.com/gdamore/tcell/v2"
	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/style"
)

type Button struct {
	Base
	Label   string
	OnClick func()
}

func NewButton(label string, onClick func()) *Button {
	return &Button{
		Base: Base{
			Style: style.Style{FG: tcell.ColorWhite, BG: tcell.ColorDefault},
			Flex:  FlexProps{Basis: -1, Shrink: 1, MinHeight: 3, MinWidth: 5},
		},
		Label:   label,
		OnClick: onClick,
	}
}

func (b *Button) Focusable() bool { return true }

func (b *Button) HandleKey(ev KeyEvent) bool {
	if ev.Key == int(tcell.KeyEnter) || ev.Rune == ' ' {
		if b.OnClick != nil {
			b.OnClick()
		}
		return true
	}
	return false
}

func (b *Button) Render(r *render.Renderer, x, y, w, h int) {
	b.Base.SetRect(x, y, w, h)
	st := b.Style
	if b.Focused {
		// Invert colors when focused
		st.FG, st.BG = st.BG, st.FG
		if st.FG == tcell.ColorDefault {
			st.FG = tcell.ColorBlack
		}
		if st.BG == tcell.ColorDefault {
			st.BG = tcell.ColorWhite
		}
	}

	if st.Border == style.BorderNone {
		st.Border = style.BorderSingle
	}

	r.DrawBorder(x, y, w, h, st.Border, st)

	ix, iy, iw, _ := st.InnerRect(x, y, w, h)
	if iw <= 0 {
		return
	}

	// Center the label symmetrically by padding both sides
	label := b.Label
	if len(label) > iw {
		label = label[:iw]
	}
	pad := iw - len(label)
	left := pad / 2
	right := pad - left
	centered := ""
	for range left {
		centered += " "
	}
	centered += label
	for range right {
		centered += " "
	}
	r.DrawText(ix, iy, centered, st, iw)
}
