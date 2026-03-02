package widget

import (
	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/style"
)

// Node is the interface all widgets implement.
type Node interface {
	// Render draws the widget into the given rect.
	Render(r *render.Renderer, x, y, w, h int)

	// FlexProps returns the flex layout properties for this node.
	FlexProps() FlexProps

	// Style returns the style for border/padding calculations.
	GetStyle() style.Style

	// Children returns child nodes (nil for leaf widgets).
	Children() []Node

	// Focusable returns true if this widget can receive focus.
	Focusable() bool

	// HandleKey processes a key event. Returns true if consumed.
	HandleKey(ev KeyEvent) bool

	// SetFocused sets the focus state.
	SetFocused(bool)
}

type KeyEvent struct {
	Key  int
	Rune rune
	Mod  int // tcell.ModMask cast to int
}

// Rect stores the last rendered position and size.
type Rect struct {
	X, Y, W, H int
}

type FlexProps struct {
	Grow      float64
	Shrink    float64
	Basis     int  // -1 means auto (use content size)
	MinWidth  int
	MinHeight int
	MaxWidth  int  // 0 means no max
	MaxHeight int  // 0 means no max
}

func DefaultFlexProps() FlexProps {
	return FlexProps{
		Grow:   0,
		Shrink: 1,
		Basis:  -1,
	}
}

// Base is embedded by widgets for default implementations.
type Base struct {
	Style   style.Style
	Flex    FlexProps
	Focused bool
	LayoutRect Rect // populated during Render
}

func (b *Base) FlexProps() FlexProps    { return b.Flex }
func (b *Base) GetStyle() style.Style   { return b.Style }
func (b *Base) Children() []Node        { return nil }
func (b *Base) Focusable() bool         { return false }
func (b *Base) HandleKey(KeyEvent) bool { return false }
func (b *Base) SetFocused(f bool)       { b.Focused = f }
func (b *Base) SetRect(x, y, w, h int) { b.LayoutRect = Rect{x, y, w, h} }
func (b *Base) GetRect() Rect           { return b.LayoutRect }
