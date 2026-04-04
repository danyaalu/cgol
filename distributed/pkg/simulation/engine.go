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
// Optimized using bitwise operations for 64-bit rows.
func (b *Board) Step() bool {
	// Precompute row neighbors: count horizontal neighbors + self for each row.
	// r0 = bit 0 of the count, r1 = bit 1 of the count.
	var r0, r1 [64]uint64
	for i := 0; i < 64; i++ {
		row := b.Grid[i]
		l := row << 1
		r := row >> 1
		r0[i] = row ^ l ^ r
		r1[i] = (row & l) | (row & r) | (l & r)
	}

	var next [64]uint64
	changed := false

	// Iterate through each row and sum vertical neighbors (North, Center, South).
	for y := 0; y < 64; y++ {
		var n0, n1, s0, s1 uint64

		// Load North (y-1) counts
		if y > 0 {
			n0, n1 = r0[y-1], r1[y-1]
		}
		// Load South (y+1) counts
		if y < 63 {
			s0, s1 = r0[y+1], r1[y+1]
		}
		// Load Center (y) counts
		c0, c1 := r0[y], r1[y]

		// We need to calculate Total Sum = N + C + S.
		// N, C, S are 2-bit numbers (0..3). The max sum is 9.
		// We use a series of half/full adders to sum these 2-bit numbers bitwise.

		// 1. Sum N + S = A
		// A0 = n0 ^ s0
		// Carry_A0 = n0 & s0
		// A1 = n1 ^ s1 ^ Carry_A0
		// Carry_A1 = (n1 & s1) | (Carry_A0 & (n1 ^ s1))
		// Result A is (Carry_A1, A1, A0) -> representing values 4, 2, 1

		a0 := n0 ^ s0
		c_a0 := n0 & s0
		a1 := n1 ^ s1 ^ c_a0
		c_a1 := (n1 & s1) | (c_a0 & (n1 ^ s1))

		// 2. Sum A + C = Total
		// We add C (bits c1, c0) to A (bits c_a1, a1, a0)
		// Bit 0:
		tot0 := a0 ^ c0
		c_tot0 := a0 & c0

		// Bit 1:
		tot1 := a1 ^ c1 ^ c_tot0
		c_tot1 := (a1 & c1) | (c_tot0 & (a1 ^ c1))

		// Bit 2:
		// A has bit value 4 in c_a1. C has 0.
		// Plus carry from Bit 1 (c_tot1).
		tot2 := c_a1 ^ c_tot1

		// Note: The total sum includes neighbors AND self (from horizontal sum).
		// Rules:
		// - If Sum == 3: Alive (Birth or Survival)
		// - If Sum == 4: Alive (Survival only, must be alive previously)
		// (Regular Life: 3 neighbors birth, 2 or 3 survive.
		//  Here Sum includes self.
		//  If Self=1: Neighbors=2 -> Sum=3. Neighbors=3 -> Sum=4.
		//  If Self=0: Neighbors=3 -> Sum=3.)

		// Sum == 3 (Binary 011): !tot2 & tot1 & tot0
		sum3 := ^tot2 & tot1 & tot0

		// Sum == 4 (Binary 100): tot2 & !tot1 & !tot0
		sum4 := tot2 & ^tot1 & ^tot0

		// Next state: 3 OR (4 AND Self)
		nxt := sum3 | (sum4 & b.Grid[y])

		next[y] = nxt
		if nxt != b.Grid[y] {
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
