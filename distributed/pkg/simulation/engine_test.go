package simulation

import (
	"testing"
)

func TestBlinker(t *testing.T) {
	// Blinker is a period 2 oscillator.
	// Vertical line at x=2, y=1,2,3
	// Using bitboard implementation: Grid[y]
	b := &Board{}
	b.Grid[1] = 1 << 2
	b.Grid[2] = 1 << 2
	b.Grid[3] = 1 << 2

	// Note: The loop detection in Run() is simplified to just check for stability or extinction.
	// The new implementation does NOT detect loops specifically unless they are period 1 (Stable).
	// So a blinker will run until maxGen.
	gens, reason := b.Run(10)
	if reason != "Max Generations Reached" {
		t.Errorf("Expected Max Generations Reached for oscillator without explicit loop detection, got %s", reason)
	}
	if gens != 10 {
		t.Errorf("Expected 10 generations, got %d", gens)
	}
}

func TestBlock(t *testing.T) {
	// Block is a still life.
	// 0 0 0 0
	// 0 1 1 0
	// 0 1 1 0
	// 0 0 0 0
	b := &Board{}
	b.Grid[1] = (1 << 1) | (1 << 2)
	b.Grid[2] = (1 << 1) | (1 << 2)

	gens, reason := b.Run(100)
	if reason != "Stable" {
		t.Errorf("Expected Stable, got %s", reason)
	}
	// It stabilizes immediately (no change on first step).
	// Step() returns change=false, so loop returns 'gen' which is 0.
	if gens != 0 {
		t.Errorf("Expected 0 generations for immediate stability, got %d", gens)
	}
}
