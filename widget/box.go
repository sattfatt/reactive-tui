package widget

import (
	"github.com/satyam/reactive-tui/layout"
	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/style"
)

type Direction int

const (
	Column Direction = iota
	Row
)

type Box struct {
	Base
	Dir      Direction
	Gap      int
	Justify  style.Justify
	Align    style.Align
	Items    []Node
}

func VBox(children ...Node) *Box {
	return &Box{
		Base: Base{Flex: FlexProps{Basis: -1, Shrink: 1}},
		Dir:  Column, Items: children,
	}
}

func HBox(children ...Node) *Box {
	return &Box{
		Base: Base{Flex: FlexProps{Basis: -1, Shrink: 1}},
		Dir:  Row, Items: children,
	}
}

func (b *Box) Children() []Node { return b.Items }

// FlexProps computes min sizes from children when Basis is auto (-1).
func (b *Box) FlexProps() FlexProps {
	fp := b.Flex
	if fp.Basis >= 0 {
		return fp
	}

	// Compute intrinsic min size from children
	minMain := 0
	minCross := 0
	for _, child := range b.Items {
		cfp := child.FlexProps()
		var childMain, childCross int
		if b.Dir == Column {
			childMain = cfp.MinHeight
			if cfp.Basis > 0 {
				childMain = cfp.Basis
			}
			childCross = cfp.MinWidth
		} else {
			childMain = cfp.MinWidth
			if cfp.Basis > 0 {
				childMain = cfp.Basis
			}
			childCross = cfp.MinHeight
		}
		minMain += childMain
		if childCross > minCross {
			minCross = childCross
		}
	}
	if len(b.Items) > 1 {
		minMain += b.Gap * (len(b.Items) - 1)
	}
	minMain += b.Style.ChromeHeight()
	minCross += b.Style.ChromeWidth()

	if b.Dir == Column {
		if fp.MinHeight < minMain {
			fp.MinHeight = minMain
		}
		if fp.MinWidth < minCross {
			fp.MinWidth = minCross
		}
	} else {
		if fp.MinWidth < minMain {
			fp.MinWidth = minMain
		}
		if fp.MinHeight < minCross {
			fp.MinHeight = minCross
		}
	}
	return fp
}

func (b *Box) Render(r *render.Renderer, x, y, w, h int) {
	b.Base.SetRect(x, y, w, h)
	// Draw border and background
	if b.Style.Border != style.BorderNone {
		r.DrawBorder(x, y, w, h, b.Style.Border, b.Style)
	}

	ix, iy, iw, ih := b.Style.InnerRect(x, y, w, h)
	if iw <= 0 || ih <= 0 {
		return
	}

	// Build flex items
	flexItems := make([]layout.Item, len(b.Items))
	for i, child := range b.Items {
		fp := child.FlexProps()
		flexItems[i] = layout.Item{
			Basis:   fp.Basis,
			Grow:    fp.Grow,
			Shrink:  fp.Shrink,
			MinSize: mainMinSize(fp, b.Dir),
			MaxSize: mainMaxSize(fp, b.Dir),
		}
	}

	// Solve main axis
	mainAvailable := ih
	if b.Dir == Row {
		mainAvailable = iw
	}
	sizes := layout.Solve(flexItems, mainAvailable, b.Gap)

	// Render children
	offset := 0
	for i, child := range b.Items {
		size := sizes[i]
		var cx, cy, cw, ch int
		if b.Dir == Column {
			cx, cy, cw, ch = ix, iy+offset, iw, size
		} else {
			cx, cy, cw, ch = ix+offset, iy, size, ih
		}
		child.Render(r, cx, cy, cw, ch)
		offset += size + b.Gap
	}
}

func mainMinSize(fp FlexProps, dir Direction) int {
	if dir == Column {
		return fp.MinHeight
	}
	return fp.MinWidth
}

func mainMaxSize(fp FlexProps, dir Direction) int {
	if dir == Column {
		return fp.MaxHeight
	}
	return fp.MaxWidth
}
