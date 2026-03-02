package widget

import (
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/satyam/reactive-tui/render"
	"github.com/satyam/reactive-tui/style"
)

type FuzzyMatch struct {
	Index   int
	Text    string
	Score   int
	Matches []int // indices of matched characters
}

type FuzzyFinder struct {
	Base
	AllItems []string
	filtered []FuzzyMatch
	query    string
	cursor   int // cursor position in query
	selected int // selected index in filtered list
	scrollY  int
	OnSelect func(int) // called with original index
}

func NewFuzzyFinder(items []string, onSelect func(int)) *FuzzyFinder {
	ff := &FuzzyFinder{
		Base: Base{
			Style: style.Style{FG: style.CurrentTheme.FG, BG: style.CurrentTheme.BG, Border: style.BorderSingle},
			Flex:  FlexProps{Basis: -1, Grow: 1, Shrink: 1, MinHeight: 5, MinWidth: 15},
		},
		AllItems: items,
		OnSelect: onSelect,
	}
	ff.refilter()
	return ff
}

func (ff *FuzzyFinder) Focusable() bool  { return true }
func (ff *FuzzyFinder) IsEditable() bool { return true }

func (ff *FuzzyFinder) Filtered() []FuzzyMatch { return ff.filtered }

func (ff *FuzzyFinder) refilter() {
	if ff.query == "" {
		ff.filtered = make([]FuzzyMatch, len(ff.AllItems))
		for i, item := range ff.AllItems {
			ff.filtered[i] = FuzzyMatch{Index: i, Text: item}
		}
	} else {
		ff.filtered = FuzzyMatchAll(ff.query, ff.AllItems)
	}
	ff.selected = 0
	ff.scrollY = 0
}

func (ff *FuzzyFinder) HandleKey(ev KeyEvent) bool {
	switch tcell.Key(ev.Key) {
	case tcell.KeyUp:
		if ff.selected > 0 {
			ff.selected--
		}
		return true
	case tcell.KeyDown:
		if ff.selected < len(ff.filtered)-1 {
			ff.selected++
		}
		return true
	case tcell.KeyEnter:
		if ff.OnSelect != nil && ff.selected < len(ff.filtered) {
			ff.OnSelect(ff.filtered[ff.selected].Index)
		}
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if !ff.Editing {
			return false
		}
		if ff.cursor > 0 {
			ff.query = ff.query[:ff.cursor-1] + ff.query[ff.cursor:]
			ff.cursor--
			ff.refilter()
		}
		return true
	case tcell.KeyDelete:
		if !ff.Editing {
			return false
		}
		if ff.cursor < len(ff.query) {
			ff.query = ff.query[:ff.cursor] + ff.query[ff.cursor+1:]
			ff.refilter()
		}
		return true
	case tcell.KeyLeft, tcell.KeyRight:
		if !ff.Editing {
			return false
		}
		if tcell.Key(ev.Key) == tcell.KeyLeft {
			if ff.cursor > 0 {
				ff.cursor--
			}
		} else {
			if ff.cursor < len(ff.query) {
				ff.cursor++
			}
		}
		return true
	case tcell.KeyRune:
		if !ff.Editing {
			return false
		}
		ff.query = ff.query[:ff.cursor] + string(ev.Rune) + ff.query[ff.cursor:]
		ff.cursor++
		ff.refilter()
		return true
	}
	return false
}

