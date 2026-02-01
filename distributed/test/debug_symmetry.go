package main

import (
	"fmt"
	"math/big"

	"cgol-distributed/distributed/pkg/combinatorics"
	"cgol-distributed/distributed/pkg/symmetry"
)

func main() {
	// Test the specific seed ranges from your logs
	configs := []struct {
		width   int
		height  int
		nActive int
		seeds   []uint64
	}{
		{5, 5, 5, []uint64{0, 1000, 10000, 50000}},
		{6, 6, 6, []uint64{0, 100000, 1000000, 1900000}},
		{5, 10, 7, []uint64{0, 1000000, 50000000, 99000000}},
	}

	for _, cfg := range configs {
		fmt.Printf("\n=== Testing %dx%d, n_active=%d ===\n", cfg.width, cfg.height, cfg.nActive)

		canonicalCount := 0
		totalChecked := len(cfg.seeds)

		for _, seed := range cfg.seeds {
			indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(seed), cfg.width*cfg.height, cfg.nActive)
			pattern := symmetry.NewPattern(cfg.width, cfg.height, indices)
			isCanonical := pattern.IsCanonical()

			if isCanonical {
				canonicalCount++
			}

			fmt.Printf("Seed %d: indices=%v, canonical=%v\n", seed, indices, isCanonical)
		}

		reductionPct := float64(totalChecked-canonicalCount) / float64(totalChecked) * 100
		fmt.Printf("Result: %d/%d canonical (%.1f%% reduction)\n", canonicalCount, totalChecked, reductionPct)
	}

	// Test a full batch to see if we get 100% reduction
	fmt.Println("\n=== Testing Full Batch (seed 0-1000) ===")
	width, height, nActive := 5, 5, 5
	canonicalInBatch := 0

	for seed := uint64(0); seed < 1000; seed++ {
		indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(seed), width*height, nActive)
		pattern := symmetry.NewPattern(width, height, indices)
		if pattern.IsCanonical() {
			canonicalInBatch++
		}
	}

	batchReduction := float64(1000-canonicalInBatch) / 1000.0 * 100
	fmt.Printf("Batch 0-1000: %d canonical, %.1f%% reduction\n", canonicalInBatch, batchReduction)

	if canonicalInBatch == 0 {
		fmt.Println("WARNING: No canonical patterns in batch! This indicates a BUG.")
	} else if batchReduction > 95 {
		fmt.Println("WARNING: >95% reduction is suspicious.")
	} else {
		fmt.Println("OK: Reduction percentage looks normal.")
	}
}
