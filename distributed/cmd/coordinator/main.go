package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"cgol-distributed/distributed/pkg/messages"
)

type Coordinator struct {
	mu              sync.Mutex
	currentSeed     uint64
	chunkSize       uint64
	width           int
	height          int
	champion        messages.ResultSubmission
	championFile    string
}

func NewCoordinator(width, height int, chunkSize uint64, championFile string) *Coordinator {
	return &Coordinator{
		width:        width,
		height:       height,
		chunkSize:    chunkSize,
		championFile: championFile,
	}
}

func (c *Coordinator) handleConfig(w http.ResponseWriter, r *http.Request) {
	resp := messages.ConfigResponse{
		Width:  c.width,
		Height: c.height,
	}
	json.NewEncoder(w).Encode(resp)
}

func (c *Coordinator) handleTask(w http.ResponseWriter, r *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple linear assignment
	start := c.currentSeed
	end := start + c.chunkSize
	
	// Check for overflow (simplified for now)
	// In a real scenario, we'd check against 2^(width*height)
	
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

	if result.Generations > c.champion.Generations {
		fmt.Printf("NEW CHAMPION! Seed: %d, Gens: %d\n", result.Seed, result.Generations)
		c.champion = result
		c.saveChampion()
	}
}

func (c *Coordinator) saveChampion() {
	file, err := os.Create(c.championFile)
	if err != nil {
		log.Printf("Error saving champion: %v", err)
		return
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(c.champion)
}

func main() {
	width := flag.Int("width", 5, "Grid width")
	height := flag.Int("height", 5, "Grid height")
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	coord := NewCoordinator(*width, *height, 100000, "champion.json")

	http.HandleFunc("/config", coord.handleConfig)
	http.HandleFunc("/task", coord.handleTask)
	http.HandleFunc("/submit", coord.handleSubmit)

	fmt.Printf("Coordinator started on port %s for %dx%d grid\n", *port, *width, *height)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
