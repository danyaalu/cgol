package symmetry

import (
	"sort"
)

// Pattern represents a set of active cell positions on a grid
type Pattern struct {
	Width  int
	Height int
	Cells  []int // Sorted list of active cell indices (row-major order)
}

// NewPattern creates a pattern from a list of active indices
func NewPattern(width, height int, indices []int) *Pattern {
	// Copy and sort to ensure canonical internal representation
	cells := make([]int, len(indices))
	copy(cells, indices)
	sort.Ints(cells)
	return &Pattern{
		Width:  width,
		Height: height,
		Cells:  cells,
	}
}

// posToXY converts a row-major index to (x, y) coordinates
func posToXY(pos, width int) (int, int) {
	return pos % width, pos / width
}

// xyToPos converts (x, y) coordinates to a row-major index
func xyToPos(x, y, width int) int {
	return y*width + x
}

// Rotate90CW rotates the pattern 90 degrees clockwise
// In a coordinate system: (x,y) -> (height-1-y, x)
func (p *Pattern) Rotate90CW() *Pattern {
	// After 90° CW rotation, width and height swap
	newWidth := p.Height
	newHeight := p.Width
	newCells := make([]int, len(p.Cells))

	for i, pos := range p.Cells {
		x, y := posToXY(pos, p.Width)
		// New coordinates after 90° CW rotation
		newX := p.Height - 1 - y
		newY := x
		newCells[i] = xyToPos(newX, newY, newWidth)
	}

	sort.Ints(newCells)
	return &Pattern{
		Width:  newWidth,
		Height: newHeight,
		Cells:  newCells,
	}
}

// Rotate180 rotates the pattern 180 degrees
// (x,y) -> (width-1-x, height-1-y)
func (p *Pattern) Rotate180() *Pattern {
	newCells := make([]int, len(p.Cells))

	for i, pos := range p.Cells {
		x, y := posToXY(pos, p.Width)
		newX := p.Width - 1 - x
		newY := p.Height - 1 - y
		newCells[i] = xyToPos(newX, newY, p.Width)
	}

	sort.Ints(newCells)
	return &Pattern{
		Width:  p.Width,
		Height: p.Height,
		Cells:  newCells,
	}
}

// Rotate270CW rotates the pattern 270 degrees clockwise (or 90 degrees CCW)
// (x,y) -> (y, width-1-x)
func (p *Pattern) Rotate270CW() *Pattern {
	newWidth := p.Height
	newHeight := p.Width
	newCells := make([]int, len(p.Cells))

	for i, pos := range p.Cells {
		x, y := posToXY(pos, p.Width)
		newX := y
		newY := p.Width - 1 - x
		newCells[i] = xyToPos(newX, newY, newWidth)
	}

	sort.Ints(newCells)
	return &Pattern{
		Width:  newWidth,
		Height: newHeight,
		Cells:  newCells,
	}
}

// FlipHorizontal flips the pattern horizontally (mirror across vertical axis)
// (x,y) -> (width-1-x, y)
func (p *Pattern) FlipHorizontal() *Pattern {
	newCells := make([]int, len(p.Cells))

	for i, pos := range p.Cells {
		x, y := posToXY(pos, p.Width)
		newX := p.Width - 1 - x
		newCells[i] = xyToPos(newX, y, p.Width)
	}

	sort.Ints(newCells)
	return &Pattern{
		Width:  p.Width,
		Height: p.Height,
		Cells:  newCells,
	}
}

// FlipVertical flips the pattern vertically (mirror across horizontal axis)
// (x,y) -> (x, height-1-y)
func (p *Pattern) FlipVertical() *Pattern {
	newCells := make([]int, len(p.Cells))

	for i, pos := range p.Cells {
		x, y := posToXY(pos, p.Width)
		newY := p.Height - 1 - y
		newCells[i] = xyToPos(x, newY, p.Width)
	}

	sort.Ints(newCells)
	return &Pattern{
		Width:  p.Width,
		Height: p.Height,
		Cells:  newCells,
	}
}

// AllSymmetries returns all 8 symmetries of the pattern:
// - Identity (0°)
// - Rotate 90° CW
// - Rotate 180°
// - Rotate 270° CW
// - Flip horizontal
// - Flip horizontal + Rotate 90° CW
// - Flip horizontal + Rotate 180°
// - Flip horizontal + Rotate 270° CW
func (p *Pattern) AllSymmetries() []*Pattern {
	symmetries := make([]*Pattern, 8)

	// First 4: rotations without flip
	symmetries[0] = p // Identity
	symmetries[1] = p.Rotate90CW()
	symmetries[2] = p.Rotate180()
	symmetries[3] = p.Rotate270CW()

	// Last 4: flip then rotations
	flipped := p.FlipHorizontal()
	symmetries[4] = flipped
	symmetries[5] = flipped.Rotate90CW()
	symmetries[6] = flipped.Rotate180()
	symmetries[7] = flipped.Rotate270CW()

	return symmetries
}

// compareLexicographic compares two patterns lexicographically.
// Returns:
//  -1 if p1 < p2
//   0 if p1 == p2
//  +1 if p1 > p2
func compareLexicographic(p1, p2 *Pattern) int {
	// First compare dimensions (smaller dimensions come first)
	if p1.Width != p2.Width {
		if p1.Width < p2.Width {
			return -1
		}
		return 1
	}
	if p1.Height != p2.Height {
		if p1.Height < p2.Height {
			return -1
		}
		return 1
	}

	// Then compare cell lists element by element
	minLen := len(p1.Cells)
	if len(p2.Cells) < minLen {
		minLen = len(p2.Cells)
	}

	for i := 0; i < minLen; i++ {
		if p1.Cells[i] < p2.Cells[i] {
			return -1
		}
		if p1.Cells[i] > p2.Cells[i] {
			return 1
		}
	}

	// If all compared elements are equal, shorter list comes first
	if len(p1.Cells) < len(p2.Cells) {
		return -1
	}
	if len(p1.Cells) > len(p2.Cells) {
		return 1
	}

	return 0
}

// CanonicalForm returns the lexicographically smallest symmetry of the pattern.
// This is used as the canonical representative for symmetry equivalence classes.
func (p *Pattern) CanonicalForm() *Pattern {
	symmetries := p.AllSymmetries()
	canonical := symmetries[0]

	for i := 1; i < len(symmetries); i++ {
		if compareLexicographic(symmetries[i], canonical) < 0 {
			canonical = symmetries[i]
		}
	}

	return canonical
}

// IsCanonical checks if the pattern is in its canonical form.
// Returns true if this pattern is the lexicographically smallest among all its symmetries.
// This is the key function for filtering: only simulate patterns where IsCanonical() == true.
func (p *Pattern) IsCanonical() bool {
	canonical := p.CanonicalForm()
	return compareLexicographic(p, canonical) == 0
}

// Equal checks if two patterns are identical (same dimensions and cells)
func (p *Pattern) Equal(other *Pattern) bool {
	return compareLexicographic(p, other) == 0
}

// IsSymmetricTo checks if two patterns are symmetrically equivalent
func (p *Pattern) IsSymmetricTo(other *Pattern) bool {
	symmetries := p.AllSymmetries()
	for _, sym := range symmetries {
		if sym.Equal(other) {
			return true
		}
	}
	return false
}
