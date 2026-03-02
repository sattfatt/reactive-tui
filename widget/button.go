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
			Style: style.Style{FG: style.CurrentTheme.FG, BG: style.CurrentTheme.BG},
			Flex:  FlexProps{Basis: -1, Shrink: 1, MinHeight: 1, MaxHeight: 1, MinWidth: 3},
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
		st.FG = style.CurrentTheme.SelectionFG
		st.BG = style.CurrentTheme.SelectionBG
	} else {
		st.FG = style.CurrentTheme.ButtonFG
		st.BG = style.CurrentTheme.ButtonBG
	}

	// Fill background and center label both horizontally and vertically
	r.FillRect(x, y, w, h, ' ', st)
	label := "[ " + b.Label + " ]"
	if len(label) > w {
		label = b.Label
		if len(label) > w {
			label = label[:w]
		}
	}
	labelLen := len(label)
	left := (w - labelLen) / 2
	midY := y + h/2
	r.DrawText(x+left, midY, label, st, labelLen)
}
