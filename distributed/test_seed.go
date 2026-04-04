package main

import (
	"fmt"
	"cgol-distributed/distributed/pkg/simulation"
)

func main() {
	// Seed 45475 (C58D)
	// 4x4 grid
	seed := uint64(45475)
	board := simulation.NewBoard(4, 4, seed)
	
	fmt.Printf("Testing Seed %d (Hex C58D)\n", seed)
	gens, reason := board.Run(1000)
	fmt.Printf("Result: Gens=%d, Reason=%s\n", gens, reason)
}
