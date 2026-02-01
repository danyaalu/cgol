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

type TaskConfig struct {
	Width   int `json:"width"`
	Height  int `json:"height"`
	NActive int `json:"n_active"`
	MaxGen  int `json:"max_gen"`
}

type Coordinator struct {
	mu          sync.Mutex
	tasks       []TaskConfig
	taskIndex   int
	currentSeed uint64
	maxSeed     uint64
	chunkSize   uint64
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
		if c.taskIndex >= len(c.tasks) {
			c.mu.Unlock()
			fmt.Println("All brute force tasks completed.")
			return
		}

		current := c.currentSeed
		max := c.maxSeed
		currentTask := c.tasks[c.taskIndex]
		taskIdx := c.taskIndex
		totalTasks := len(c.tasks)
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

		fmt.Printf("Task %d/%d [%dx%d n=%d]: Seed %d / %d (%.2f%%) | Speed: %.2f PPS\n",
			taskIdx+1, totalTasks, currentTask.Width, currentTask.Height, currentTask.NActive,
			current, max, percent, avg)
	}
}

func NewCoordinator(tasks []TaskConfig, chunkSize uint64, store *storage.ChampionStore) *Coordinator {
	if len(tasks) == 0 {
		log.Fatal("No tasks provided to coordinator")
	}

	c := &Coordinator{
		tasks:     tasks,
		chunkSize: chunkSize,
		store:     store,
		taskIndex: 0,
	}
	c.initCurrentTask()
	return c
}

func (c *Coordinator) initCurrentTask() {
	if c.taskIndex >= len(c.tasks) {
		return
	}
	task := c.tasks[c.taskIndex]

	// Calculate max combinations (width*height Choose nActive)
	totalCells := int64(task.Width * task.Height)
	maxComb := combinatorics.Binomial(totalCells, int64(task.NActive))

	var maxSeed uint64
	if maxComb.IsUint64() {
		maxSeed = maxComb.Uint64()
		// If explicit overflow check is needed, logic goes here
	} else {
		maxSeed = ^uint64(0) // Cap at MaxUint64
		log.Printf("Warning: Total combinations exceed uint64. Capped at %d", maxSeed)
	}

	c.maxSeed = maxSeed
	c.currentSeed = 0 // CRITICAL: Reset seed counter for new task
	fmt.Printf("Initialized task %d: maxSeed=%d, currentSeed reset to 0\n", c.taskIndex, maxSeed)
}

func (c *Coordinator) handleConfig(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return the configuration of the current task, or the first one if finished
	idx := c.taskIndex
	if idx >= len(c.tasks) {
		idx = 0
	}
	task := c.tasks[idx]

	resp := messages.ConfigResponse{
		Width:   task.Width,
		Height:  task.Height,
		NActive: task.NActive,
		MaxGen:  task.MaxGen,
	}
	json.NewEncoder(w).Encode(resp)
}

func (c *Coordinator) handleTask(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to advance to the next task
	for c.currentSeed >= c.maxSeed {
		c.taskIndex++
		if c.taskIndex >= len(c.tasks) {
			json.NewEncoder(w).Encode(messages.TaskResponse{
				TaskID: "DONE",
			})
			return
		}
		c.initCurrentTask()
		fmt.Printf("Advancing to Task %d/%d: %dx%d n=%d\n", c.taskIndex+1, len(c.tasks), c.tasks[c.taskIndex].Width, c.tasks[c.taskIndex].Height, c.tasks[c.taskIndex].NActive)
	}

	task := c.tasks[c.taskIndex]

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
		TaskID:    fmt.Sprintf("%d-%d-%d", c.taskIndex, start, end),
		Width:     task.Width,
		Height:    task.Height,
		NActive:   task.NActive,
		MaxGen:    task.MaxGen,
	}
	json.NewEncoder(w).Encode(resp)
}

func (c *Coordinator) handleSubmit(w http.ResponseWriter, r *http.Request) {
	var result messages.ResultSubmission
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("Received submission: Seed %d, Gens %d from %s\n", result.Seed, result.Generations, result.WorkerID)

	// Only accept Extinction results
	if result.FinalState != "Extinction" {
		fmt.Printf("Ignoring submission from %s: Seed %d ended in %s (not Extinction)\n", result.WorkerID, result.Seed, result.FinalState)
		return
	}

	// Use parameters from result, fallback to current task if missing (should not happen with updated workers)
	// Fallback logic is risky if task switched, but best effort.
	scopeWidth := result.Width
	scopeHeight := result.Height
	scopeNActive := result.NActive
	scopeMaxGen := result.MaxGen

	if scopeWidth == 0 {
		// Log warning or try to infer?
		// For now, let's just log a warning and try current task, but this is dangerous.
		// Assuming updated workers.
		fmt.Println("Warning: Submission missing task parameters. Attributes may be incorrect.")
	}

	// Calculate Hex if not provided by worker
	if result.Hex == "" {
		result.Hex = c.encodeHex(result.Seed, scopeWidth, scopeHeight, scopeNActive)
	}

	scope := storage.Scope{
		W:       scopeWidth,
		H:       scopeHeight,
		NActive: scopeNActive,
		GenCap:  scopeMaxGen,
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
		fmt.Printf("Duplicate champion ignored for scope %dx%d n_active=%d gen_cap=%d\n", scopeWidth, scopeHeight, scopeNActive, scopeMaxGen)
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

func (c *Coordinator) encodeHex(seedIndex uint64, width, height, nActive int) string {
	// Convert index to actual positions
	indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(seedIndex), width*height, nActive)

	// Create a bitmap from positions
	// Note: This only works if width*height <= 64.
	// If larger, we can't represent as single uint64, but we can still generate hex string.

	totalBits := width * height
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
	tasksFile := flag.String("tasks", "", "Path to JSON file containing list of tasks (overrides individual flags)")
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

	var tasks []TaskConfig
	if *tasksFile != "" {
		data, err := os.ReadFile(*tasksFile)
		if err != nil {
			log.Fatalf("Failed to read tasks file: %v", err)
		}
		if err := json.Unmarshal(data, &tasks); err != nil {
			log.Fatalf("Failed to parse tasks file: %v", err)
		}
		fmt.Printf("Loaded %d tasks from %s\n", len(tasks), *tasksFile)
	} else {
		tasks = []TaskConfig{{
			Width:   *width,
			Height:  *height,
			NActive: *nActive,
			MaxGen:  *maxGen,
		}}
	}

	coord := NewCoordinator(tasks, uint64(*chunkSize), store)

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

	if *tasksFile != "" {
		fmt.Printf("Coordinator started on port %s running %d tasks\n", *port, len(tasks))
	} else {
		fmt.Printf("Coordinator started on port %s for %dx%d grid\n", *port, *width, *height)
	}

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server error: %v", err)
	}

	if err := coord.Close(); err != nil {
		log.Printf("Error closing coordinator: %v", err)
	}
}
