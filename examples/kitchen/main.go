package main

import (
	"fmt"
	"log"
	"math"

	"github.com/gdamore/tcell/v2"
	"github.com/satyam/reactive-tui/app"
	"github.com/satyam/reactive-tui/signal"
	"github.com/satyam/reactive-tui/style"
	"github.com/satyam/reactive-tui/widget"
)

func main() {
	// --- Signals ---
	count := signal.New(0)
	progress := signal.New(0.0)
	logLines := signal.New("")

	appendLog := func(msg string) {
		cur := logLines.Get()
		if cur != "" {
			cur += "\n"
		}
		logLines.Set(cur + msg)
	}

	// --- Counter section ---
	title := widget.StaticText("Kitchen Sink Demo")
	title.Style.Bold = true
	title.Style.FG = tcell.ColorAqua

	decBtn := widget.NewButton("[ - ]", func() {
		count.Update(func(v int) int { return v - 1 })
		p := progress.Get() - 0.1
		if p < 0 {
			p = 0
		}
		progress.Set(p)
	})
	decBtn.Flex.Basis = 10

	incBtn := widget.NewButton("[ + ]", func() {
		count.Update(func(v int) int { return v + 1 })
		p := progress.Get() + 0.1
		if p > 1 {
			p = 1
		}
		progress.Set(p)
	})
	incBtn.Flex.Basis = 10

	resetBtn := widget.NewButton("[Reset]", func() {
		count.Set(0)
		progress.Set(0)
	})
	resetBtn.Flex.Basis = 10

	btnRow := widget.HBox(decBtn, incBtn, resetBtn)
	btnRow.Gap = 1

	countDisplay := widget.BoundText(func() string {
		return fmt.Sprintf("Count: %d", count.Get())
	})

	pb := widget.NewProgressBar(func() float64 {
		return math.Abs(progress.Get())
	})

	counterSection := widget.VBox(title, countDisplay, pb, btnRow)
	counterSection.Style.Border = style.BorderRounded
	counterSection.Style.Padding = style.Pad(1)
	counterSection.Gap = 1

	// --- Table section ---
	tableData := [][]string{
		{"Alice", "28", "Engineer"},
		{"Bob", "35", "Designer"},
		{"Charlie", "42", "Manager"},
		{"Diana", "31", "Scientist"},
		{"Eve", "26", "Analyst"},
	}
	table := widget.NewTable(
		[]widget.TableColumn{
			{Header: "Name", Grow: 2},
			{Header: "Age", Width: 5},
			{Header: "Role", Grow: 1},
		},
		func() [][]string { return tableData },
		func(i int) {
			appendLog(fmt.Sprintf("Selected: %s", tableData[i][0]))
		},
	)
	table.Flex.Grow = 1

	// --- TextArea ---
	textArea := widget.NewTextArea(nil)
	textArea.SetText("Type here...\nMulti-line editing!")
	textArea.Flex.Grow = 1

	// --- FuzzyFinder ---
	languages := []string{
		"Go", "Rust", "Python", "JavaScript", "TypeScript",
		"C", "C++", "Java", "Kotlin", "Swift",
		"Ruby", "Elixir", "Haskell", "OCaml", "Zig",
	}
	finder := widget.NewFuzzyFinder(languages, func(i int) {
		appendLog(fmt.Sprintf("Picked: %s", languages[i]))
	})
	finder.Flex.Grow = 1

	// --- Tabs combining Table, TextArea, FuzzyFinder ---
	tabs := widget.NewTabs(
		widget.Tab{Label: "Table", Content: table},
		widget.Tab{Label: "Editor", Content: textArea},
		widget.Tab{Label: "Finder", Content: finder},
	)
	tabs.Style.Border = style.BorderRounded
	tabs.Flex.Grow = 1

	// --- Log output ---
	logDisplay := widget.BoundText(func() string {
		t := logLines.Get()
		if t == "" {
			return "(no events yet)"
		}
		return t
	})
	logDisplay.Style.FG = tcell.ColorGray
	logDisplay.ScrollBottom = true

	logBox := widget.VBox(logDisplay)
	logBox.Style.Border = style.BorderSingle
	logBox.Style.Padding = style.Pad(0)
	logBox.Flex.Basis = 5

	// --- Help ---
	help := widget.StaticText("Tab: cycle focus | Ctrl+HJKL: directional focus | Arrows: navigate | Enter: select | Ctrl+C: quit")
	help.Style.FG = tcell.ColorGray

	// --- Layout ---
	topRow := widget.HBox(counterSection, tabs)
	topRow.Gap = 1
	topRow.Flex.Grow = 1

	root := widget.VBox(topRow, logBox, help)
	root.Style.Padding = style.Pad(1)
	root.Gap = 1
	root.Flex.Grow = 1

	a := app.New(root)
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
