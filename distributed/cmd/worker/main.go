package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sync"
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

	totalRange := task.EndSeed - task.StartSeed
	chunkSize := totalRange / uint64(numThreads)

	for i := 0; i < numThreads; i++ {
		wg.Add(1)
		start := task.StartSeed + uint64(i)*chunkSize
		end := start + chunkSize
		if i == numThreads-1 {
			end = task.EndSeed
		}

		go func(s, e uint64) {
			defer wg.Done()
			var localBestSeed uint64
			var localMaxGens int
			var localFinalState string

			for seed := s; seed < e; seed++ {
				// Convert seed (index) to combination
				indices := combinatorics.IndexToCombination(new(big.Int).SetUint64(seed), w.width*w.height, w.nActive)
				board := simulation.NewBoardFromPositions(w.width, w.height, indices)
				
				// Limit max generations to avoid infinite loops
				gens, reason := board.Run(w.maxGen)

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

			if localMaxGens > 0 {
				results <- messages.ResultSubmission{
					WorkerID:    w.workerID,
					Seed:        localBestSeed,
					Generations: localMaxGens,
					FinalState:  localFinalState,
				}
			}
		}(start, end)
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
		fmt.Printf("Batch Best: Seed %d, Gens %d\n", bestResult.Seed, bestResult.Generations)
		w.submitResult(bestResult)
	}
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
