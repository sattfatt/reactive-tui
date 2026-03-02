# reactive-tui

A reactive terminal UI framework for Go with vim-modal navigation, flexbox layout, reactive signals, and embedded neovim support.

## Architecture

```
signal/     Reactive state primitives (Signal[T], ListSignal[T], Batch)
style/      Theming, border styles, spacing, color roles
render/     Low-level tcell screen abstraction (text, borders, fill)
layout/     1D flexbox solver (basis → grow/shrink → clamp)
widget/     UI components implementing the Node interface
app/        Application runtime, event loop, focus management, modal navigation
```

### Reactive Signal System (`signal/`)

Signals are the core reactivity primitive. When a signal's value changes, it notifies watchers and triggers a full UI re-render.

```go
count := signal.New(0)           // Signal[int]
count.Set(5)                     // triggers watchers + re-render
count.Update(func(v int) int {   // atomic read-modify-write
    return v + 1
})
count.Get()                      // read current value (thread-safe)
count.Watch(func(v int) { ... }) // register change callback
```

`ListSignal[T]` works the same way for slices (which aren't comparable, so every `Set` fires). `signal.Batch(fn)` defers re-render notifications until the batch completes, preventing redundant renders during bulk updates.

**How it drives rendering:** `signal.OnChange` is a global `func()` that the App sets to push into its rerender channel. Any `Signal.Set()` call fires this hook, which wakes the App's main loop to re-collect focusables (the widget tree may have changed via `Dynamic` nodes) and re-render.

### Node Interface (`widget/node.go`)

Every widget implements `Node`:

```go
type Node interface {
    Render(r *render.Renderer, x, y, w, h int)  // draw into allocated rect
    FlexProps() FlexProps                         // layout sizing hints
    GetStyle() style.Style                        // border, padding, colors
    Children() []Node                             // child widgets (for containers)
    Focusable() bool                              // can receive focus?
    HandleKey(ev KeyEvent) bool                   // consume key event?
    SetFocused(bool)                              // focus state callback
}
```

**Optional interfaces** a widget can implement:

| Interface | Method | Purpose |
|---|---|---|
| `Editable` | `IsEditable() bool` | Can enter edit mode with `i` |
| `EscapeThreshold` | `EscapesToExit() int` | Escapes needed to exit edit mode (default 1, Terminal/Neovim use 3) |

`Base` is the shared struct embedded by all widgets. It holds `Style`, `Flex` (sizing), `Label` (badge drawn on border), `Focused`, `Editing`, and `LayoutRect` (populated during render for spatial focus navigation).

### Flexbox Layout (`layout/`)

Layout uses a 1D flexbox algorithm identical in spirit to CSS flexbox:

```go
type FlexProps struct {
    Basis     int  // desired size (-1 = auto/use MinHeight)
    Grow      int  // share of extra space
    Shrink    int  // share of deficit
    MinHeight int
    MaxHeight int
    MinWidth  int
    MaxWidth  int
}
```

`layout.Solve(items, available, gap)` resolves sizes in one pass: assign basis → distribute surplus via grow factors → shrink proportionally when over-constrained → clamp to min/max. Boxes with `Basis: -1` compute intrinsic minimum sizes from their children recursively.

### Style & Theming (`style/`)

`style.Style` composes border type, padding, margin, FG/BG colors, and bold/italic flags. `InnerRect()` computes the content area after border + padding. `TcellStyle()` converts to tcell's native style type.

`style.Theme` defines 15+ semantic color roles. `style.CurrentTheme` is the global active theme (default: `TokyoNight()`). Widgets reference theme colors for borders, focus states, selection highlighting, cursor, buttons, progress bars, and muted text.

### App Runtime (`app/`)

The App owns the tcell screen, the render loop, and the two-mode event model.

**Main loop:**
1. Poll `tcell.PollEvent()` in a goroutine, push into `eventCh`
2. `select` on `eventCh` or `rerender` channel
3. On event: `handleEvent()` dispatches based on current mode
4. On rerender: `collectFocusables()` (tree may have changed), then `render()`
5. `render()` clears screen, calls `Root.Render()` (or `overlay.Render()` if set), then `Show()`

**Two navigation modes:**

| Mode | Trigger | Behavior |
|---|---|---|
| `ModeNavigation` | Default / Escape | `hjkl` spatial focus, `Tab`/`Shift+Tab` cycle, `i` enters edit mode on editable widgets |
| `ModeEdit` | Press `i` on editable widget | All input forwarded to focused widget. Escape exits (1× for inputs, 3× for terminals) |

`Ctrl+N` / `Ctrl+P` are remapped to Down/Up before mode dispatch.

**Focus management:** `collectFocusables()` does a depth-first traversal of the widget tree, collecting all `Focusable()` nodes into a flat list. Focus is preserved across tree changes by pointer identity. Spatial navigation (`hjkl`) uses dot-product direction checks + Euclidean distance to find the nearest widget.

**Overlay system:** When `app.overlay` is set (e.g., via `OpenNeovimPopup`), it renders full-screen instead of Root, and the focus ring traverses the overlay tree.

## Widgets

### Containers

**Box** — Flexbox container. `VBox(children...)` / `HBox(children...)`. Has `Gap`, `Justify`, `Align`. Themed borders when focused.

**Tabs** — Tab switcher. Only the active tab's content is in the focus ring. Left/Right or h/l switches tabs. Each tab is `Tab{Label, Content}`.

**Dynamic** — Conditional rendering. Wraps a `func() Node` that can return nil. When nil, collapses to zero size so flex layout reclaims space. Use with signals for toggling panels.

### Inputs

**Input** — Single-line text field. Editable. Has `Value`, `Cursor`, `OnChange`, `Placeholder`.

**TextArea** — Multi-line editor. Editable. Has `Lines []string`, `CursorRow/Col`, `ScrollY`, `OnChange`. Full cursor movement with line-end wrapping.

**FuzzyFinder** — Interactive fuzzy search over a string list. Editable. Scores by consecutive matches + word boundaries. Has `OnSelect(index)`.

### Display

**Text** — Static or reactive text. `StaticText("hello")` or `BoundText(func() string { return signal.Get() })`. Supports `ScrollBottom` for log-style auto-scroll.

**ProgressBar** — 0.0–1.0 bar with filled/empty characters and optional percentage label. Non-focusable.

**Button** — Clickable with Enter/Space. Themed focus/unfocus states. `NewButton(label, onClick)`.

**Checkbox** — Boolean toggle. Space/Enter toggles in nav mode (no edit mode needed). `NewCheckbox(label, onChange)`. Renders `[✓]` or `[ ]` with label.

**NumberInput** — Integer stepper with optional min/max range. Nav mode: Up/Down (or k/j) increment/decrement by `Step`. Edit mode (`i`): type digits directly, `-` toggles sign. `NewNumberInput(value, onChange).WithRange(min, max)`. Renders `◀ 42 ▶` with themed arrows.

**List** — Selectable item list with Up/Down/Enter navigation. Items provided as `func() []string` for reactivity.

**Table** — Data grid with fixed-width and grow columns. Header row + separator + scrollable body. `OnSelect(rowIndex)`.

### Rich

**Terminal** — Full PTY terminal emulator. Runs any command (shell, TUI app). ANSI/VT100 parser with SGR colors, alternate buffer, scrolling regions, UTF-8. Needs 3× Escape to exit edit mode. Call `Start()` before use, `Stop()` to cleanup.

**NeovimEditor** — Embeds neovim inside a Terminal widget. Creates a temp file with the correct extension (e.g., `.go`) so neovim's filetype detection and LSP attach automatically. When neovim exits (`:wq`), reads the file back and fires `OnChange(content)`. Supports zen mode (hides nvim chrome). Preview mode shows content as static text when neovim isn't running; press `i` to launch.

## Usage Pattern

```go
package main

import (
    "fmt"
    "github.com/sattfatt/reactive-tui/app"
    "github.com/sattfatt/reactive-tui/signal"
    "github.com/sattfatt/reactive-tui/style"
    "github.com/sattfatt/reactive-tui/widget"
)

func main() {
    // 1. Create signals (reactive state)
    count := signal.New(0)

    // 2. Build widget tree with signal bindings
    display := widget.BoundText(func() string {
        return fmt.Sprintf("Count: %d", count.Get())
    })

    inc := widget.NewButton("+", func() {
        count.Update(func(v int) int { return v + 1 })
    })

    root := widget.VBox(display, inc)
    root.Style.Border = style.BorderRounded
    root.Style.Padding = style.Pad(1)
    root.Gap = 1

    // 3. Run
    a := app.New(root)
    a.Run()
}
```

Key points:
- Signals drive reactivity. Bind widget content to `signal.Get()` calls inside closures.
- `signal.Set()` automatically triggers re-render — no manual invalidation needed.
- Layout is declarative via flex props. Set `Flex.Grow`, `Flex.Basis`, `Flex.Shrink` on widgets.
- Containers (`VBox`/`HBox`) handle layout. Nest them for complex UIs.
- Modal navigation is automatic. Focusable widgets get vim-style hjkl navigation. Editable widgets enter edit mode with `i`.

## Dependencies

| Package | Purpose |
|---|---|
| `github.com/gdamore/tcell/v2` | Terminal screen, events, truecolor |
| `github.com/creack/pty` | PTY management for Terminal widget |
| `github.com/mattn/go-runewidth` | Unicode character width |
| `golang.org/x/term` | Terminal control |

## Examples

- `examples/counter/` — Minimal: one signal, three buttons, bound text
- `examples/kitchen/` — All widgets: counter + progress bar, tabbed views (table, textarea, fuzzy finder, terminal, neovim), event log, dynamic sidebar toggle
