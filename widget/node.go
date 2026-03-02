package widget

import (
	"github.com/sattfatt/reactive-tui/render"
	"github.com/sattfatt/reactive-tui/style"
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

// Editable is implemented by widgets that accept text input (Input, TextArea,
// Terminal, FuzzyFinder). The app enters Edit mode when the user presses 'i'
// on an Editable widget.
type Editable interface {
	IsEditable() bool
}

// EscapeThreshold is optionally implemented by Editable widgets that use
// Escape internally (e.g., terminal/vim). Returns the number of consecutive
// Escape presses needed to exit Edit mode. Default is 1 if not implemented.
type EscapeThreshold interface {
	EscapesToExit() int
}

// Refreshable is implemented by widgets that cache state between frames
// (e.g., Dynamic). The app calls Refresh() on all Refreshable widgets in
// the tree before collecting focusables, so that the same widget pointers
// are used for focus tracking and rendering.
type Refreshable interface {
	Refresh()
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
	Style      style.Style
	Flex       FlexProps
	Label      string // optional label drawn on the top border
	Focused    bool
	Editing    bool // set by app when in edit mode on this widget
	LayoutRect Rect // populated during Render
}

func (b *Base) FlexProps() FlexProps    { return b.Flex }
func (b *Base) GetStyle() style.Style   { return b.Style }
func (b *Base) Children() []Node        { return nil }
func (b *Base) Focusable() bool         { return false }
func (b *Base) HandleKey(KeyEvent) bool { return false }
func (b *Base) SetFocused(f bool)       { b.Focused = f }
func (b *Base) SetEditing(e bool)       { b.Editing = e }
func (b *Base) SetRect(x, y, w, h int) { b.LayoutRect = Rect{x, y, w, h} }
func (b *Base) GetRect() Rect           { return b.LayoutRect }

// RenderLabel draws the Base.Label as a badge on the top-right of the border.
// Call this immediately after DrawBorder. It's a no-op if Label is empty or
// there's no border.
func (b *Base) RenderLabel(r *render.Renderer, x, y, w int) {
	if b.Label == "" || b.Style.Border == style.BorderNone {
		return
	}
	badge := " " + b.Label + " "
	if w <= len(badge)+4 {
		return // not enough room
	}
	badgeSt := style.Style{
		FG:   style.CurrentTheme.BG,
		BG:   style.CurrentTheme.ButtonFG,
		Bold: true,
	}
	bx := x + w - len(badge) - 2 // padding from corner
	r.DrawText(bx, y, badge, badgeSt, len(badge))
}
