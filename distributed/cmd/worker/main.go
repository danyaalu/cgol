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
	"cgol-distributed/distributed/pkg/symmetry"
)

type Worker struct {
	serverURL   string
	width       int
	height      int
	nActive     int
	maxGen      int
	workerID    string
	useSymmetry bool
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
	w.useSymmetry = config.UseSymmetry
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
	useSymmetry := task.UseSymmetry

	if width == 0 {
		width = w.width
		height = w.height
		nActive = w.nActive
		maxGen = w.maxGen
		useSymmetry = w.useSymmetry
	}

	// Metrics: track patterns skipped due to symmetry
	var totalProcessed, skippedSymmetry uint64

	for i := 0; i < numThreads; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			var localBestSeed uint64
			var localMaxGens int
			var localFinalState string
			var localProcessed, localSkipped uint64

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

					localProcessed++

					// Apply symmetry-based pruning only when requested by coordinator
					if useSymmetry {
						pattern := symmetry.NewPattern(width, height, indices)
						if !pattern.IsCanonical() {
							localSkipped++
							continue
						}
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

			// Update global metrics
			atomic.AddUint64(&totalProcessed, localProcessed)
			atomic.AddUint64(&skippedSymmetry, localSkipped)

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

	// Report symmetry reduction statistics when enabled
	if useSymmetry && totalProcessed > 0 {
		reductionPct := float64(skippedSymmetry) / float64(totalProcessed) * 100
		fmt.Printf("Symmetry Filter: %d/%d patterns skipped (%.1f%% reduction)\n",
			skippedSymmetry, totalProcessed, reductionPct)
	} else if !useSymmetry {
		fmt.Println("Symmetry optimization disabled by coordinator; simulated full search space for this batch.")
	}

	var bestResult messages.ResultSubmission
	for res := range results {
		if res.Generations > bestResult.Generations {
			bestResult = res
		}
	}

	if bestResult.Generations > 0 {
		bestResult.Hex = encodeHex(bestResult.Seed, width, height, nActive)
		fmt.Printf("Batch Best: Seed %d, Gens %d\n", bestResult.Seed, bestResult.Generations)
		w.submitResult(bestResult)
	}
}

func encodeHex(seedIndex uint64, width, height, nActive int) string {
	// Build a presence map for row-major positions (y * width + x)
	indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(seedIndex), width*height, nActive)
	set := make(map[int]struct{}, len(indices))
	for _, idx := range indices {
		set[idx] = struct{}{}
	}

	var hexStr strings.Builder
	var nibble byte
	bitCount := 0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pos := y*width + x
			_, on := set[pos]
			bit := byte(0)
			if on {
				bit = 1
			}

			nibble = (nibble << 1) | bit
			bitCount++

			if bitCount == 4 {
				hexStr.WriteString(fmt.Sprintf("%X", nibble))
				nibble = 0
				bitCount = 0
			}
		}
	}

	// Pad remaining bits (if any) with zeros on the right
	if bitCount > 0 {
		nibble <<= (4 - bitCount)
		hexStr.WriteString(fmt.Sprintf("%X", nibble))
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
