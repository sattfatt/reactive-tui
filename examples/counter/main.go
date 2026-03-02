package main

import (
	"fmt"
	"log"

	"github.com/sattfatt/reactive-tui/app"
	"github.com/sattfatt/reactive-tui/signal"
	"github.com/sattfatt/reactive-tui/style"
	"github.com/sattfatt/reactive-tui/widget"
)

func main() {
	count := signal.New(0)

	decBtn := widget.NewButton("[ - ]", func() {
		count.Update(func(v int) int { return v - 1 })
	})
	decBtn.Flex.Basis = 10

	incBtn := widget.NewButton("[ + ]", func() {
		count.Update(func(v int) int { return v + 1 })
	})
	incBtn.Flex.Basis = 10

	resetBtn := widget.NewButton("[Reset]", func() {
		count.Set(0)
	})
	resetBtn.Flex.Basis = 10

	btnRow := widget.HBox(decBtn, incBtn, resetBtn)
	btnRow.Gap = 2

	title := widget.StaticText("Reactive TUI Counter")
	title.Style.Bold = true
	title.Style.FG = style.CurrentTheme.NavFocusFG

	countDisplay := widget.BoundText(func() string {
		return fmt.Sprintf("Count: %d", count.Get())
	})

	help := widget.StaticText("Tab: switch focus | Enter/Space: press button | Ctrl+C: quit")
	help.Style.FG = style.CurrentTheme.MutedFG

	root := widget.VBox(title, countDisplay, btnRow, help)
	root.Style.Border = style.BorderRounded
	root.Style.Padding = style.Pad(1)
	root.Style.FG = style.CurrentTheme.FG
	root.Flex.Grow = 1

	a := app.New(root)
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
