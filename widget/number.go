package widget

import (
	"fmt"
	"strconv"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/sattfatt/reactive-tui/render"
	"github.com/sattfatt/reactive-tui/style"
)

// NumberInput is a numeric stepper widget. In navigation mode, Up/Down
// (or k/j) increment and decrement the value by Step. Pressing 'i' enters
// edit mode where digits can be typed directly; Escape exits edit mode and
// parses the typed value.
type NumberInput struct {
	Base
	Value    int
	Min      int
	Max      int
	HasMin   bool
	HasMax   bool
	Step     int
	OnChange func(int)

	// edit mode state
	editBuf    string
	editCursor int
	prevValue  int // for reverting on invalid input
}

// NewNumberInput creates a number input with a default step of 1.
func NewNumberInput(value int, onChange func(int)) *NumberInput {
	return &NumberInput{
		Base: Base{
			Style: style.Style{FG: style.CurrentTheme.FG, BG: style.CurrentTheme.BG, Border: style.BorderSingle},
			Flex:  FlexProps{Basis: -1, Shrink: 1, MinHeight: 3, MaxHeight: 3, MinWidth: 12},
		},
		Value:    value,
		Step:     1,
		OnChange: onChange,
	}
}

// WithRange sets minimum and maximum bounds.
func (n *NumberInput) WithRange(min, max int) *NumberInput {
	n.Min = min
	n.Max = max
	n.HasMin = true
	n.HasMax = true
	n.Value = n.clamp(n.Value)
	return n
}

// WithMin sets a minimum bound only.
func (n *NumberInput) WithMin(min int) *NumberInput {
	n.Min = min
	n.HasMin = true
	n.Value = n.clamp(n.Value)
	return n
}

// WithMax sets a maximum bound only.
func (n *NumberInput) WithMax(max int) *NumberInput {
	n.Max = max
	n.HasMax = true
	n.Value = n.clamp(n.Value)
	return n
}

func (n *NumberInput) Focusable() bool  { return true }
func (n *NumberInput) IsEditable() bool { return true }

func (n *NumberInput) clamp(v int) int {
	if n.HasMin && v < n.Min {
		return n.Min
	}
	if n.HasMax && v > n.Max {
		return n.Max
	}
	return v
}

func (n *NumberInput) setValue(v int) {
	v = n.clamp(v)
	if v != n.Value {
		n.Value = v
		if n.OnChange != nil {
			n.OnChange(n.Value)
		}
	}
}

// SetEditing overrides Base to initialize/finalize the edit buffer.
func (n *NumberInput) SetEditing(editing bool) {
	if editing && !n.Editing {
		// Entering edit mode: snapshot value into buffer
		n.prevValue = n.Value
		n.editBuf = strconv.Itoa(n.Value)
		n.editCursor = len(n.editBuf)
	} else if !editing && n.Editing {
		// Exiting edit mode: parse buffer
		n.commitEdit()
	}
	n.Editing = editing
}

func (n *NumberInput) commitEdit() {
	if n.editBuf == "" || n.editBuf == "-" {
		// Invalid — revert
		n.Value = n.prevValue
		return
	}
	v, err := strconv.Atoi(n.editBuf)
	if err != nil {
		n.Value = n.prevValue
		return
	}
	n.setValue(v)
}

func (n *NumberInput) HandleKey(ev KeyEvent) bool {
	if n.Editing {
		return n.handleEditKey(ev)
	}
	return n.handleNavKey(ev)
}

func (n *NumberInput) handleNavKey(ev KeyEvent) bool {
	switch {
	case ev.Key == int(tcell.KeyUp) || ev.Rune == 'k':
		n.setValue(n.Value + n.Step)
		return true
	case ev.Key == int(tcell.KeyDown) || ev.Rune == 'j':
		n.setValue(n.Value - n.Step)
		return true
	}
	return false
}

