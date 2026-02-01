package main

import (
	"fmt"
	"math/big"
	"sort"
	"time"

	"cgol-distributed/distributed/pkg/combinatorics"
	"cgol-distributed/distributed/pkg/simulation"
	"cgol-distributed/distributed/pkg/symmetry"
)

type Champion struct {
	Seed        uint64
	Generations int
	FinalState  string
	Hex         string
}

func runBruteForce(width, height, nActive, maxGen int, useSymmetry bool) []Champion {
	totalCells := width * height
	maxComb := combinatorics.Binomial(int64(totalCells), int64(nActive))

	if !maxComb.IsUint64() {
		panic("Too many combinations")
	}

	maxSeed := maxComb.Uint64()
	fmt.Printf("\n=== Running %dx%d n=%d (Symmetry: %v) ===\n", width, height, nActive, useSymmetry)
	fmt.Printf("Total patterns to test: %d\n", maxSeed)

	var champions []Champion
	bestGens := 0
	processed := uint64(0)
	skipped := uint64(0)

	startTime := time.Now()
	lastPrint := startTime

	for seed := uint64(0); seed < maxSeed; seed++ {
		// Symmetry check
		if useSymmetry {
			indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(seed), totalCells, nActive)
			pattern := symmetry.NewPattern(width, height, indices)
			if !pattern.IsCanonical() {
				skipped++
				continue
			}
		}

		processed++

		// Create board and simulate
		indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(seed), totalCells, nActive)
		board := simulation.NewBoardFromPositions(width, height, indices)
		gens, finalState := board.Run(maxGen)

		// Only track extinction results
		if finalState == "Extinction" {
			if gens > bestGens {
				// New best - clear previous champions
				bestGens = gens
				champions = []Champion{{
					Seed:        seed,
					Generations: gens,
					FinalState:  finalState,
					Hex:         encodeHex(seed, width, height, nActive),
				}}
			} else if gens == bestGens {
				// Tie - add to champions
				champions = append(champions, Champion{
					Seed:        seed,
					Generations: gens,
					FinalState:  finalState,
					Hex:         encodeHex(seed, width, height, nActive),
				})
			}
		}

		// Progress update every second
		now := time.Now()
		if now.Sub(lastPrint) >= time.Second {
			elapsed := now.Sub(startTime).Seconds()
			pps := float64(processed) / elapsed
			percent := float64(seed) / float64(maxSeed) * 100
			fmt.Printf("Progress: %.2f%% | Processed: %d | Skipped: %d | Best: %d gens | Speed: %.0f PPS\n",
				percent, processed, skipped, bestGens, pps)
			lastPrint = now
		}
	}

	elapsed := time.Since(startTime)
	pps := float64(processed) / elapsed.Seconds()

	fmt.Printf("\n=== Results (Symmetry: %v) ===\n", useSymmetry)
	fmt.Printf("Total seeds: %d\n", maxSeed)
	fmt.Printf("Processed: %d\n", processed)
	fmt.Printf("Skipped: %d (%.2f%%)\n", skipped, float64(skipped)/float64(maxSeed)*100)
	fmt.Printf("Time: %v\n", elapsed)
	fmt.Printf("Speed: %.0f PPS\n", pps)
	fmt.Printf("Best generations: %d\n", bestGens)
	fmt.Printf("Champions found: %d\n", len(champions))

	return champions
}

func encodeHex(seedIndex uint64, width, height, nActive int) string {
	indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(seedIndex), width*height, nActive)
	
	totalBits := width * height
	result := make([]byte, (totalBits+3)/4)
	
	for _, idx := range indices {
		byteIdx := idx / 4
		bitPos := 3 - (idx % 4)
		result[byteIdx] |= (1 << bitPos)
	}
	
	hexStr := ""
	for _, b := range result {
		hexStr += fmt.Sprintf("%X", b)
	}
	return hexStr
}

