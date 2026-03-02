package main

import (
	"fmt"
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/satyam/reactive-tui/app"
	"github.com/satyam/reactive-tui/signal"
	"github.com/satyam/reactive-tui/style"
	"github.com/satyam/reactive-tui/widget"
)

func main() {
	count := signal.New(0)

	root := &widget.Box{
		Base: widget.Base{
			Style: style.Style{
				Border:  style.BorderRounded,
				Padding: style.Pad(1),
				FG:      tcell.ColorWhite,
			},
			Flex: widget.FlexProps{Grow: 1},
		},
		Dir: widget.Column,
		Gap: 1,
		Items: []widget.Node{
			// Title
			func() widget.Node {
				t := widget.StaticText("Reactive TUI Counter")
				t.Style.Bold = true
				t.Style.FG = tcell.ColorAqua
				return t
			}(),
			// Count display
			widget.BoundText(func() string {
				return fmt.Sprintf("Count: %d", count.Get())
			}),
			// Buttons row
			&widget.Box{
				Dir: widget.Row,
				Gap: 2,
				Items: []widget.Node{
					func() widget.Node {
						b := widget.NewButton("[ - ]", func() {
							count.Update(func(v int) int { return v - 1 })
						})
						b.Flex.Basis = 10
						return b
					}(),
					func() widget.Node {
						b := widget.NewButton("[ + ]", func() {
							count.Update(func(v int) int { return v + 1 })
						})
						b.Flex.Basis = 10
						return b
					}(),
					func() widget.Node {
						b := widget.NewButton("[Reset]", func() {
							count.Set(0)
						})
						b.Flex.Basis = 10
						return b
					}(),
				},
			},
			// Instructions
			func() widget.Node {
				t := widget.StaticText("Tab: switch focus | Enter/Space: press button | Ctrl+C: quit")
				t.Style.FG = tcell.ColorGray
				return t
			}(),
		},
	}

	a := app.New(root)
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
