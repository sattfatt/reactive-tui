package widget

import (
	"github.com/gdamore/tcell/v2"
	"github.com/sattfatt/reactive-tui/render"
	"github.com/sattfatt/reactive-tui/style"
)

// Checkbox is a boolean toggle widget. It does not require edit mode —
// Space or Enter toggles the value directly in navigation mode.
type Checkbox struct {
	Base
	Checked  bool
	Text     string // inline text next to the check box
	OnChange func(bool)
}

// NewCheckbox creates a checkbox with inline text and change callback.
func NewCheckbox(text string, onChange func(bool)) *Checkbox {
	return &Checkbox{
		Base: Base{
			Style: style.Style{FG: style.CurrentTheme.FG, BG: style.CurrentTheme.BG},
			Flex:  FlexProps{Basis: -1, Grow: 1, Shrink: 1, MinHeight: 1, MaxHeight: 1, MinWidth: 6},
		},
		Text:     text,
		OnChange: onChange,
	}
}

func (c *Checkbox) Focusable() bool { return true }

func (c *Checkbox) HandleKey(ev KeyEvent) bool {
	if ev.Key == int(tcell.KeyEnter) || ev.Rune == ' ' {
		c.Checked = !c.Checked
		if c.OnChange != nil {
			c.OnChange(c.Checked)
		}
		return true
	}
	return false
}

func (c *Checkbox) Render(r *render.Renderer, x, y, w, h int) {
	c.Base.SetRect(x, y, w, h)
	st := c.Style
	if c.Focused {
		st.FG = style.CurrentTheme.SelectionFG
		st.BG = style.CurrentTheme.SelectionBG
	} else {
		st.FG = style.CurrentTheme.ButtonFG
		st.BG = style.CurrentTheme.ButtonBG
	}

	r.FillRect(x, y, w, h, ' ', st)

	check := "[ ]"
	if c.Checked {
		check = "[✓]"
	}
	text := check + " " + c.Text
	if len(text) > w {
		text = text[:w]
	}

	// Center vertically
	midY := y + h/2
	r.DrawText(x+1, midY, text, st, w-1)

	// Draw the check mark character in accent color when checked
	if c.Checked && w > 2 {
		checkSt := st
		checkSt.FG = style.CurrentTheme.ProgressFilledFG
		r.Screen.SetContent(x+2, midY, '✓', nil, checkSt.TcellStyle())
	}
}
