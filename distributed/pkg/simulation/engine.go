package simulation

import (
	"hash/fnv"
	"sort"
	"fmt"
)

// Cell represents a coordinate in the infinite grid.
type Cell struct {
	X, Y int
}

// Board represents the game grid (infinite).
type Board struct {
	Cells map[Cell]bool
}

// NewBoard creates a board from a seed ID.
// width and height define the bounding box for the seed bits.
func NewBoard(width, height int, seed uint64) *Board {
	cells := make(map[Cell]bool)
	for i := 0; i < width*height; i++ {
		if (seed>>i)&1 == 1 {
			// Row-major: i = y*width + x
			y := i / width
			x := i % width
			cells[Cell{X: x, Y: y}] = true
		}
	}
	return &Board{Cells: cells}
}

// NewBoardFromPositions creates a board from a list of active indices.
// indices are in range [0, width*height-1].
func NewBoardFromPositions(width, height int, indices []int) *Board {
	cells := make(map[Cell]bool)
	for _, i := range indices {
		y := i / width
		x := i % width
		cells[Cell{X: x, Y: y}] = true
	}
	return &Board{Cells: cells}
}

// Hash returns a unique identifier for the normalized board state.
// Used for cycle detection.
func (b *Board) Hash() uint64 {
	if len(b.Cells) == 0 {
		return 0
	}

	// Collect and find bounds for normalization
	var live []Cell
	minX, minY := int(^uint(0)>>1), int(^uint(0)>>1) // Max int

	for c := range b.Cells {
		live = append(live, c)
		if c.X < minX { minX = c.X }
		if c.Y < minY { minY = c.Y }
	}

	// Normalize
	for i := range live {
		live[i].X -= minX
		live[i].Y -= minY
	}

	// Sort for deterministic hashing
	sort.Slice(live, func(i, j int) bool {
		if live[i].Y == live[j].Y {
			return live[i].X < live[j].X
		}
		return live[i].Y < live[j].Y
	})

	h := fnv.New64a()
	for _, c := range live {
		fmt.Fprintf(h, "%d,%d|", c.X, c.Y)
	}
	return h.Sum64()
}

// Step advances the board by one generation.
// Returns true if the board changed, false if it's static (stable).
func (b *Board) Step() bool {
	if len(b.Cells) == 0 {
		return false
	}

	counts := make(map[Cell]int)
	for c := range b.Cells {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 { continue }
				counts[Cell{X: c.X + dx, Y: c.Y + dy}]++
			}
		}
	}

	newCells := make(map[Cell]bool)
	changed := false

	for c, count := range counts {
		alive := b.Cells[c]
		if count == 3 || (alive && count == 2) {
			newCells[c] = true
		}
		
		if newCells[c] != alive {
			changed = true
		}
	}
	
	for c := range b.Cells {
		if _, ok := counts[c]; !ok {
			changed = true
		}
	}

	b.Cells = newCells
	return changed
}

// Run simulates the board until it halts (empty or static) or loops.
// Returns generations count and reason.
func (b *Board) Run(maxGen int) (int, string) {
	// history := make(map[uint64]int)
	
	for gen := 0; gen < maxGen; gen++ {
		// h := b.Hash()
		// if _, seen := history[h]; seen {
		// 	return gen, "Loop Detected"
		// }
		// history[h] = gen

		if len(b.Cells) == 0 {
			return gen, "Extinction"
		}

		changed := b.Step()
		if !changed {
			return gen, "Stable"
		}
	}
	return maxGen, "Max Generations Reached"
}
