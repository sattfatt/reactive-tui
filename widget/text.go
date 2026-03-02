package widget

import (
	"strings"

	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/style"
)

type Text struct {
	Base
	Content      func() string // reactive getter
	ScrollBottom bool          // if true, show last lines instead of first
}

// StaticText creates a text widget with a fixed string.
func StaticText(s string) *Text {
	return &Text{
		Base:    Base{Flex: FlexProps{Basis: -1, Shrink: 1, MinHeight: 1, MinWidth: 1}},
		Content: func() string { return s },
	}
}

// BoundText creates a text widget bound to a signal's Get method.
func BoundText(getter func() string) *Text {
	return &Text{
		Base:    Base{Flex: FlexProps{Basis: -1, Shrink: 1, MinHeight: 1, MinWidth: 1}},
		Content: getter,
	}
}

func (t *Text) Render(r *render.Renderer, x, y, w, h int) {
	t.Base.SetRect(x, y, w, h)
	if t.Style.Border != style.BorderNone {
		r.DrawBorder(x, y, w, h, t.Style.Border, t.Style)
	}

	ix, iy, iw, ih := t.Style.InnerRect(x, y, w, h)
	if iw <= 0 || ih <= 0 {
		return
	}

	text := t.Content()
	lines := WrapText(text, iw)

	// If ScrollBottom, show the last ih lines
	start := 0
	if t.ScrollBottom && len(lines) > ih {
		start = len(lines) - ih
	}

	for i := start; i < len(lines) && i-start < ih; i++ {
		r.DrawText(ix, iy+i-start, lines[i], t.Style, iw)
	}
}

// WrapText splits text into lines that fit within width, breaking at spaces.
func WrapText(text string, width int) []string {
	if width <= 0 {
		return nil
	}
	var lines []string
	for _, raw := range strings.Split(text, "\n") {
		if len(raw) <= width {
			lines = append(lines, raw)
			continue
		}
		for len(raw) > width {
			// Try to break at space
			breakAt := width
			for i := width - 1; i > 0; i-- {
				if raw[i] == ' ' {
					breakAt = i
					break
				}
			}
			lines = append(lines, raw[:breakAt])
			raw = strings.TrimLeft(raw[breakAt:], " ")
		}
		if len(raw) > 0 {
			lines = append(lines, raw)
		}
	}
	return lines
}
