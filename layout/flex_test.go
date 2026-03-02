package layout

import (
	"testing"
)

func TestSolveEmpty(t *testing.T) {
	result := Solve(nil, 100, 0)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestSolveSingleItemFixedBasis(t *testing.T) {
	items := []Item{{Basis: 50}}
	sizes := Solve(items, 100, 0)
	if sizes[0] != 50 {
		t.Errorf("expected 50, got %d", sizes[0])
	}
}

func TestSolveSingleItemGrow(t *testing.T) {
	items := []Item{{Basis: 0, Grow: 1}}
	sizes := Solve(items, 100, 0)
	if sizes[0] != 100 {
		t.Errorf("expected 100, got %d", sizes[0])
	}
}

func TestSolveEqualGrow(t *testing.T) {
	items := []Item{
		{Basis: 0, Grow: 1},
		{Basis: 0, Grow: 1},
	}
	sizes := Solve(items, 100, 0)
	if sizes[0] != 50 || sizes[1] != 50 {
		t.Errorf("expected [50, 50], got %v", sizes)
	}
}

func TestSolveUnequalGrow(t *testing.T) {
	items := []Item{
		{Basis: 0, Grow: 1},
		{Basis: 0, Grow: 3},
	}
	sizes := Solve(items, 100, 0)
	if sizes[0] != 25 || sizes[1] != 75 {
		t.Errorf("expected [25, 75], got %v", sizes)
	}
}

func TestSolveWithGap(t *testing.T) {
	items := []Item{
		{Basis: 0, Grow: 1},
		{Basis: 0, Grow: 1},
		{Basis: 0, Grow: 1},
	}
	// 100 - 2 gaps * 5 = 90, split 3 ways = 30 each
	sizes := Solve(items, 100, 5)
	if sizes[0] != 30 || sizes[1] != 30 || sizes[2] != 30 {
		t.Errorf("expected [30, 30, 30], got %v", sizes)
	}
}

func TestSolveBasisPlusGrow(t *testing.T) {
	items := []Item{
		{Basis: 20, Grow: 1},
		{Basis: 30, Grow: 1},
	}
	// total basis=50, remaining=50, split 25 each
	sizes := Solve(items, 100, 0)
	if sizes[0] != 45 || sizes[1] != 55 {
		t.Errorf("expected [45, 55], got %v", sizes)
	}
}

func TestSolveShrink(t *testing.T) {
	items := []Item{
		{Basis: 60, Shrink: 1},
		{Basis: 60, Shrink: 1},
	}
	sizes := Solve(items, 100, 0)
	// deficit=20, each shrinks proportionally: 60*1/(60+60)*20 = 10
	if sizes[0] != 50 || sizes[1] != 50 {
		t.Errorf("expected [50, 50], got %v", sizes)
	}
}

func TestSolveMinSize(t *testing.T) {
	items := []Item{
		{Basis: 10, MinSize: 30},
		{Basis: 10},
	}
	sizes := Solve(items, 50, 0)
	if sizes[0] < 30 {
		t.Errorf("expected sizes[0] >= 30, got %d", sizes[0])
	}
}

func TestSolveMaxSize(t *testing.T) {
	items := []Item{
		{Basis: 0, Grow: 1, MaxSize: 40},
		{Basis: 0, Grow: 1},
	}
	sizes := Solve(items, 100, 0)
	if sizes[0] > 40 {
		t.Errorf("expected sizes[0] <= 40, got %d", sizes[0])
	}
}

func TestSolveAutoBasis(t *testing.T) {
	items := []Item{
		{Basis: -1, MinSize: 10, Grow: 1},
		{Basis: -1, MinSize: 20, Grow: 1},
	}
	// auto basis uses MinSize: 10+20=30, remaining=70, split 35 each
	sizes := Solve(items, 100, 0)
	if sizes[0] != 45 || sizes[1] != 55 {
		t.Errorf("expected [45, 55], got %v", sizes)
	}
}

func TestSolveNoGrowNoShrink(t *testing.T) {
	items := []Item{
		{Basis: 30},
		{Basis: 40},
	}
	sizes := Solve(items, 100, 0)
	// No grow/shrink, items keep basis
	if sizes[0] != 30 || sizes[1] != 40 {
		t.Errorf("expected [30, 40], got %v", sizes)
	}
}

func TestSolveZeroAvailable(t *testing.T) {
	items := []Item{
		{Basis: 50, Shrink: 1},
		{Basis: 50, Shrink: 1},
	}
	sizes := Solve(items, 0, 0)
	// Should shrink to 0 or near 0
	for i, s := range sizes {
		if s < 0 {
			t.Errorf("sizes[%d] should not be negative: %d", i, s)
		}
	}
}

func TestSolveNegativeGap(t *testing.T) {
	items := []Item{
		{Basis: 0, Grow: 1},
		{Basis: 0, Grow: 1},
	}
	sizes := Solve(items, 100, 0)
	total := 0
	for _, s := range sizes {
		total += s
	}
	if total != 100 {
		t.Errorf("expected total=100, got %d", total)
	}
}

func TestSolveManyItems(t *testing.T) {
	items := make([]Item, 10)
	for i := range items {
		items[i] = Item{Basis: 0, Grow: 1}
	}
	sizes := Solve(items, 100, 0)
	for i, s := range sizes {
		if s != 10 {
			t.Errorf("sizes[%d] expected 10, got %d", i, s)
		}
	}
}
