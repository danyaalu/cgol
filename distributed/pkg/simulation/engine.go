package simulation

import (
	"hash/fnv"
)

// Board represents the game grid.
type Board struct {
	Width  int
	Height int
	Cells  []bool
}

// NewBoard creates a board from a seed ID (brute force index).
// This maps a number (0...2^N) to a specific pattern.
func NewBoard(width, height int, seed uint64) *Board {
	cells := make([]bool, width*height)
	for i := 0; i < width*height; i++ {
		// Check if the i-th bit is set in the seed
		if (seed>>i)&1 == 1 {
			cells[i] = true
		}
	}
	return &Board{Width: width, Height: height, Cells: cells}
}

// Hash returns a unique identifier for the current board state.
// Used for cycle detection.
func (b *Board) Hash() uint64 {
	h := fnv.New64a()
	// Simple packing for small boards, or hashing for larger ones
	// For strict correctness on larger boards, we write the bytes.
	for _, cell := range b.Cells {
		if cell {
			h.Write([]byte{1})
		} else {
			h.Write([]byte{0})
		}
	}
	return h.Sum64()
}

// Step advances the board by one generation.
// Returns true if the board changed, false if it's static (stable).
func (b *Board) Step() bool {
	newCells := make([]bool, len(b.Cells))
	changed := false

	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			idx := y*b.Width + x
			neighbors := b.countNeighbors(x, y)
			alive := b.Cells[idx]

			if alive && (neighbors == 2 || neighbors == 3) {
				newCells[idx] = true
			} else if !alive && neighbors == 3 {
				newCells[idx] = true
			} else {
				newCells[idx] = false
			}

			if newCells[idx] != alive {
				changed = true
			}
		}
	}

	b.Cells = newCells
	return changed
}

func (b *Board) countNeighbors(x, y int) int {
	count := 0
	// Check all 8 neighbors
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			
			nx, ny := x+dx, y+dy
			
			// Boundary checks (non-wrapping / hard edges)
			if nx >= 0 && nx < b.Width && ny >= 0 && ny < b.Height {
				if b.Cells[ny*b.Width+nx] {
					count++
				}
			}
		}
	}
	return count
}

// Run simulates the board until it halts (empty or static) or loops.
// Returns generations count and reason.
func (b *Board) Run(maxGen int) (int, string) {
	history := make(map[uint64]int)
	
	for gen := 0; gen < maxGen; gen++ {
		h := b.Hash()
		if _, seen := history[h]; seen {
			return gen, "Loop Detected"
		}
		history[h] = gen

		// Check for empty board
		isEmpty := true
		for _, c := range b.Cells {
			if c {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			return gen, "Extinction"
		}

		changed := b.Step()
		if !changed {
			return gen, "Stable"
		}
	}
	return maxGen, "Max Generations Reached"
}
