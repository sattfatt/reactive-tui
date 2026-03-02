package widget

import (
	"github.com/gdamore/tcell/v2"
	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/style"
)

type Input struct {
	Base
	Value       string
	Cursor      int
	OnChange    func(string)
	Placeholder string
}

// NewInput creates a single-line text input with a border (3 rows tall).
func NewInput(placeholder string, onChange func(string)) *Input {
	return &Input{
		Base: Base{
			Style: style.Style{FG: style.CurrentTheme.FG, BG: style.CurrentTheme.BG, Border: style.BorderSingle},
			Flex:  FlexProps{Basis: -1, Shrink: 1, MinHeight: 3, MinWidth: 5},
		},
		Placeholder: placeholder,
		OnChange:    onChange,
	}
}

// NewInlineInput creates a borderless single-line input (1 row tall).
// Useful for compact layouts where a full bordered input is too heavy.
func NewInlineInput(placeholder string, onChange func(string)) *Input {
	return &Input{
		Base: Base{
			Style: style.Style{FG: style.CurrentTheme.FG, BG: style.CurrentTheme.BG},
			Flex:  FlexProps{Basis: -1, Grow: 1, Shrink: 1, MinHeight: 1, MaxHeight: 1, MinWidth: 5},
		},
		Placeholder: placeholder,
		OnChange:    onChange,
	}
}

func (inp *Input) Focusable() bool  { return true }
func (inp *Input) IsEditable() bool { return true }

func (inp *Input) HandleKey(ev KeyEvent) bool {
	switch tcell.Key(ev.Key) {
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if !inp.Editing {
			return false
		}
		if inp.Cursor > 0 {
			inp.Value = inp.Value[:inp.Cursor-1] + inp.Value[inp.Cursor:]
			inp.Cursor--
			if inp.OnChange != nil {
				inp.OnChange(inp.Value)
			}
		}
		return true
	case tcell.KeyDelete:
		if !inp.Editing {
			return false
		}
		if inp.Cursor < len(inp.Value) {
			inp.Value = inp.Value[:inp.Cursor] + inp.Value[inp.Cursor+1:]
			if inp.OnChange != nil {
				inp.OnChange(inp.Value)
			}
		}
		return true
	case tcell.KeyLeft, tcell.KeyRight:
		if !inp.Editing {
			return false
		}
		if tcell.Key(ev.Key) == tcell.KeyLeft {
			if inp.Cursor > 0 {
				inp.Cursor--
			}
		} else {
			if inp.Cursor < len(inp.Value) {
				inp.Cursor++
			}
		}
		return true
	case tcell.KeyRune:
		if !inp.Editing {
			return false
		}
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
	inp.Base.SetRect(x, y, w, h)
	st := inp.Style

	if st.Border != style.BorderNone {
		// Bordered mode: draw border with themed color
		borderSt := st
		if inp.Focused {
			if inp.Editing {
				borderSt.FG = style.CurrentTheme.EditFocusFG
			} else {
				borderSt.FG = style.CurrentTheme.NavFocusFG
			}
		} else {
			borderSt.FG = style.CurrentTheme.BorderFG
		}
		r.DrawBorder(x, y, w, h, st.Border, borderSt)
		inp.Base.RenderLabel(r, x, y, w)
	} else {
		// Inline mode: fill background, use underline-style highlight
		bgSt := st
		if inp.Focused {
			bgSt.BG = style.CurrentTheme.SelectionBG
		} else {
			bgSt.BG = style.CurrentTheme.ButtonBG
		}
		r.FillRect(x, y, w, h, ' ', bgSt)
		st = bgSt
	}

	ix, iy, iw, ih := st.InnerRect(x, y, w, h)
	if iw <= 0 {
		return
	}

	// Center text vertically within inner rect
	iy = iy + ih/2

	display := inp.Value
	if len(display) == 0 && !inp.Focused {
		display = inp.Placeholder
		st.FG = style.CurrentTheme.PlaceholderFG
	}

	// Scroll horizontally to keep cursor visible
	scrollX := 0
	if inp.Cursor > iw-1 {
		scrollX = inp.Cursor - iw + 1
	}
	if scrollX > 0 {
		if scrollX < len(display) {
			display = display[scrollX:]
		} else {
			display = ""
		}
	}

	if len(display) > iw {
		display = display[:iw]
	}
	r.DrawText(ix, iy, display, st, iw)

	// Draw cursor
	cursorScreen := inp.Cursor - scrollX
	if inp.Focused && cursorScreen >= 0 && cursorScreen < iw {
		cursorX := ix + cursorScreen
		ch := ' '
		if inp.Cursor < len(inp.Value) {
			ch = rune(inp.Value[inp.Cursor])
		}
		cursorStyle := style.Style{FG: style.CurrentTheme.CursorFG, BG: style.CurrentTheme.CursorBG}
		r.Screen.SetContent(cursorX, iy, ch, nil, cursorStyle.TcellStyle())
	}
}
