package messages

// ConfigResponse is sent to workers on startup.
type ConfigResponse struct {
	Width   int `json:"width"`
	Height  int `json:"height"`
	NActive int `json:"n_active"`
	MaxGen  int `json:"max_gen"`
}

// TaskRequest is sent by a worker to ask for work.
type TaskRequest struct {
	WorkerID string `json:"worker_id"`
}

// TaskResponse assigns a range of seeds to a worker.
type TaskResponse struct {
	StartSeed uint64 `json:"start_seed"`
	EndSeed   uint64 `json:"end_seed"` // Exclusive
	TaskID    string `json:"task_id"`
}

// ResultSubmission is sent by a worker when it finds a candidate.
type ResultSubmission struct {
	WorkerID      string `json:"worker_id"`
	Seed          uint64 `json:"seed"`
	Generations   int    `json:"generations"`
	MaxPopulation int    `json:"max_population"`
	Hex           string `json:"hex,omitempty"`
	FinalState    string `json:"final_state,omitempty"`
}
