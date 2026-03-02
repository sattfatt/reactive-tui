package widget

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/sattfatt/reactive-tui/render"
	"github.com/sattfatt/reactive-tui/style"
)

type TextArea struct {
	Base
	Lines     []string
	CursorRow int
	CursorCol int
	ScrollY   int
	OnChange  func(string)
}

func (ta *TextArea) IsEditable() bool { return true }

func NewTextArea(onChange func(string)) *TextArea {
	return &TextArea{
		Base: Base{
			Style: style.Style{FG: style.CurrentTheme.FG, BG: style.CurrentTheme.BG, Border: style.BorderSingle},
			Flex:  FlexProps{Basis: -1, Grow: 1, Shrink: 1, MinHeight: 5, MinWidth: 10},
		},
		Lines:    []string{""},
		OnChange: onChange,
	}
}

func (ta *TextArea) Focusable() bool { return true }

func (ta *TextArea) Text() string {
	return strings.Join(ta.Lines, "\n")
}

func (ta *TextArea) SetText(s string) {
	ta.Lines = strings.Split(s, "\n")
	if len(ta.Lines) == 0 {
		ta.Lines = []string{""}
	}
	ta.CursorRow = 0
	ta.CursorCol = 0
	ta.ScrollY = 0
}

func (ta *TextArea) changed() {
	if ta.OnChange != nil {
		ta.OnChange(ta.Text())
	}
}

func (ta *TextArea) HandleKey(ev KeyEvent) bool {
	switch tcell.Key(ev.Key) {
	case tcell.KeyRune:
		if !ta.Editing {
			return false // let nav mode handle hjkl etc.
		}
		line := ta.Lines[ta.CursorRow]
		ta.Lines[ta.CursorRow] = line[:ta.CursorCol] + string(ev.Rune) + line[ta.CursorCol:]
		ta.CursorCol++
		ta.changed()
		return true

	case tcell.KeyEnter:
		line := ta.Lines[ta.CursorRow]
		before := line[:ta.CursorCol]
		after := line[ta.CursorCol:]
		ta.Lines[ta.CursorRow] = before
		// Insert new line after current
		newLines := make([]string, len(ta.Lines)+1)
		copy(newLines, ta.Lines[:ta.CursorRow+1])
		newLines[ta.CursorRow+1] = after
		copy(newLines[ta.CursorRow+2:], ta.Lines[ta.CursorRow+1:])
		ta.Lines = newLines
		ta.CursorRow++
		ta.CursorCol = 0
		ta.changed()
		return true

	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if ta.CursorCol > 0 {
			line := ta.Lines[ta.CursorRow]
			ta.Lines[ta.CursorRow] = line[:ta.CursorCol-1] + line[ta.CursorCol:]
			ta.CursorCol--
		} else if ta.CursorRow > 0 {
			// Merge with previous line
			prevLen := len(ta.Lines[ta.CursorRow-1])
			ta.Lines[ta.CursorRow-1] += ta.Lines[ta.CursorRow]
			ta.Lines = append(ta.Lines[:ta.CursorRow], ta.Lines[ta.CursorRow+1:]...)
			ta.CursorRow--
			ta.CursorCol = prevLen
		}
		ta.changed()
		return true

	case tcell.KeyDelete:
		line := ta.Lines[ta.CursorRow]
		if ta.CursorCol < len(line) {
			ta.Lines[ta.CursorRow] = line[:ta.CursorCol] + line[ta.CursorCol+1:]
		} else if ta.CursorRow < len(ta.Lines)-1 {
			// Merge with next line
			ta.Lines[ta.CursorRow] += ta.Lines[ta.CursorRow+1]
			ta.Lines = append(ta.Lines[:ta.CursorRow+1], ta.Lines[ta.CursorRow+2:]...)
		}
		ta.changed()
		return true

	case tcell.KeyLeft, tcell.KeyRight, tcell.KeyUp, tcell.KeyDown:
		if !ta.Editing {
			return false
		}
		switch tcell.Key(ev.Key) {
		case tcell.KeyLeft:
			if ta.CursorCol > 0 {
				ta.CursorCol--
			} else if ta.CursorRow > 0 {
				ta.CursorRow--
				ta.CursorCol = len(ta.Lines[ta.CursorRow])
			}
		case tcell.KeyRight:
			if ta.CursorCol < len(ta.Lines[ta.CursorRow]) {
				ta.CursorCol++
			} else if ta.CursorRow < len(ta.Lines)-1 {
				ta.CursorRow++
				ta.CursorCol = 0
			}
		case tcell.KeyUp:
			if ta.CursorRow > 0 {
				ta.CursorRow--
				if ta.CursorCol > len(ta.Lines[ta.CursorRow]) {
					ta.CursorCol = len(ta.Lines[ta.CursorRow])
				}
			}
		case tcell.KeyDown:
			if ta.CursorRow < len(ta.Lines)-1 {
				ta.CursorRow++
				if ta.CursorCol > len(ta.Lines[ta.CursorRow]) {
					ta.CursorCol = len(ta.Lines[ta.CursorRow])
				}
			}
		}
		return true
	}
	return false
}

func (ta *TextArea) Render(r *render.Renderer, x, y, w, h int) {
	ta.Base.SetRect(x, y, w, h)
	st := ta.Style

	borderSt := st
	if ta.Focused {
		if ta.Editing {
			borderSt.FG = style.CurrentTheme.EditFocusFG
		} else {
			borderSt.FG = style.CurrentTheme.NavFocusFG
		}
	} else {
		borderSt.FG = style.CurrentTheme.BorderFG
	}
	if st.Border != style.BorderNone {
		r.DrawBorder(x, y, w, h, st.Border, borderSt)
		ta.Base.RenderLabel(r, x, y, w)
	}

	ix, iy, iw, ih := st.InnerRect(x, y, w, h)
	if iw <= 0 || ih <= 0 {
		return
	}

	// Clamp scroll to keep cursor visible
	if ta.CursorRow < ta.ScrollY {
		ta.ScrollY = ta.CursorRow
	}
	if ta.CursorRow >= ta.ScrollY+ih {
		ta.ScrollY = ta.CursorRow - ih + 1
	}

	// Draw lines
	for row := 0; row < ih; row++ {
		lineIdx := ta.ScrollY + row
		if lineIdx >= len(ta.Lines) {
			break
		}
		line := ta.Lines[lineIdx]
		if len(line) > iw {
			line = line[:iw]
		}
		r.DrawText(ix, iy+row, line, st, iw)
	}

	// Draw cursor
	if ta.Focused {
		cursorScreenRow := ta.CursorRow - ta.ScrollY
		if cursorScreenRow >= 0 && cursorScreenRow < ih && ta.CursorCol < iw {
			ch := ' '
			if ta.CursorCol < len(ta.Lines[ta.CursorRow]) {
				ch = rune(ta.Lines[ta.CursorRow][ta.CursorCol])
			}
			cursorStyle := style.Style{FG: style.CurrentTheme.CursorFG, BG: style.CurrentTheme.CursorBG}
			r.Screen.SetContent(ix+ta.CursorCol, iy+cursorScreenRow, ch, nil, cursorStyle.TcellStyle())
		}
	}
}
