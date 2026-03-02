package layout

// Item describes a child for the flex solver.
type Item struct {
	Basis     int     // desired size (-1 = auto, meaning use MinSize or 0)
	Grow      float64 // flex-grow factor
	Shrink    float64 // flex-shrink factor
	MinSize   int     // minimum size
	MaxSize   int     // maximum size (0 = no limit)
}

// Solve distributes `available` space among `items` using flexbox logic.
// Returns the allocated size for each item.
func Solve(items []Item, available int, gap int) []int {
	n := len(items)
	if n == 0 {
		return nil
	}

	sizes := make([]int, n)
	totalGap := gap * (n - 1)
	space := available - totalGap
	if space < 0 {
		space = 0
	}

	// Step 1: assign basis sizes
	totalBasis := 0
	for i, item := range items {
		basis := item.Basis
		if basis < 0 {
			basis = item.MinSize
		}
		sizes[i] = basis
		totalBasis += basis
	}

	remaining := space - totalBasis

	if remaining > 0 {
		// Step 2a: distribute positive remaining space via grow
		totalGrow := 0.0
		for _, item := range items {
			totalGrow += item.Grow
		}
		if totalGrow > 0 {
			for i, item := range items {
				extra := int(float64(remaining) * item.Grow / totalGrow)
				sizes[i] += extra
			}
		}
	} else if remaining < 0 {
		// Step 2b: shrink items
		deficit := -remaining
		totalShrink := 0.0
		for i, item := range items {
			totalShrink += item.Shrink * float64(sizes[i])
			_ = i
		}
		if totalShrink > 0 {
			for i, item := range items {
				reduction := int(float64(deficit) * item.Shrink * float64(sizes[i]) / totalShrink)
				sizes[i] -= reduction
			}
		}
	}

	// Step 3: clamp to min/max
	for i, item := range items {
		if sizes[i] < item.MinSize {
			sizes[i] = item.MinSize
		}
		if item.MaxSize > 0 && sizes[i] > item.MaxSize {
			sizes[i] = item.MaxSize
		}
		if sizes[i] < 0 {
			sizes[i] = 0
		}
	}

	return sizes
}
