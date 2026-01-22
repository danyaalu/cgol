package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cgol-distributed/distributed/pkg/combinatorics"
	"cgol-distributed/distributed/pkg/messages"
	"cgol-distributed/distributed/pkg/simulation"
)

type Worker struct {
	serverURL string
	width     int
	height    int
	nActive   int
	maxGen    int
	workerID  string
}

func (w *Worker) fetchConfig() error {
	resp, err := http.Get(w.serverURL + "/config")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var config messages.ConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return err
	}
	w.width = config.Width
	w.height = config.Height
	w.nActive = config.NActive
	w.maxGen = config.MaxGen
	return nil
}

func (w *Worker) fetchTask() (*messages.TaskResponse, error) {
	// In a real app, we might POST a request with worker ID
	resp, err := http.Get(w.serverURL + "/task")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var task messages.TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (w *Worker) submitResult(result messages.ResultSubmission) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	resp, err := http.Post(w.serverURL+"/submit", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (w *Worker) processTask(task *messages.TaskResponse, numThreads int) {
	var wg sync.WaitGroup
	results := make(chan messages.ResultSubmission, numThreads)

	// Dynamic work stealing: threads grab batches of seeds atomically
	currentSeed := task.StartSeed
	const batchSize = 100

	// Use task parameters if available, otherwise fall back to worker defaults
	width := task.Width
	height := task.Height
	nActive := task.NActive
	maxGen := task.MaxGen

	if width == 0 {
		width = w.width
		height = w.height
		nActive = w.nActive
		maxGen = w.maxGen
	}

	for i := 0; i < numThreads; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			var localBestSeed uint64
			var localMaxGens int
			var localFinalState string

			for {
				// Atomically claim a batch
				endBatch := atomic.AddUint64(&currentSeed, batchSize)
				startBatch := endBatch - batchSize

				if startBatch >= task.EndSeed {
					break
				}

				// Clamp the end of the batch to the task limit
				if endBatch > task.EndSeed {
					endBatch = task.EndSeed
				}

				// Initialize the first combination for this batch
				indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(startBatch), width*height, nActive)

				for seed := startBatch; seed < endBatch; seed++ {
					// For subsequent seeds in the batch, calculate next combination incrementally
					if seed > startBatch {
						combinatorics.NextCombination(indices, width*height)
					}

					board := simulation.NewBoardFromPositions(width, height, indices)

					// Limit max generations to avoid infinite loops
					gens, reason := board.Run(maxGen)

					// Only consider programs that halt with extinction (all dead)
					if reason != "Extinction" {
						continue
					}

					if gens > localMaxGens {
						localMaxGens = gens
						localBestSeed = seed
						localFinalState = reason
					}
				}
			}

			if localMaxGens > 0 {
				results <- messages.ResultSubmission{
					WorkerID:    w.workerID,
					Seed:        localBestSeed,
					Generations: localMaxGens,
					FinalState:  localFinalState,
					Width:       width,
					Height:      height,
					NActive:     nActive,
					MaxGen:      maxGen,
				}
			}
		}()
	}

	wg.Wait()
	close(results)

	var bestResult messages.ResultSubmission
	for res := range results {
		if res.Generations > bestResult.Generations {
			bestResult = res
		}
	}

	if bestResult.Generations > 0 {
		bestResult.Hex = w.encodeHex(bestResult.Seed)
		fmt.Printf("Batch Best: Seed %d, Gens %d\n", bestResult.Seed, bestResult.Generations)
		w.submitResult(bestResult)
	}
}

func (w *Worker) encodeHex(seedIndex uint64) string {
	// Convert index to actual positions
	indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(seedIndex), w.width*w.height, w.nActive)

	totalBits := w.width * w.height
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
	server := flag.String("server", "http://localhost:8080", "Coordinator URL")
	threads := flag.Int("threads", 1, "Number of concurrent simulation threads")
	flag.Parse()

	w := &Worker{
		serverURL: *server,
		workerID:  fmt.Sprintf("worker-%d", time.Now().UnixNano()),
	}

	if err := w.fetchConfig(); err != nil {
		log.Fatalf("Failed to fetch config: %v", err)
	}
	fmt.Printf("Worker started with %d threads. Grid: %dx%d\n", *threads, w.width, w.height)

	// Main loop
	for {
		task, err := w.fetchTask()
		if err != nil {
			log.Printf("Error fetching task: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if task.TaskID == "DONE" {
			fmt.Println("No more work available. Exiting.")
			return
		}

		fmt.Printf("Processing task: %d - %d\n", task.StartSeed, task.EndSeed)

		w.processTask(task, *threads)
	}
}
