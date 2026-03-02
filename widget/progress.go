package widget

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/sattfatt/reactive-tui/render"
	"github.com/sattfatt/reactive-tui/style"
)

type ProgressBar struct {
	Base
	Value      func() float64 // 0.0–1.0, reactive getter
	FilledChar rune
	EmptyChar  rune
	FilledFG   tcell.Color
	EmptyFG    tcell.Color
	ShowLabel  bool
}

func NewProgressBar(value func() float64) *ProgressBar {
	return &ProgressBar{
		Base: Base{
			Flex: FlexProps{Basis: -1, Grow: 1, Shrink: 1, MinHeight: 1, MaxHeight: 1, MinWidth: 10},
		},
		Value:      value,
		FilledChar: '█',
		EmptyChar:  '░',
		FilledFG:   style.CurrentTheme.ProgressFilledFG,
		EmptyFG:    style.CurrentTheme.ProgressEmptyFG,
		ShowLabel:  true,
	}
}

func (p *ProgressBar) Render(r *render.Renderer, x, y, w, h int) {
	p.Base.SetRect(x, y, w, h)
	if p.Style.Border != style.BorderNone {
		r.DrawBorder(x, y, w, h, p.Style.Border, p.Style)
		p.Base.RenderLabel(r, x, y, w)
	}

	ix, iy, iw, ih := p.Style.InnerRect(x, y, w, h)
	if iw <= 0 || ih <= 0 {
		return
	}

	v := p.Value()
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}

	barWidth := iw
	label := ""
	if p.ShowLabel {
		label = fmt.Sprintf(" %3.0f%%", v*100)
		barWidth -= len(label)
		if barWidth < 1 {
			barWidth = iw // no room for label, skip it
			label = ""
		}
	}

	filled := int(float64(barWidth) * v)
	if filled > barWidth {
		filled = barWidth
	}

	filledStyle := p.Style
	filledStyle.FG = p.FilledFG
	emptyStyle := p.Style
	emptyStyle.FG = p.EmptyFG

	for col := 0; col < barWidth; col++ {
		ch := p.EmptyChar
		st := emptyStyle
		if col < filled {
			ch = p.FilledChar
			st = filledStyle
		}
		r.Screen.SetContent(ix+col, iy, ch, nil, st.TcellStyle())
	}

	// Draw label
	if label != "" {
		labelStyle := p.Style
		labelStyle.FG = style.CurrentTheme.FG
		r.DrawText(ix+barWidth, iy, label, labelStyle, len(label))
	}
}
