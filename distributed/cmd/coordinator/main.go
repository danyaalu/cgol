package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"

	"cgol-distributed/distributed/pkg/combinatorics"
	"cgol-distributed/distributed/pkg/messages"
)

type Coordinator struct {
	mu              sync.Mutex
	currentSeed     uint64
	maxSeed         uint64
	chunkSize       uint64
	width           int
	height          int
	nActive         int
	maxGen          int
	champions       []messages.ResultSubmission
	championFile    string
}

func NewCoordinator(width, height, nActive, maxGen int, chunkSize uint64, championFile string) *Coordinator {
	// Calculate max combinations (width*height Choose nActive)
	totalCells := int64(width * height)
	maxComb := combinatorics.Binomial(totalCells, int64(nActive))
	
	var maxSeed uint64
	if maxComb.IsUint64() {
		maxSeed = maxComb.Uint64()
	} else {
		maxSeed = ^uint64(0) // Cap at MaxUint64
		log.Printf("Warning: Total combinations exceed uint64. Capped at %d", maxSeed)
	}

	return &Coordinator{
		width:        width,
		height:       height,
		nActive:      nActive,
		maxGen:       maxGen,
		chunkSize:    chunkSize,
		championFile: championFile,
		maxSeed:      maxSeed,
	}
}

func (c *Coordinator) handleConfig(w http.ResponseWriter, r *http.Request) {
	resp := messages.ConfigResponse{
		Width:   c.width,
		Height:  c.height,
		NActive: c.nActive,
		MaxGen:  c.maxGen,
	}
	json.NewEncoder(w).Encode(resp)
}

func (c *Coordinator) handleTask(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.currentSeed >= c.maxSeed {
		json.NewEncoder(w).Encode(messages.TaskResponse{
			TaskID: "DONE",
		})
		return
	}

	// Simple linear assignment
	start := c.currentSeed
	end := start + c.chunkSize
	
	if end > c.maxSeed {
		end = c.maxSeed
	}
	
	c.currentSeed = end

	resp := messages.TaskResponse{
		StartSeed: start,
		EndSeed:   end,
		TaskID:    fmt.Sprintf("%d-%d", start, end),
	}
	json.NewEncoder(w).Encode(resp)
}

func (c *Coordinator) handleSubmit(w http.ResponseWriter, r *http.Request) {
	var result messages.ResultSubmission
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	fmt.Printf("Received submission: Seed %d, Gens %d\n", result.Seed, result.Generations)

	// Only accept Extinction results
	if result.FinalState != "Extinction" {
		fmt.Printf("Ignoring submission from %s: Seed %d ended in %s (not Extinction)\n", result.WorkerID, result.Seed, result.FinalState)
		return
	}

	// Calculate Hex
	result.Hex = c.encodeHex(result.Seed)

	if len(c.champions) == 0 || result.Generations > c.champions[0].Generations {
		fmt.Printf("NEW CHAMPION! Seed: %d, Gens: %d\n", result.Seed, result.Generations)
		c.champions = []messages.ResultSubmission{result}
		c.saveChampions()
	} else if result.Generations == c.champions[0].Generations {
		fmt.Printf("MATCHING CHAMPION! Seed: %d, Gens: %d\n", result.Seed, result.Generations)
		c.champions = append(c.champions, result)
		c.saveChampions()
	}
}

func (c *Coordinator) saveChampions() {
	file, err := os.Create(c.championFile)
	if err != nil {
		log.Printf("Error saving champions: %v", err)
		return
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(c.champions)
}

func (c *Coordinator) encodeHex(seedIndex uint64) string {
	// Convert index to actual positions
	indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(seedIndex), c.width*c.height, c.nActive)
	
	// Create a bitmap from positions
	// Note: This only works if width*height <= 64. 
	// If larger, we can't represent as single uint64, but we can still generate hex string.
	
	totalBits := c.width * c.height
	var hexStr strings.Builder

	// Helper to check if bit 'pos' is set
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
		// Bit 0 (MSB of nibble)
		if i < totalBits && isSet(i) {
			val |= 8
		}
		// Bit 1
		if i+1 < totalBits && isSet(i+1) {
			val |= 4
		}
		// Bit 2
		if i+2 < totalBits && isSet(i+2) {
			val |= 2
		}
		// Bit 3 (LSB of nibble)
		if i+3 < totalBits && isSet(i+3) {
			val |= 1
		}

		hexStr.WriteString(fmt.Sprintf("%X", val))
	}
	return hexStr.String()
}

func main() {
	width := flag.Int("width", 5, "Grid width")
	height := flag.Int("height", 5, "Grid height")
	nActive := flag.Int("n-active", 5, "Number of active cells")
	maxGen := flag.Int("max-gen", 2000, "Maximum generations")
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	coord := NewCoordinator(*width, *height, *nActive, *maxGen, 1000, "champion.json")

	http.HandleFunc("/config", coord.handleConfig)
	http.HandleFunc("/task", coord.handleTask)
	http.HandleFunc("/submit", coord.handleSubmit)

	fmt.Printf("Coordinator started on port %s for %dx%d grid\n", *port, *width, *height)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
