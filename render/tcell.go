package render

import (
	"github.com/gdamore/tcell/v2"
	"github.com/satyam/reactive-tui/style"
)

type Renderer struct {
	Screen tcell.Screen
}

func New(screen tcell.Screen) *Renderer {
	return &Renderer{Screen: screen}
}

func (r *Renderer) Size() (int, int) {
	return r.Screen.Size()
}

func (r *Renderer) Clear() {
	r.Screen.Clear()
}

func (r *Renderer) Show() {
	r.Screen.Show()
}

// DrawText renders a string at (x, y), clipped to maxWidth.
func (r *Renderer) DrawText(x, y int, text string, st style.Style, maxWidth int) {
	ts := st.TcellStyle()
	col := 0
	for _, ch := range text {
		if col >= maxWidth {
			break
		}
		r.Screen.SetContent(x+col, y, ch, nil, ts)
		col++
	}
}

// FillRect fills a rectangle with a character and style.
func (r *Renderer) FillRect(x, y, w, h int, ch rune, st style.Style) {
	ts := st.TcellStyle()
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			r.Screen.SetContent(col, row, ch, nil, ts)
		}
	}
}

// DrawBorder draws a border around the given rect.
func (r *Renderer) DrawBorder(x, y, w, h int, border style.BorderStyle, st style.Style) {
	if border == style.BorderNone || w < 2 || h < 2 {
		return
	}
	chars := border.Chars()
	tl, tr, bl, br, horiz, vert := chars[0], chars[1], chars[2], chars[3], chars[4], chars[5]
	ts := st.TcellStyle()

	// Corners
	r.Screen.SetContent(x, y, tl, nil, ts)
	r.Screen.SetContent(x+w-1, y, tr, nil, ts)
	r.Screen.SetContent(x, y+h-1, bl, nil, ts)
	r.Screen.SetContent(x+w-1, y+h-1, br, nil, ts)

	// Top and bottom edges
	for col := x + 1; col < x+w-1; col++ {
		r.Screen.SetContent(col, y, horiz, nil, ts)
		r.Screen.SetContent(col, y+h-1, horiz, nil, ts)
	}

	// Left and right edges
	for row := y + 1; row < y+h-1; row++ {
		r.Screen.SetContent(x, row, vert, nil, ts)
		r.Screen.SetContent(x+w-1, row, vert, nil, ts)
	}
}
