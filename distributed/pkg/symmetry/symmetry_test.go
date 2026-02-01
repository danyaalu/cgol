package symmetry

import (
	"testing"
)

// TestRotations verifies that rotations work correctly
func TestRotations(t *testing.T) {
	// Simple L-shape pattern on 3x3 grid:
	// X X .
	// . X .
	// . . .
	// Positions: 0, 1, 4 (row-major: 0=top-left, 1=top-middle, 4=middle-center)
	p := NewPattern(3, 3, []int{0, 1, 4})

	// 90° CW rotation should give:
	// . . X
	// . X X
	// . . .
	// On the rotated grid (height=3, width=3), positions should be: 2, 4, 5
	r90 := p.Rotate90CW()
	expected90 := []int{2, 4, 5}
	if !sliceEqual(r90.Cells, expected90) {
		t.Errorf("Rotate90CW failed: got %v, want %v", r90.Cells, expected90)
	}

	// 180° rotation
	// . . .
	// . X .
	// . X X
	r180 := p.Rotate180()
	expected180 := []int{4, 7, 8}
	if !sliceEqual(r180.Cells, expected180) {
		t.Errorf("Rotate180 failed: got %v, want %v", r180.Cells, expected180)
	}

	// 270° CW rotation
	// . . .
	// X X .
	// X . .
	r270 := p.Rotate270CW()
	expected270 := []int{3, 4, 6}
	if !sliceEqual(r270.Cells, expected270) {
		t.Errorf("Rotate270CW failed: got %v, want %v", r270.Cells, expected270)
	}

	// Four 90° rotations should return to original
	r360 := p.Rotate90CW().Rotate90CW().Rotate90CW().Rotate90CW()
	if !p.Equal(r360) {
		t.Errorf("Four 90° rotations should equal identity: got %v, want %v", r360.Cells, p.Cells)
	}
}

// TestFlips verifies that flips work correctly
func TestFlips(t *testing.T) {
	// Pattern on 3x3:
	// X . .
	// . X .
	// . . X
	// Diagonal line, positions: 0, 4, 8
	p := NewPattern(3, 3, []int{0, 4, 8})

	// Horizontal flip should give:
	// . . X
	// . X .
	// X . .
	fh := p.FlipHorizontal()
	expectedH := []int{2, 4, 6}
	if !sliceEqual(fh.Cells, expectedH) {
		t.Errorf("FlipHorizontal failed: got %v, want %v", fh.Cells, expectedH)
	}

	// Vertical flip should give:
	// . . X
	// . X .
	// X . .
	fv := p.FlipVertical()
	expectedV := []int{2, 4, 6}
	if !sliceEqual(fv.Cells, expectedV) {
		t.Errorf("FlipVertical failed: got %v, want %v", fv.Cells, expectedV)
	}

	// Double horizontal flip should return to original
	fhh := p.FlipHorizontal().FlipHorizontal()
	if !p.Equal(fhh) {
		t.Errorf("Double horizontal flip should equal identity")
	}
}

// TestAllSymmetries verifies that 8 symmetries are generated
func TestAllSymmetries(t *testing.T) {
	// Asymmetric pattern on 3x3:
	// X X .
	// . X .
	// . . .
	p := NewPattern(3, 3, []int{0, 1, 4})

	symmetries := p.AllSymmetries()

	if len(symmetries) != 8 {
		t.Errorf("Expected 8 symmetries, got %d", len(symmetries))
	}

	// Verify all symmetries are distinct (for asymmetric pattern)
	seen := make(map[string]bool)
	for _, sym := range symmetries {
		key := patternKey(sym)
		if seen[key] {
			t.Errorf("Duplicate symmetry found: %v", sym.Cells)
		}
		seen[key] = true
	}
}

// TestSymmetricPattern verifies that symmetric patterns have duplicate symmetries
func TestSymmetricPattern(t *testing.T) {
	// Fully symmetric pattern (single cell in center of 3x3)
	// . . .
	// . X .
	// . . .
	p := NewPattern(3, 3, []int{4})

	symmetries := p.AllSymmetries()

	// All 8 symmetries should be identical for this pattern
	for i, sym := range symmetries {
		if !sym.Equal(p) {
			t.Errorf("Symmetry %d should equal original for centered single cell", i)
		}
	}
}

