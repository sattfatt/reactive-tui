package main

import (
	"fmt"
	"log"
	"math"

	"github.com/sattfatt/reactive-tui/app"
	"github.com/sattfatt/reactive-tui/signal"
	"github.com/sattfatt/reactive-tui/style"
	"github.com/sattfatt/reactive-tui/widget"
)

func main() {
	// --- Signals ---
	count := signal.New(0)
	progress := signal.New(0.0)
	logLines := signal.New("")
	showSidebar := signal.New(false)

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
	title.Style.FG = style.CurrentTheme.NavFocusFG

	decBtn := widget.NewButton("Dec", func() {
		count.Update(func(v int) int { return v - 1 })
		p := progress.Get() - 0.1
		if p < 0 {
			p = 0
		}
		progress.Set(p)
	})
	decBtn.Flex.Basis = 10

	incBtn := widget.NewButton("Inc", func() {
		count.Update(func(v int) int { return v + 1 })
		p := progress.Get() + 0.1
		if p > 1 {
			p = 1
		}
		progress.Set(p)
	})
	incBtn.Flex.Basis = 10

	resetBtn := widget.NewButton("Reset", func() {
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

	// --- Checkbox ---
	darkMode := widget.NewCheckbox("Dark mode", func(v bool) {
		if v {
			appendLog("Dark mode enabled")
		} else {
			appendLog("Dark mode disabled")
		}
	})

	// --- Number input ---
	numInput := widget.NewNumberInput(42, func(v int) {
		appendLog(fmt.Sprintf("Number: %d", v))
	})
	numInput.WithRange(-100, 100)
	numInput.Label = "Step"

	// --- Text input ---
	nameInput := widget.NewInput("your name", func(v string) {
		appendLog(fmt.Sprintf("Name: %s", v))
	})
	nameInput.Label = "Name"

	controlsRow := widget.HBox(darkMode, numInput)
	controlsRow.Gap = 1

	counterSection := widget.VBox(title, countDisplay, pb, btnRow, controlsRow, nameInput)
	counterSection.Style.Border = style.BorderRounded
	counterSection.Style.Padding = style.Pad(1)
	counterSection.Label = "Counter"
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
	table.Label = "Table"

	// --- TextArea ---
	textArea := widget.NewTextArea(nil)
	textArea.SetText("Type here...\nMulti-line editing!")
	textArea.Flex.Grow = 1
	textArea.Label = "TextArea"

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
	finder.Label = "Fuzzy Finder"

	// --- Terminal ---
	term := widget.NewTerminal("zsh", "-l")
	term.Flex.Grow = 1
	term.Label = "Terminal"
	term.Start()
	defer term.Stop()

	// --- Neovim Editor ---
	nvimEditor := widget.NewNeovimEditor("go", func(s string) {
		lines := len([]byte(s))
		appendLog(fmt.Sprintf("Neovim saved (%d bytes)", lines))
	})
	nvimEditor.SetText("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello from neovim!\")\n}\n")
	nvimEditor.Flex.Grow = 1

	// --- Tabs combining Table, TextArea, FuzzyFinder, Terminal, Neovim ---
	tabs := widget.NewTabs(
		widget.Tab{Label: "Table", Content: table},
		widget.Tab{Label: "Editor", Content: textArea},
		widget.Tab{Label: "Finder", Content: finder},
		widget.Tab{Label: "Terminal", Content: term},
		widget.Tab{Label: "Neovim", Content: nvimEditor},
	)
	tabs.Style.Border = style.BorderRounded
	tabs.Label = "Widgets"
	tabs.Flex.Grow = 1

	// --- Log output ---
	logDisplay := widget.BoundText(func() string {
		t := logLines.Get()
		if t == "" {
			return "(no events yet)"
		}
		return t
	})
	logDisplay.Style.FG = style.CurrentTheme.MutedFG
	logDisplay.ScrollBottom = true

	logBox := widget.VBox(logDisplay)
	logBox.Style.Border = style.BorderSingle
	logBox.Style.Padding = style.Pad(0)
	logBox.Label = "Events"
	logBox.Flex.Basis = 5

	// --- Help ---
	help := widget.StaticText("hjkl: move focus | Tab: cycle | i: edit mode | Esc: back to nav (3x for terminal) | Ctrl+C: quit")
	help.Style.FG = style.CurrentTheme.MutedFG

	// --- Right sidebar (conditional) ---
	sidebarList := widget.NewList(
		func() []string {
			return []string{
				fmt.Sprintf("Count: %d", count.Get()),
				fmt.Sprintf("Progress: %.0f%%", progress.Get()*100),
				"",
				"Sidebar is a Dynamic",
				"widget that appears",
				"conditionally based",
				"on signal state.",
			}
		},
		nil,
	)
	sidebarList.Style.Border = style.BorderSingle
	sidebarList.Label = "Sidebar"
	sidebarList.Flex.Basis = 25
	sidebarList.Flex.Grow = 0
	sidebarList.Flex.Shrink = 0

	sidebar := widget.NewDynamic(func() widget.Node {
		if showSidebar.Get() {
			return sidebarList
		}
		return nil
	})

	// --- Toggle sidebar button ---
	toggleBtn := widget.NewButton("Sidebar", func() {
		showSidebar.Update(func(v bool) bool { return !v })
		sidebar.Invalidate() // rebuild the Dynamic widget tree
		if showSidebar.Get() {
			appendLog("Sidebar opened")
		} else {
			appendLog("Sidebar closed")
		}
	})
	toggleBtn.Flex.Basis = 12

	// --- Layout ---
	topRow := widget.HBox(counterSection, tabs, sidebar)
	topRow.Gap = 1
	topRow.Flex.Grow = 1

	btnBar := widget.HBox(toggleBtn, help)
	btnBar.Gap = 1

	root := widget.VBox(topRow, logBox, btnBar)
	root.Style.Padding = style.Pad(1)
	root.Gap = 1
	root.Flex.Grow = 1

	a := app.New(root)
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
