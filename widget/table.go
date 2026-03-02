package widget

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/style"
)

type TableColumn struct {
	Header string
	Width  int     // fixed width; 0 means proportional
	Grow   float64 // proportion of remaining space
}

type Table struct {
	Base
	Columns  []TableColumn
	Rows     func() [][]string
	Selected int
	ScrollY  int
	OnSelect func(int)
}

func NewTable(columns []TableColumn, rows func() [][]string, onSelect func(int)) *Table {
	return &Table{
		Base: Base{
			Style: style.Style{FG: tcell.ColorWhite, BG: tcell.ColorDefault, Border: style.BorderSingle},
			Flex:  FlexProps{Basis: -1, Grow: 1, Shrink: 1, MinHeight: 4, MinWidth: 20},
		},
		Columns:  columns,
		Rows:     rows,
		OnSelect: onSelect,
	}
}

func (t *Table) Focusable() bool { return true }

// resolveColumnWidths computes the actual width of each column.
func (t *Table) resolveColumnWidths(totalWidth int) []int {
	widths := make([]int, len(t.Columns))
	fixedUsed := 0
	totalGrow := 0.0
	for i, col := range t.Columns {
		if col.Width > 0 {
			widths[i] = col.Width
			fixedUsed += col.Width
		} else {
			g := col.Grow
			if g <= 0 {
				g = 1
			}
			totalGrow += g
		}
	}
	// Add separator space between columns
	separatorSpace := 0
	if len(t.Columns) > 1 {
		separatorSpace = len(t.Columns) - 1
	}
	remaining := totalWidth - fixedUsed - separatorSpace
	if remaining < 0 {
		remaining = 0
	}
	if totalGrow > 0 {
		for i, col := range t.Columns {
			if col.Width == 0 {
				g := col.Grow
				if g <= 0 {
					g = 1
				}
				widths[i] = int(float64(remaining) * g / totalGrow)
				if widths[i] < 1 {
					widths[i] = 1
				}
			}
		}
	}
	return widths
}

func (t *Table) HandleKey(ev KeyEvent) bool {
	rows := t.Rows()
	switch tcell.Key(ev.Key) {
	case tcell.KeyUp:
		if t.Selected > 0 {
			t.Selected--
		}
		return true
	case tcell.KeyDown:
		if t.Selected < len(rows)-1 {
			t.Selected++
		}
		return true
	case tcell.KeyEnter:
		if t.OnSelect != nil && t.Selected < len(rows) {
			t.OnSelect(t.Selected)
		}
		return true
	}
	return false
}

func (t *Table) Render(r *render.Renderer, x, y, w, h int) {
	t.Base.SetRect(x, y, w, h)
	st := t.Style

	if st.Border != style.BorderNone {
		r.DrawBorder(x, y, w, h, st.Border, st)
	}

	ix, iy, iw, ih := st.InnerRect(x, y, w, h)
	if iw <= 0 || ih <= 0 {
		return
	}

	colWidths := t.resolveColumnWidths(iw)

	// Draw header
	drawRow := func(rowY int, cells []string, rowStyle style.Style, fill bool) {
		if fill {
			r.FillRect(ix, rowY, iw, 1, ' ', rowStyle)
		}
		colX := 0
		for i, cell := range cells {
			if i >= len(colWidths) {
				break
			}
			cw := colWidths[i]
			text := cell
			if len(text) > cw {
				text = text[:cw]
			}
			r.DrawText(ix+colX, rowY, text, rowStyle, cw)
			colX += cw + 1 // +1 for separator
		}
	}

	headerStyle := st
	headerStyle.Bold = true
	headers := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		headers[i] = col.Header
	}
	drawRow(iy, headers, headerStyle, false)

	// Draw separator
	if ih > 1 {
		sep := strings.Repeat("─", iw)
		r.DrawText(ix, iy+1, sep, st, iw)
	}

	// Draw data rows
	rows := t.Rows()
	dataStartY := 2 // header + separator

	// Clamp scroll
	visibleRows := ih - dataStartY
	if visibleRows < 1 {
		return
	}
	if t.Selected < t.ScrollY {
		t.ScrollY = t.Selected
	}
	if t.Selected >= t.ScrollY+visibleRows {
		t.ScrollY = t.Selected - visibleRows + 1
	}

	for i := 0; i < visibleRows; i++ {
		rowIdx := t.ScrollY + i
		if rowIdx >= len(rows) {
			break
		}
		rowStyle := st
		fill := false
		if rowIdx == t.Selected && t.Focused {
			rowStyle.FG = tcell.ColorBlack
			rowStyle.BG = tcell.ColorWhite
			fill = true
		}
		drawRow(iy+dataStartY+i, rows[rowIdx], rowStyle, fill)
	}
}