// TestCanonicalForm verifies that canonical form is deterministic and minimal
func TestCanonicalForm(t *testing.T) {
	// Pattern: X X . / . X . / . . .
	p := NewPattern(3, 3, []int{0, 1, 4})

	canonical := p.CanonicalForm()

	// All symmetries should have the same canonical form
	symmetries := p.AllSymmetries()
	for i, sym := range symmetries {
		symCanonical := sym.CanonicalForm()
		if !canonical.Equal(symCanonical) {
			t.Errorf("Symmetry %d has different canonical form: got %v, want %v",
				i, symCanonical.Cells, canonical.Cells)
		}
	}

	// Canonical form should be one of the 8 symmetries
	found := false
	for _, sym := range symmetries {
		if canonical.Equal(sym) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Canonical form %v is not among the symmetries", canonical.Cells)
	}
}

// TestIsCanonical verifies the canonical check
func TestIsCanonical(t *testing.T) {
	// Pattern: X X . / . X . / . . .
	p := NewPattern(3, 3, []int{0, 1, 4})

	symmetries := p.AllSymmetries()

	// Exactly one symmetry should be canonical (or multiple if pattern has symmetry)
	canonicalCount := 0
	for _, sym := range symmetries {
		if sym.IsCanonical() {
			canonicalCount++
		}
	}

	if canonicalCount == 0 {
		t.Errorf("No symmetry is canonical")
	}

	// All canonical ones should be equal to the canonical form
	canonical := p.CanonicalForm()
	for _, sym := range symmetries {
		if sym.IsCanonical() && !sym.Equal(canonical) {
			t.Errorf("IsCanonical returned true but pattern != canonical form")
		}
		if !sym.IsCanonical() && sym.Equal(canonical) {
			t.Errorf("IsCanonical returned false but pattern == canonical form")
		}
	}
}

// TestIsSymmetricTo verifies symmetry equivalence check
func TestIsSymmetricTo(t *testing.T) {
	p1 := NewPattern(3, 3, []int{0, 1, 4})
	p2 := p1.Rotate90CW()
	p3 := NewPattern(3, 3, []int{2, 5, 8}) // Different pattern

	if !p1.IsSymmetricTo(p2) {
		t.Errorf("p1 and p2 (rotated) should be symmetric")
	}

	if p1.IsSymmetricTo(p3) {
		t.Errorf("p1 and p3 should not be symmetric")
	}
}

// TestSquareGrid verifies correctness on square grids of different sizes
func TestSquareGrid(t *testing.T) {
	testCases := []struct {
		size  int
		cells []int
	}{
		{2, []int{0, 1}},             // 2x2
		{3, []int{0, 1, 4}},          // 3x3
		{4, []int{0, 5, 10, 15}},     // 4x4 diagonal
		{5, []int{0, 6, 12, 18, 24}}, // 5x5 diagonal
	}

	for _, tc := range testCases {
		p := NewPattern(tc.size, tc.size, tc.cells)

		// Verify all symmetries lead to the same canonical form
		symmetries := p.AllSymmetries()
		canonical := p.CanonicalForm()

		for i, sym := range symmetries {
			symCanonical := sym.CanonicalForm()
			if !canonical.Equal(symCanonical) {
				t.Errorf("Size %d: symmetry %d has different canonical: got %v, want %v",
					tc.size, i, symCanonical.Cells, canonical.Cells)
			}
		}
	}
}

