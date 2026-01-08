package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"cgol-distributed/distributed/pkg/combinatorics"
	"cgol-distributed/distributed/pkg/messages"
	"cgol-distributed/distributed/pkg/storage"
)

type Coordinator struct {
	mu          sync.Mutex
	currentSeed uint64
	maxSeed     uint64
	chunkSize   uint64
	width       int
	height      int
	nActive     int
	maxGen      int
	store       *storage.ChampionStore
}

func (c *Coordinator) Monitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastSeed uint64
	// Moving average window
	const windowSize = 30
	history := make([]uint64, 0, windowSize)

	for range ticker.C {
		c.mu.Lock()
		current := c.currentSeed
		max := c.maxSeed
		c.mu.Unlock()

		diff := current - lastSeed
		lastSeed = current

		if len(history) >= windowSize {
			history = history[1:]
		}
		history = append(history, diff)

		var sum uint64
		for _, v := range history {
			sum += v
		}
		avg := float64(sum) / float64(len(history))

		percent := 0.0
		if max > 0 {
			percent = float64(current) / float64(max) * 100
		}

		fmt.Printf("Status: Seed %d / %d (%.2f%%) | Speed: %.2f PPS\n",
			current, max, percent, avg)

		if current >= max {
			fmt.Println("Brute force completed.")
			return
		}
	}
}

func NewCoordinator(width, height, nActive, maxGen int, chunkSize uint64, store *storage.ChampionStore) *Coordinator {
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
		width:     width,
		height:    height,
		nActive:   nActive,
		maxGen:    maxGen,
		chunkSize: chunkSize,
		maxSeed:   maxSeed,
		store:     store,
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

	fmt.Printf("Received submission: Seed %d, Gens %d\n", result.Seed, result.Generations)

	// Only accept Extinction results
	if result.FinalState != "Extinction" {
		fmt.Printf("Ignoring submission from %s: Seed %d ended in %s (not Extinction)\n", result.WorkerID, result.Seed, result.FinalState)
		return
	}

	// Calculate Hex if not provided by worker
	if result.Hex == "" {
		result.Hex = c.encodeHex(result.Seed)
	}

	scope := storage.Scope{
		W:       c.width,
		H:       c.height,
		NActive: c.nActive,
		GenCap:  c.maxGen,
	}

	res := c.store.Submit(storage.Candidate{
		Scope:    scope,
		GenCount: result.Generations,
		Program:  result.Hex,
	})

	if res.Err != nil {
		log.Printf("Error saving champion: %v", res.Err)
		http.Error(w, "failed to persist champion", http.StatusInternalServerError)
		return
	}

	switch res.Outcome {
	case storage.OutcomeInsertedNewBest:
		fmt.Printf("NEW CHAMPION! Seed: %d, Gens: %d\n", result.Seed, result.Generations)
	case storage.OutcomeInsertedTie:
		fmt.Printf("MATCHING CHAMPION! Seed: %d, Gens: %d\n", result.Seed, result.Generations)
	case storage.OutcomeCachedTie:
		fmt.Printf("Cached matching champion: Seed %d, Gens %d\n", result.Seed, result.Generations)
	case storage.OutcomeDuplicate:
		fmt.Printf("Duplicate champion ignored for scope %dx%d n_active=%d gen_cap=%d\n", c.width, c.height, c.nActive, c.maxGen)
	case storage.OutcomeIgnoredLower:
		fmt.Printf("Ignoring inferior submission from %s: Seed %d Gens %d (best %d)\n", result.WorkerID, result.Seed, result.Generations, res.BestGen)
	}
}

func (c *Coordinator) Close() error {
	if c.store == nil {
		return nil
	}
	return c.store.Close()
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
	chunkSize := flag.Int64("chunk-size", 100000, "Number of programs per chunk")
	championDB := flag.String("champion-db", "champions.db", "Path to champions SQLite database")
	cacheSize := flag.Int("champion-cache-size", 1, "Number of tie submissions to cache per scope before flushing")
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	store, err := storage.NewChampionStore(*championDB, *cacheSize)
	if err != nil {
		log.Fatalf("Failed to start champion store: %v", err)
	}
	defer store.Close()

	coord := NewCoordinator(*width, *height, *nActive, *maxGen, uint64(*chunkSize), store)

	go coord.Monitor()

	mux := http.NewServeMux()
	mux.HandleFunc("/config", coord.handleConfig)
	mux.HandleFunc("/task", coord.handleTask)
	mux.HandleFunc("/submit", coord.handleSubmit)

	srv := &http.Server{
		Addr:    ":" + *port,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
		if err := coord.Close(); err != nil {
			log.Printf("Error closing coordinator: %v", err)
		}
	}()

	fmt.Printf("Coordinator started on port %s for %dx%d grid\n", *port, *width, *height)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server error: %v", err)
	}

	if err := coord.Close(); err != nil {
		log.Printf("Error closing coordinator: %v", err)
	}
}
