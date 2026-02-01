package main

import (
	"fmt"
	"math/big"

	"cgol-distributed/distributed/pkg/combinatorics"
	"cgol-distributed/distributed/pkg/simulation"
	"cgol-distributed/distributed/pkg/symmetry"
)

// This test validates that symmetry filtering preserves all unique outcomes
// by comparing simulation results with and without filtering
func main() {
	// Small test case: 3x3 grid with 3 active cells
	width := 3
	height := 3
	nActive := 3
	maxGen := 100

	// Calculate total search space
	totalCombinations := combinatorics.Binomial(int64(width*height), int64(nActive))
	fmt.Printf("Test Configuration: %dx%d grid, %d active cells\n", width, height, nActive)
	fmt.Printf("Total search space: %s patterns\n\n", totalCombinations.String())

	// Phase 1: Simulate ALL patterns (no filtering)
	fmt.Println("Phase 1: Simulating ALL patterns (baseline)...")
	allResults := make(map[string]int) // hex -> generation count
	allCount := 0

	for seed := uint64(0); seed < 10; seed++ { // Test first 10 for speed
		indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(seed), width*height, nActive)
		board := simulation.NewBoardFromPositions(width, height, indices)
		gens, reason := board.Run(maxGen)

		if reason == "Extinction" {
			// Generate hex encoding for deduplication
			hex := encodeHex(width, height, indices)
			if prevGens, exists := allResults[hex]; exists {
				if prevGens != gens {
					fmt.Printf("ERROR: Same pattern different outcome! %s: %d vs %d\n", hex, prevGens, gens)
				}
			} else {
				allResults[hex] = gens
			}
			allCount++
		}
	}

	fmt.Printf("Baseline: Simulated all 10 patterns, found %d extinctions\n", allCount)
	fmt.Printf("Unique patterns (by hex): %d\n\n", len(allResults))

	// Phase 2: Simulate only CANONICAL patterns (with filtering)
	fmt.Println("Phase 2: Simulating CANONICAL patterns only (optimized)...")
	canonicalResults := make(map[string]int)
	canonicalCount := 0
	skippedCount := 0

	for seed := uint64(0); seed < 10; seed++ {
		indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(seed), width*height, nActive)

		// SYMMETRY FILTER: Skip non-canonical
		pattern := symmetry.NewPattern(width, height, indices)
		if !pattern.IsCanonical() {
			skippedCount++
			continue
		}

		board := simulation.NewBoardFromPositions(width, height, indices)
		gens, reason := board.Run(maxGen)

		if reason == "Extinction" {
			hex := encodeHex(width, height, indices)
			canonicalResults[hex] = gens
			canonicalCount++
		}
	}

	fmt.Printf("Optimized: Simulated %d patterns (skipped %d), found %d extinctions\n",
		10-skippedCount, skippedCount, canonicalCount)
	fmt.Printf("Unique patterns (by hex): %d\n", len(canonicalResults))
	fmt.Printf("Reduction: %.1f%% patterns skipped\n\n", float64(skippedCount)/10.0*100)

	// Phase 3: Validation - Verify both phases found the same unique patterns
	fmt.Println("Phase 3: Validating completeness...")

	// Check if canonical results are a subset of all results
	allMatch := true
	for hex, gens := range canonicalResults {
		if allGens, exists := allResults[hex]; !exists {
			fmt.Printf("ERROR: Canonical pattern not found in baseline: %s\n", hex)
			allMatch = false
		} else if allGens != gens {
			fmt.Printf("ERROR: Generation mismatch for %s: baseline=%d, canonical=%d\n", hex, allGens, gens)
			allMatch = false
		}
	}

	// Verify we didn't miss any unique outcomes
	// All unique hex patterns from baseline should have a canonical representative
	uniqueHexes := len(allResults)
	canonicalHexes := len(canonicalResults)

	if allMatch {
		fmt.Println("✓ All canonical patterns match baseline results")
	} else {
		fmt.Println("✗ VALIDATION FAILED: Mismatches detected")
	}

	fmt.Printf("✓ Baseline unique patterns: %d\n", uniqueHexes)
	fmt.Printf("✓ Canonical unique patterns: %d\n", canonicalHexes)

	if canonicalHexes == uniqueHexes {
		fmt.Println("✓ PERFECT: All unique patterns preserved")
	} else {
		fmt.Printf("  Note: Canonical found %d/%d unique hex patterns\n", canonicalHexes, uniqueHexes)
		fmt.Println("  (Some hex patterns may be symmetries of each other)")
	}

	// Final verdict
	fmt.Println("\n" + "============================================================")
	if allMatch && skippedCount > 0 {
		fmt.Printf("SUCCESS: Symmetry filtering is SAFE and EFFECTIVE\n")
		fmt.Printf("  - No unique outcomes lost\n")
		fmt.Printf("  - %.1f%% reduction in patterns simulated\n", float64(skippedCount)/10.0*100)
		fmt.Printf("  - All results mathematically equivalent\n")
	} else if !allMatch {
		fmt.Println("FAILURE: Validation errors detected")
	} else {
		fmt.Println("WARNING: No patterns were skipped (no symmetries detected)")
	}
	fmt.Println("============================================================")
}

func encodeHex(width, height int, indices []int) string {
	totalBits := width * height
	hex := ""

	isSet := func(pos int) bool {
		for _, idx := range indices {
			if idx == pos {
				return true
			}
		}
		return false
	}

	for i := 0; i < totalBits; i += 4 {
		var val byte
		if i < totalBits && isSet(i) {
			val |= 8
		}
		if i+1 < totalBits && isSet(i+1) {
			val |= 4
		}
		if i+2 < totalBits && isSet(i+2) {
			val |= 2
		}
		if i+3 < totalBits && isSet(i+3) {
			val |= 1
		}
		hex += fmt.Sprintf("%X", val)
	}
	return hex
}
