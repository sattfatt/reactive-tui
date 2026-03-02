package widget

import (
	"github.com/gdamore/tcell/v2"
	"github.com/sattfatt/reactive-tui/render"
	"github.com/sattfatt/reactive-tui/style"
)

type List struct {
	Base
	Items    func() []string // reactive getter
	Selected int
	OnSelect func(int)
}

func NewList(items func() []string, onSelect func(int)) *List {
	return &List{
		Base: Base{
			Style: style.Style{FG: style.CurrentTheme.FG, BG: style.CurrentTheme.BG},
			Flex:  FlexProps{Basis: -1, Shrink: 1, Grow: 1, MinHeight: 1, MinWidth: 1},
		},
		Items:    items,
		OnSelect: onSelect,
	}
}

func (l *List) Focusable() bool { return true }

func (l *List) HandleKey(ev KeyEvent) bool {
	items := l.Items()
	switch tcell.Key(ev.Key) {
	case tcell.KeyUp:
		if l.Selected > 0 {
			l.Selected--
		}
		return true
	case tcell.KeyDown:
		if l.Selected < len(items)-1 {
			l.Selected++
		}
		return true
	case tcell.KeyEnter:
		if l.OnSelect != nil && l.Selected < len(items) {
			l.OnSelect(l.Selected)
		}
		return true
	}
	return false
}

func (l *List) Render(r *render.Renderer, x, y, w, h int) {
	l.Base.SetRect(x, y, w, h)
	if l.Style.Border != style.BorderNone {
		r.DrawBorder(x, y, w, h, l.Style.Border, l.Style)
		l.Base.RenderLabel(r, x, y, w)
	}

	ix, iy, iw, ih := l.Style.InnerRect(x, y, w, h)
	if iw <= 0 || ih <= 0 {
		return
	}

	items := l.Items()
	for i, item := range items {
		if i >= ih {
			break
		}
		st := l.Style
		if i == l.Selected && l.Focused {
			st.FG = style.CurrentTheme.SelectionFG
			st.BG = style.CurrentTheme.SelectionBG
			r.FillRect(ix, iy+i, iw, 1, ' ', st)
		}
		label := item
		if len(label) > iw {
			label = label[:iw]
		}
		r.DrawText(ix, iy+i, label, st, iw)
	}
}
