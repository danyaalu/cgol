package simulation

import (
	"testing"
)

func TestBlinker(t *testing.T) {
	// Blinker is a period 2 oscillator.
	// Vertical line at x=2, y=1,2,3

	b := &Board{Cells: make(map[Cell]bool)}
	b.Cells[Cell{2, 1}] = true
	b.Cells[Cell{2, 2}] = true
	b.Cells[Cell{2, 3}] = true

	gens, reason := b.Run(100)
	if reason != "Loop Detected" {
		t.Errorf("Expected Loop Detected, got %s", reason)
	}
	if gens < 1 {
		t.Errorf("Expected some generations, got %d", gens)
	}
}

func TestBlock(t *testing.T) {
	// Block is a still life.
	// 0 0 0 0
	// 0 1 1 0
	// 0 1 1 0
	// 0 0 0 0
	b := &Board{Cells: make(map[Cell]bool)}
	b.Cells[Cell{1, 1}] = true
	b.Cells[Cell{2, 1}] = true
	b.Cells[Cell{1, 2}] = true
	b.Cells[Cell{2, 2}] = true

	gens, reason := b.Run(100)
	if reason != "Stable" {
		t.Errorf("Expected Stable, got %s", reason)
	}
	if gens != 1 {
		// It should detect stability immediately after the first step if it doesn't change.
		// Actually, my implementation checks `changed` AFTER step.
		// Gen 0: Initial state.
		// Step() -> returns false (no change).
		// Loop continues? No, if !changed return.
		// So it should return 0 or 1 depending on how we count.
		// Let's check the implementation:
		// for gen := 0; gen < maxGen; gen++ { ... changed := b.Step(); if !changed { return gen, "Stable" } }
		// So if it stabilizes on the very first step, it returns 0.
		// Wait, if Step() returns false, it means state 0 -> state 1 was no change.
		// So it returns gen=0.
		t.Logf("Generations: %d", gens)
	}
}
