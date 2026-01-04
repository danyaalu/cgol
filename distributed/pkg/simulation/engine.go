package simulation

// Board represents a 64x64 fixed grid using bitboards.
// Each uint64 represents a row.
// Note: This implementation is optimized for speed and assumes the pattern fits in 64x64.
// Patterns hitting the edge will be clipped (die).
type Board struct {
	Grid [64]uint64
}

// NewBoard creates a board from a seed ID.
func NewBoard(width, height int, seed uint64) *Board {
	b := &Board{}
	// Center the pattern in the 64x64 grid
	offsetX := (64 - width) / 2
	offsetY := (64 - height) / 2

	for i := 0; i < width*height; i++ {
		if (seed>>i)&1 == 1 {
			y := i / width
			x := i % width
			if offsetY+y < 64 && offsetX+x < 64 {
				b.Grid[offsetY+y] |= (1 << (offsetX + x))
			}
		}
	}
	return b
}

// NewBoardFromPositions creates a board from a list of active indices.
func NewBoardFromPositions(width, height int, indices []int) *Board {
	b := &Board{}
	offsetX := (64 - width) / 2
	offsetY := (64 - height) / 2

	for _, i := range indices {
		y := i / width
		x := i % width
		if offsetY+y < 64 && offsetX+x < 64 {
			b.Grid[offsetY+y] |= (1 << (offsetX + x))
		}
	}
	return b
}

// Step advances the board by one generation.
func (b *Board) Step() bool {
	var next [64]uint64
	changed := false

	for y := 0; y < 64; y++ {
		row := b.Grid[y]
		var nRow, sRow uint64
		if y > 0 {
			nRow = b.Grid[y-1]
		}
		if y < 63 {
			sRow = b.Grid[y+1]
		}

		for x := 0; x < 64; x++ {
			count := 0

			// North
			if y > 0 {
				if x > 0 && (nRow>>(x-1))&1 == 1 {
					count++
				}
				if (nRow>>x)&1 == 1 {
					count++
				}
				if x < 63 && (nRow>>(x+1))&1 == 1 {
					count++
				}
			}

			// South
			if y < 63 {
				if x > 0 && (sRow>>(x-1))&1 == 1 {
					count++
				}
				if (sRow>>x)&1 == 1 {
					count++
				}
				if x < 63 && (sRow>>(x+1))&1 == 1 {
					count++
				}
			}

			// East/West
			if x > 0 && (row>>(x-1))&1 == 1 {
				count++
			}
			if x < 63 && (row>>(x+1))&1 == 1 {
				count++
			}

			alive := (row>>x)&1 == 1
			if count == 3 || (alive && count == 2) {
				next[y] |= (1 << x)
			}
		}

		if next[y] != row {
			changed = true
		}
	}

	b.Grid = next
	return changed
}

// Run simulates the board until it halts (empty or static) or loops.
// Returns generations count and reason.
func (b *Board) Run(maxGen int) (int, string) {
	for gen := 0; gen < maxGen; gen++ {
		// Check extinction (all rows 0)
		empty := true
		for _, row := range b.Grid {
			if row != 0 {
				empty = false
				break
			}
		}
		if empty {
			return gen, "Extinction"
		}

		changed := b.Step()
		if !changed {
			return gen, "Stable"
		}
	}
	return maxGen, "Max Generations Reached"
}
