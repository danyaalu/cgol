package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"cgol-distributed/distributed/pkg/messages"
	"cgol-distributed/distributed/pkg/simulation"
)

type Worker struct {
	serverURL string
	width     int
	height    int
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

func (w *Worker) processTask(task *messages.TaskResponse) {
	var bestSeed uint64
	var maxGens int

	// Simple sequential processing for now, can be parallelized
	for seed := task.StartSeed; seed < task.EndSeed; seed++ {
		board := simulation.NewBoard(w.width, w.height, seed)
		// Limit max generations to avoid infinite loops if detection fails or takes too long
		gens, reason := board.Run(2000) 

		if reason == "Max Generations Reached" || reason == "Loop Detected" {
			// We are looking for HALTING programs (Extinction or Stable)
			// If it loops or runs forever, we ignore it for "longest running halting"
			continue
		}

		if gens > maxGens {
			maxGens = gens
			bestSeed = seed
		}
	}

	if maxGens > 0 {
		fmt.Printf("Batch Best: Seed %d, Gens %d\n", bestSeed, maxGens)
		w.submitResult(messages.ResultSubmission{
			WorkerID:    w.workerID,
			Seed:        bestSeed,
			Generations: maxGens,
		})
	}
}

func main() {
	server := flag.String("server", "http://localhost:8080", "Coordinator URL")
	flag.Parse()

	w := &Worker{
		serverURL: *server,
		workerID:  fmt.Sprintf("worker-%d", time.Now().UnixNano()),
	}

	if err := w.fetchConfig(); err != nil {
		log.Fatalf("Failed to fetch config: %v", err)
	}
	fmt.Printf("Worker started. Grid: %dx%d\n", w.width, w.height)

	// Main loop
	for {
		task, err := w.fetchTask()
		if err != nil {
			log.Printf("Error fetching task: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		fmt.Printf("Processing task: %d - %d\n", task.StartSeed, task.EndSeed)
		
		// Parallelize within the worker if needed, or just run multiple worker processes
		// For now, let's just run the task
		w.processTask(task)
	}
}
