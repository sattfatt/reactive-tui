package widget

import (
	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/style"
)

// Dynamic is a widget that delegates to whatever node a function returns.
// This enables conditional rendering based on signal state. If the function
// returns nil, the widget renders nothing and is not focusable.
//
// Example:
//
//	widget.Dynamic(func() widget.Node {
//	    if showPanel.Get() {
//	        return panel
//	    }
//	    return nil
//	})
type Dynamic struct {
	Base
	Content func() Node
}

// NewDynamic creates a dynamic widget that renders whatever Content returns.
func NewDynamic(content func() Node) *Dynamic {
	return &Dynamic{
		Base: Base{
			Flex: FlexProps{Basis: -1, Grow: 1, Shrink: 1},
		},
		Content: content,
	}
}

func (d *Dynamic) current() Node {
	if d.Content != nil {
		return d.Content()
	}
	return nil
}

func (d *Dynamic) Focusable() bool {
	if n := d.current(); n != nil {
		return n.Focusable()
	}
	return false
}

func (d *Dynamic) Children() []Node {
	if n := d.current(); n != nil {
		return []Node{n}
	}
	return nil
}

func (d *Dynamic) HandleKey(ev KeyEvent) bool {
	if n := d.current(); n != nil {
		return n.HandleKey(ev)
	}
	return false
}

func (d *Dynamic) FlexProps() FlexProps {
	if n := d.current(); n != nil {
		fp := n.FlexProps()
		// Inherit grow/shrink from Dynamic if the child doesn't set them
		if d.Flex.Grow > 0 && fp.Grow == 0 {
			fp.Grow = d.Flex.Grow
		}
		if d.Flex.Shrink > 0 && fp.Shrink == 0 {
			fp.Shrink = d.Flex.Shrink
		}
		return fp
	}
	// When nil, return zero-size flex so it collapses
	return FlexProps{Basis: 0, Grow: 0, Shrink: 1}
}

func (d *Dynamic) GetStyle() style.Style {
	if n := d.current(); n != nil {
		return n.GetStyle()
	}
	return d.Style
}

func (d *Dynamic) Render(r *render.Renderer, x, y, w, h int) {
	d.Base.SetRect(x, y, w, h)
	if n := d.current(); n != nil {
		n.Render(r, x, y, w, h)
	}
}

func (d *Dynamic) SetFocused(f bool) {
	d.Focused = f
	if n := d.current(); n != nil {
		n.SetFocused(f)
	}
}
