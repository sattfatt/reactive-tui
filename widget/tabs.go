package widget

import (
	"github.com/gdamore/tcell/v2"
	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/style"
)

type Tab struct {
	Label   string
	Content Node
}

type Tabs struct {
	Base
	Tabs   []Tab
	Active int
}

func NewTabs(tabs ...Tab) *Tabs {
	return &Tabs{
		Base: Base{
			Style: style.Style{FG: tcell.ColorWhite, BG: tcell.ColorDefault},
			Flex:  FlexProps{Basis: -1, Grow: 1, Shrink: 1, MinHeight: 4, MinWidth: 10},
		},
		Tabs: tabs,
	}
}

func (t *Tabs) Focusable() bool { return true }

// Children returns only the active tab's content so focus ring only traverses it.
func (t *Tabs) Children() []Node {
	if t.Active >= 0 && t.Active < len(t.Tabs) && t.Tabs[t.Active].Content != nil {
		return []Node{t.Tabs[t.Active].Content}
	}
	return nil
}

func (t *Tabs) HandleKey(ev KeyEvent) bool {
	switch tcell.Key(ev.Key) {
	case tcell.KeyLeft:
		if t.Active > 0 {
			t.Active--
			return true
		}
	case tcell.KeyRight:
		if t.Active < len(t.Tabs)-1 {
			t.Active++
			return true
		}
	}
	return false
}

func (t *Tabs) Render(r *render.Renderer, x, y, w, h int) {
	t.Base.SetRect(x, y, w, h)
	st := t.Style
	if t.Focused {
		st.FG = tcell.ColorYellow
	}

	if st.Border != style.BorderNone {
		r.DrawBorder(x, y, w, h, st.Border, st)
	}

	ix, iy, iw, ih := st.InnerRect(x, y, w, h)
	if iw <= 0 || ih <= 0 {
		return
	}

	// Draw tab bar
	col := 0
	for i, tab := range t.Tabs {
		label := " " + tab.Label + " "
		if len(label)+col > iw {
			break
		}

		tabStyle := st
		if i == t.Active {
			tabStyle.FG, tabStyle.BG = tcell.ColorBlack, tcell.ColorWhite
			r.FillRect(ix+col, iy, len(label), 1, ' ', tabStyle)
		}
		r.DrawText(ix+col, iy, label, tabStyle, len(label))
		col += len(label)
	}

	// Separator line
	if ih > 1 {
		for c := 0; c < iw; c++ {
			r.Screen.SetContent(ix+c, iy+1, '─', nil, st.TcellStyle())
		}
	}

	// Render active content
	contentH := ih - 2 // tab bar + separator
	if contentH > 0 && t.Active >= 0 && t.Active < len(t.Tabs) && t.Tabs[t.Active].Content != nil {
		t.Tabs[t.Active].Content.Render(r, ix, iy+2, iw, contentH)
	}
}
