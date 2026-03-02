package widget

import (
	"github.com/gdamore/tcell/v2"
	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/style"
)

type Input struct {
	Base
	Value      string
	Cursor     int
	OnChange   func(string)
	Placeholder string
}

func NewInput(placeholder string, onChange func(string)) *Input {
	return &Input{
		Base:        Base{Style: style.Style{FG: tcell.ColorWhite, BG: tcell.ColorDefault, Border: style.BorderSingle}},
		Placeholder: placeholder,
		OnChange:    onChange,
	}
}

func (inp *Input) Focusable() bool { return true }

func (inp *Input) HandleKey(ev KeyEvent) bool {
	switch tcell.Key(ev.Key) {
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if inp.Cursor > 0 {
			inp.Value = inp.Value[:inp.Cursor-1] + inp.Value[inp.Cursor:]
			inp.Cursor--
			if inp.OnChange != nil {
				inp.OnChange(inp.Value)
			}
		}
		return true
	case tcell.KeyDelete:
		if inp.Cursor < len(inp.Value) {
			inp.Value = inp.Value[:inp.Cursor] + inp.Value[inp.Cursor+1:]
			if inp.OnChange != nil {
				inp.OnChange(inp.Value)
			}
		}
		return true
	case tcell.KeyLeft:
		if inp.Cursor > 0 {
			inp.Cursor--
		}
		return true
	case tcell.KeyRight:
		if inp.Cursor < len(inp.Value) {
			inp.Cursor++
		}
		return true
	case tcell.KeyRune:
		inp.Value = inp.Value[:inp.Cursor] + string(ev.Rune) + inp.Value[inp.Cursor:]
		inp.Cursor++
		if inp.OnChange != nil {
			inp.OnChange(inp.Value)
		}
		return true
	}
	return false
}

func (inp *Input) Render(r *render.Renderer, x, y, w, h int) {
	st := inp.Style
	if inp.Focused {
		st.FG = tcell.ColorYellow
	}

	r.DrawBorder(x, y, w, h, st.Border, st)

	ix, iy, iw, _ := st.InnerRect(x, y, w, h)
	if iw <= 0 {
		return
	}

	display := inp.Value
	if len(display) == 0 && !inp.Focused {
		display = inp.Placeholder
		st.FG = tcell.ColorGray
	}

	if len(display) > iw {
		display = display[:iw]
	}
	r.DrawText(ix, iy, display, st, iw)

	// Draw cursor
	if inp.Focused && inp.Cursor <= iw {
		cursorX := ix + inp.Cursor
		if cursorX < ix+iw {
			ch := ' '
			if inp.Cursor < len(inp.Value) {
				ch = rune(inp.Value[inp.Cursor])
			}
			cursorStyle := style.Style{FG: tcell.ColorBlack, BG: tcell.ColorWhite}
			r.Screen.SetContent(cursorX, iy, ch, nil, cursorStyle.TcellStyle())
		}
	}
}