func (ff *FuzzyFinder) Render(r *render.Renderer, x, y, w, h int) {
	ff.Base.SetRect(x, y, w, h)
	st := ff.Style

	borderSt := st
	if ff.Focused {
		if ff.Editing {
			borderSt.FG = style.CurrentTheme.EditFocusFG
		} else {
			borderSt.FG = style.CurrentTheme.NavFocusFG
		}
	} else {
		borderSt.FG = style.CurrentTheme.BorderFG
	}
	if st.Border != style.BorderNone {
		r.DrawBorder(x, y, w, h, st.Border, borderSt)
		ff.Base.RenderLabel(r, x, y, w)
	}

	ix, iy, iw, ih := st.InnerRect(x, y, w, h)
	if iw <= 0 || ih <= 0 {
		return
	}

	// Draw query input area (1 line: "> query")
	prompt := "> " + ff.query
	if len(prompt) > iw {
		prompt = prompt[:iw]
	}
	r.DrawText(ix, iy, prompt, st, iw)

	// Draw cursor
	if ff.Focused {
		cursorX := ix + 2 + ff.cursor // 2 for "> "
		if cursorX < ix+iw {
			ch := ' '
			if ff.cursor < len(ff.query) {
				ch = rune(ff.query[ff.cursor])
			}
			cursorStyle := style.Style{FG: style.CurrentTheme.CursorFG, BG: style.CurrentTheme.CursorBG}
			r.Screen.SetContent(cursorX, iy, ch, nil, cursorStyle.TcellStyle())
		}
	}

	// Draw match count
	listStartY := 1
	listH := ih - listStartY
	if listH <= 0 {
		return
	}

	// Clamp scroll
	if ff.selected < ff.scrollY {
		ff.scrollY = ff.selected
	}
	if ff.selected >= ff.scrollY+listH {
		ff.scrollY = ff.selected - listH + 1
	}

	// Draw filtered items
	for i := 0; i < listH; i++ {
		idx := ff.scrollY + i
		if idx >= len(ff.filtered) {
			break
		}
		item := ff.filtered[idx]
		rowStyle := st
		if idx == ff.selected && ff.Focused {
			rowStyle.FG = style.CurrentTheme.SelectionFG
			rowStyle.BG = style.CurrentTheme.SelectionBG
			r.FillRect(ix, iy+listStartY+i, iw, 1, ' ', rowStyle)
		}
		text := item.Text
		if len(text) > iw {
			text = text[:iw]
		}
		r.DrawText(ix, iy+listStartY+i, text, rowStyle, iw)
	}
}

// FuzzyMatchAll performs fuzzy matching of query against all items.
// Returns matches sorted by score (highest first).
func FuzzyMatchAll(query string, items []string) []FuzzyMatch {
	queryLower := strings.ToLower(query)
	var matches []FuzzyMatch
	for i, item := range items {
		score, positions := fuzzyScore(queryLower, strings.ToLower(item), item)
		if score >= 0 {
			matches = append(matches, FuzzyMatch{
				Index:   i,
				Text:    item,
				Score:   score,
				Matches: positions,
			})
		}
	}
	// Sort by score descending (insertion sort for simplicity)
	for i := 1; i < len(matches); i++ {
		j := i
		for j > 0 && matches[j].Score > matches[j-1].Score {
			matches[j], matches[j-1] = matches[j-1], matches[j]
			j--
		}
	}
	return matches
}

// fuzzyScore returns (score, matched positions) or (-1, nil) if no match.
func fuzzyScore(queryLower, itemLower, itemOriginal string) (int, []int) {
	if len(queryLower) == 0 {
		return 0, nil
	}

	qRunes := []rune(queryLower)
	iRunes := []rune(itemLower)
	oRunes := []rune(itemOriginal)

	qi := 0
	var positions []int
	score := 0
	lastMatchPos := -1

	for ii := 0; ii < len(iRunes) && qi < len(qRunes); ii++ {
		if iRunes[ii] == qRunes[qi] {
			positions = append(positions, ii)
			// Consecutive match bonus
			if lastMatchPos >= 0 && ii == lastMatchPos+1 {
				score += 5
			}
			// Word boundary bonus
			if ii == 0 || !unicode.IsLetter(rune(iRunes[ii-1])) || (unicode.IsUpper(oRunes[ii]) && unicode.IsLower(oRunes[ii-1])) {
				score += 10
			}
			score += 1
			lastMatchPos = ii
			qi++
		}
	}

	if qi < len(qRunes) {
		return -1, nil // not all query chars matched
	}
	return score, positions
}