// TestRectangularGrid verifies correctness on non-square grids
func TestRectangularGrid(t *testing.T) {
	// 2x3 grid pattern
	p := NewPattern(2, 3, []int{0, 2, 4})

	symmetries := p.AllSymmetries()
	canonical := p.CanonicalForm()

	// Verify all symmetries have same canonical
	for i, sym := range symmetries {
		symCanonical := sym.CanonicalForm()
		if !canonical.Equal(symCanonical) {
			t.Errorf("Rectangular: symmetry %d has different canonical: got %v (dims %dx%d), want %v (dims %dx%d)",
				i, symCanonical.Cells, symCanonical.Width, symCanonical.Height,
				canonical.Cells, canonical.Width, canonical.Height)
		}
	}

	// After rotation, width and height should swap
	r90 := p.Rotate90CW()
	if r90.Width != p.Height || r90.Height != p.Width {
		t.Errorf("After 90° rotation, dimensions should swap: got %dx%d, want %dx%d",
			r90.Width, r90.Height, p.Height, p.Width)
	}
}

// TestEmptyPattern verifies edge case of empty pattern
func TestEmptyPattern(t *testing.T) {
	p := NewPattern(3, 3, []int{})

	if !p.IsCanonical() {
		t.Errorf("Empty pattern should be canonical")
	}

	canonical := p.CanonicalForm()
	if len(canonical.Cells) != 0 {
		t.Errorf("Empty pattern canonical form should be empty")
	}
}

// TestSingleCell verifies single cell patterns
func TestSingleCell(t *testing.T) {
	// Single cell at different positions should have same canonical
	p1 := NewPattern(3, 3, []int{0}) // Top-left
	p2 := NewPattern(3, 3, []int{2}) // Top-right
	p3 := NewPattern(3, 3, []int{6}) // Bottom-left
	p4 := NewPattern(3, 3, []int{8}) // Bottom-right

	c1 := p1.CanonicalForm()
	c2 := p2.CanonicalForm()
	c3 := p3.CanonicalForm()
	c4 := p4.CanonicalForm()

	// All corners should map to the same canonical form
	if !c1.Equal(c2) || !c1.Equal(c3) || !c1.Equal(c4) {
		t.Errorf("All corner single cells should have same canonical form")
	}
}

// TestDeterminism verifies that canonical form is deterministic
func TestDeterminism(t *testing.T) {
	p := NewPattern(4, 4, []int{0, 1, 5, 10})

	// Call canonical form multiple times
	c1 := p.CanonicalForm()
	c2 := p.CanonicalForm()
	c3 := p.CanonicalForm()

	if !c1.Equal(c2) || !c1.Equal(c3) {
		t.Errorf("Canonical form should be deterministic")
	}
}

// TestReductionRatio measures the symmetry reduction for typical patterns
func TestReductionRatio(t *testing.T) {
	patterns := []*Pattern{
		NewPattern(3, 3, []int{0, 1, 4}),            // Asymmetric
		NewPattern(4, 4, []int{5, 6, 9, 10}),        // Block (symmetric)
		NewPattern(5, 5, []int{2, 7, 12, 17, 22}),   // Vertical line
		NewPattern(5, 5, []int{10, 11, 12, 13, 14}), // Horizontal line
	}

	for i, p := range patterns {
		symmetries := p.AllSymmetries()
		uniqueCanonicals := make(map[string]bool)

		for _, sym := range symmetries {
			canonical := sym.CanonicalForm()
			key := patternKey(canonical)
			uniqueCanonicals[key] = true
		}

		// All 8 symmetries should map to 1 canonical form
		if len(uniqueCanonicals) != 1 {
			t.Errorf("Pattern %d: expected 1 unique canonical, got %d", i, len(uniqueCanonicals))
		}

		// Count how many of the 8 symmetries are themselves canonical
		canonicalCount := 0
		for _, sym := range symmetries {
			if sym.IsCanonical() {
				canonicalCount++
			}
		}

		t.Logf("Pattern %d: %d/%d symmetries are canonical (%.1fx reduction)",
			i, canonicalCount, len(symmetries), float64(len(symmetries))/float64(canonicalCount))
	}
}

// Helper functions

func sliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func patternKey(p *Pattern) string {
	// Create a unique string key for the pattern
	key := ""
	for _, cell := range p.Cells {
		key += string(rune(cell))
	}
	return key
}