func (n *NumberInput) handleEditKey(ev KeyEvent) bool {
	switch tcell.Key(ev.Key) {
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if n.editCursor > 0 {
			n.editBuf = n.editBuf[:n.editCursor-1] + n.editBuf[n.editCursor:]
			n.editCursor--
		}
		return true
	case tcell.KeyDelete:
		if n.editCursor < len(n.editBuf) {
			n.editBuf = n.editBuf[:n.editCursor] + n.editBuf[n.editCursor+1:]
		}
		return true
	case tcell.KeyLeft:
		if n.editCursor > 0 {
			n.editCursor--
		}
		return true
	case tcell.KeyRight:
		if n.editCursor < len(n.editBuf) {
			n.editCursor++
		}
		return true
	case tcell.KeyEnter:
		n.commitEdit()
		return true
	case tcell.KeyRune:
		if unicode.IsDigit(ev.Rune) {
			n.editBuf = n.editBuf[:n.editCursor] + string(ev.Rune) + n.editBuf[n.editCursor:]
			n.editCursor++
			return true
		}
		if ev.Rune == '-' {
			// Toggle sign: add or remove leading minus
			if len(n.editBuf) > 0 && n.editBuf[0] == '-' {
				n.editBuf = n.editBuf[1:]
				if n.editCursor > 0 {
					n.editCursor--
				}
			} else {
				n.editBuf = "-" + n.editBuf
				n.editCursor++
			}
			return true
		}
		return true // consume other runes silently in edit mode
	}
	return false
}

func (n *NumberInput) Render(r *render.Renderer, x, y, w, h int) {
	n.Base.SetRect(x, y, w, h)
	st := n.Style

	// Border with themed color
	borderSt := st
	if n.Focused {
		if n.Editing {
			borderSt.FG = style.CurrentTheme.EditFocusFG
		} else {
			borderSt.FG = style.CurrentTheme.NavFocusFG
		}
	} else {
		borderSt.FG = style.CurrentTheme.BorderFG
	}
	r.DrawBorder(x, y, w, h, st.Border, borderSt)
	n.Base.RenderLabel(r, x, y, w)

	ix, iy, iw, _ := st.InnerRect(x, y, w, h)
	if iw <= 0 {
		return
	}

	// Build display string
	var display string
	if n.Editing {
		display = n.editBuf
		if display == "" {
			display = "_"
		}
	} else {
		display = fmt.Sprintf("%d", n.Value)
	}

	// Arrows + value: ◀ value ▶
	arrow := "◀ "
	arrowR := " ▶"
	full := arrow + display + arrowR
	if len(full) > iw {
		// No room for arrows, just show the number
		full = display
		if len(full) > iw {
			full = full[:iw]
		}
	}

	// Center horizontally
	left := max((iw-len(full))/2, 0)

	// Draw arrows in muted color, value in normal color
	arrowSt := st
	arrowSt.FG = style.CurrentTheme.MutedFG
	valSt := st

	cx := ix + left
	if len(arrow)+len(display)+len(arrowR) <= iw {
		// Draw left arrow
		r.DrawText(cx, iy, arrow, arrowSt, len(arrow))
		cx += len(arrow)
		// Draw value
		r.DrawText(cx, iy, display, valSt, len(display))
		valStart := cx
		cx += len(display)
		// Draw right arrow
		r.DrawText(cx, iy, arrowR, arrowSt, len(arrowR))

		// Draw cursor in edit mode
		if n.Editing && n.Focused {
			cursorPos := valStart + n.editCursor
			if cursorPos < ix+iw {
				ch := ' '
				if n.editCursor < len(n.editBuf) {
					ch = rune(n.editBuf[n.editCursor])
				}
				cursorStyle := style.Style{FG: style.CurrentTheme.CursorFG, BG: style.CurrentTheme.CursorBG}
				r.Screen.SetContent(cursorPos, iy, ch, nil, cursorStyle.TcellStyle())
			}
		}
	} else {
		// Fallback: just the number
		r.DrawText(cx, iy, full, valSt, len(full))
	}
}
