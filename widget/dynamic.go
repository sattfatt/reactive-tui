package widget

import (
	"github.com/sattfatt/reactive-tui/render"
	"github.com/sattfatt/reactive-tui/style"
)

// Dynamic is a widget that delegates to whatever node a function returns.
// This enables conditional rendering based on signal state. If the function
// returns nil, the widget renders nothing and is not focusable.
//
// The Content function is called once on first access and the result is
// cached. To rebuild the widget tree (e.g., when a selection changes),
// call Invalidate(). This prevents signal-triggered re-renders from
// destroying widget state (editing, focus, cursor position) by accidentally
// rebuilding the tree.
//
// Example:
//
//	sidebar := widget.NewDynamic(func() widget.Node {
//	    if showPanel.Get() {
//	        return panel
//	    }
//	    return nil
//	})
//
//	toggleBtn := widget.NewButton("Toggle", func() {
//	    showPanel.Update(func(v bool) bool { return !v })
//	    sidebar.Invalidate() // rebuild on next access
//	})
type Dynamic struct {
	Base
	Content func() Node
	cached  Node
	valid   bool // true once Content() has been called and result cached
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

// Invalidate marks the cached content as stale. The next call to any
// method (Children, Render, etc.) will re-evaluate Content().
// Call this when the data the Content function depends on has changed
// and you want the widget tree to be rebuilt.
func (d *Dynamic) Invalidate() {
	d.valid = false
}

// Refresh re-evaluates the Content function and caches the result.
// Implements the Refreshable interface for backward compatibility,
// but prefer Invalidate() for explicit control.
func (d *Dynamic) Refresh() {
	if d.Content != nil {
		d.cached = d.Content()
	} else {
		d.cached = nil
	}
	d.valid = true
}

func (d *Dynamic) current() Node {
	if !d.valid {
		d.Refresh()
	}
	return d.cached
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