func compareChampions(withoutSymmetry, withSymmetry []Champion) {
	fmt.Printf("\n=== COMPARISON ===\n")
	fmt.Printf("Champions without symmetry: %d\n", len(withoutSymmetry))
	fmt.Printf("Champions with symmetry: %d\n", len(withSymmetry))

	// Group by generation count
	groupByGens := func(champs []Champion) map[int][]Champion {
		m := make(map[int][]Champion)
		for _, c := range champs {
			m[c.Generations] = append(m[c.Generations], c)
		}
		return m
	}

	withoutMap := groupByGens(withoutSymmetry)
	withMap := groupByGens(withSymmetry)

	// Get unique generation counts
	gensSet := make(map[int]bool)
	for g := range withoutMap {
		gensSet[g] = true
	}
	for g := range withMap {
		gensSet[g] = true
	}

	var gensList []int
	for g := range gensSet {
		gensList = append(gensList, g)
	}
	sort.Ints(gensList)

	fmt.Printf("\nGeneration count breakdown:\n")
	for _, gens := range gensList {
		countWithout := len(withoutMap[gens])
		countWith := len(withMap[gens])
		fmt.Printf("  %d gens: %d champions (no symmetry) -> %d champions (with symmetry)\n",
			gens, countWithout, countWith)
	}

	// Verify that the best generation count is the same in both runs
	fmt.Printf("\n=== Verifying BEST OUTCOME PRESERVED ===\n")

	var bestWithout, bestWith int
	if len(withoutSymmetry) > 0 {
		bestWithout = withoutSymmetry[0].Generations
	}
	if len(withSymmetry) > 0 {
		bestWith = withSymmetry[0].Generations
	}

	if bestWithout == bestWith {
		fmt.Printf("  ✅ Best generation count: %d (same in both runs)\n", bestWithout)
	} else {
		fmt.Printf("  ❌ CRITICAL: Best generation count differs!\n")
		fmt.Printf("     Without symmetry: %d\n", bestWithout)
		fmt.Printf("     With symmetry: %d\n", bestWith)
	}

	// Verify logical equivalence by checking patterns
	fmt.Printf("\n=== Verifying UNIQUE OUTCOME GROUPS ===\n")
	
	// Build symmetry equivalence classes for WITHOUT symmetry champions
	// Group patterns that are symmetrically equivalent
	equivalenceClasses := make(map[string][]Champion)
	
	for _, champ := range withoutSymmetry {
		indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(champ.Seed), 5*5, 5)
		pattern := symmetry.NewPattern(5, 5, indices)
		canonical := pattern.CanonicalForm()
		
		// Use canonical pattern cells as key
		key := fmt.Sprintf("%v", canonical.Cells)
		equivalenceClasses[key] = append(equivalenceClasses[key], champ)
	}
	
	fmt.Printf("  Without symmetry: %d champions = %d unique symmetry classes\n", 
		len(withoutSymmetry), len(equivalenceClasses))
	fmt.Printf("  With symmetry: %d champions selected\n", len(withSymmetry))
	
	// Verify each equivalence class has exactly one representative in WITH symmetry
	classesRepresented := 0
	for _, class := range equivalenceClasses {
		// Check if any member of this class appears in withSymmetry
		found := false
		for _, classChamp := range class {
			for _, withChamp := range withSymmetry {
				if classChamp.Seed == withChamp.Seed && classChamp.Generations == withChamp.Generations {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if found {
			classesRepresented++
		}
	}
	
	if classesRepresented == len(equivalenceClasses) {
		fmt.Printf("  ✅ All %d unique outcome groups have representatives\n", len(equivalenceClasses))
	} else {
		fmt.Printf("  ❌ Only %d/%d outcome groups represented!\n", classesRepresented, len(equivalenceClasses))
	}
	
	// Show sample equivalence class
	if len(equivalenceClasses) > 0 {
		fmt.Printf("\n=== Sample Equivalence Class ===\n")
		for key, class := range equivalenceClasses {
			fmt.Printf("Canonical form: %s\n", key)
			fmt.Printf("  Contains %d symmetrically equivalent patterns:\n", len(class))
			for i, c := range class {
				if i >= 3 {
					fmt.Printf("  ... and %d more\n", len(class)-3)
					break
				}
				fmt.Printf("    Seed %d -> %d gens\n", c.Seed, c.Generations)
			}
			break // Just show one example
		}
	}

	// Calculate expected reduction ratio
	expectedReduction := float64(len(withoutSymmetry)-len(withSymmetry)) / float64(len(withoutSymmetry)) * 100
	fmt.Printf("\nChampion reduction: %.2f%% (from %d to %d)\n",
		expectedReduction, len(withoutSymmetry), len(withSymmetry))
}

func main() {
	// Use a small task that completes quickly: 5x5 grid with 5 active cells
	// Total patterns: C(25,5) = 53,130
	width := 5
	height := 5
	nActive := 5
	maxGen := 2000

	fmt.Println("Conway's Game of Life - Symmetry Optimization Validation")
	fmt.Println("=========================================================")
	fmt.Printf("Task: %dx%d grid, %d active cells, max %d generations\n", width, height, nActive, maxGen)
	fmt.Printf("Testing EXTINCTION-only busy beaver search\n")

	// Run without symmetry optimization
	championsWithout := runBruteForce(width, height, nActive, maxGen, false)

	// Run with symmetry optimization
	championsWith := runBruteForce(width, height, nActive, maxGen, true)

	// Compare results
	compareChampions(championsWithout, championsWith)

	fmt.Println("\n=== VALIDATION COMPLETE ===")
}
