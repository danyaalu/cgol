package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"cgol-distributed/distributed/pkg/combinatorics"
)

type TaskConfig struct {
	Width   int `json:"width"`
	Height  int `json:"height"`
	NActive int `json:"n_active"`
	MaxGen  int `json:"max_gen"`
}

func main() {
	// Load tasks
	data, err := os.ReadFile("tasks.json")
	if err != nil {
		fmt.Printf("Error reading tasks.json: %v\n", err)
		return
	}

	var tasks []TaskConfig
	if err := json.Unmarshal(data, &tasks); err != nil {
		fmt.Printf("Error parsing tasks.json: %v\n", err)
		return
	}

	fmt.Println("=== Task Analysis ===\n")

	var cumulativeSeed uint64
	for i, task := range tasks {
		totalCells := int64(task.Width * task.Height)
		maxComb := combinatorics.Binomial(totalCells, int64(task.NActive))

		var maxSeed uint64
		if maxComb.IsUint64() {
			maxSeed = maxComb.Uint64()
		} else {
			maxSeed = ^uint64(0)
		}

		fmt.Printf("Task %d: %dx%d grid, n_active=%d\n", i+1, task.Width, task.Height, task.NActive)
		fmt.Printf("  Seed range: %d - %d\n", cumulativeSeed, cumulativeSeed+maxSeed-1)
		fmt.Printf("  Total patterns: %s\n", maxComb.String())
		fmt.Printf("  Max seed: %d\n\n", maxSeed)

		// Check if seed 4279200000 falls in this range
		testSeed := uint64(4279200000)
		if testSeed >= cumulativeSeed && testSeed < cumulativeSeed+maxSeed {
			fmt.Printf("  ⚠️  Seed %d IS IN THIS TASK\n", testSeed)
			fmt.Printf("  This seems wrong! Max seed for this task is %d\n\n", maxSeed)
		}

		cumulativeSeed += maxSeed
	}

	// Check the problematic seed directly
	problemSeed := uint64(4279200000)
	fmt.Printf("\n=== Analyzing Problem Seed %d ===\n", problemSeed)

	for i, task := range tasks {
		totalCells := task.Width * task.Height
		indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(problemSeed), totalCells, task.NActive)

		fmt.Printf("\nTask %d (%dx%d, n_active=%d):\n", i+1, task.Width, task.Height, task.NActive)
		fmt.Printf("  Indices generated: %v\n", indices)

		// Check if indices are valid
		allValid := true
		for _, idx := range indices {
			if idx >= totalCells {
				fmt.Printf("  ❌ INVALID: Index %d >= total cells %d\n", idx, totalCells)
				allValid = false
			}
		}

		if allValid {
			fmt.Printf("  ✓ All indices valid\n")
		}
	}
}
