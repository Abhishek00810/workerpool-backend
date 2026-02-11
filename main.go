package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type WorkerPool struct {
	jobs    chan Job
	results chan Result
	wg      sync.WaitGroup
	quit    chan struct{}
}

// Job represents the unit of work
type Job struct {
	ID   int
	Num1 int
	Num2 int
}

type Result struct {
	Sum   int
	JobID int
	Err   error
}

func NewWorkerPool(bufferSize int, resultSize int) *WorkerPool {
	return &WorkerPool{
		jobs:    make(chan Job, bufferSize),
		results: make(chan Result, bufferSize),
		quit:    make(chan struct{}),
	}
}

func (wp *WorkerPool) Start(ctx context.Context, numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		wp.wg.Add(1) // Increment waitgroup
		go wp.worker(ctx, i)
	}
}

func (wp *WorkerPool) worker(ctx context.Context, id int) {
	defer wp.wg.Done() // gracefully or in momment shutdown
	fmt.Printf("Worker %d started\n", id)

	for {
		select {
		case <-ctx.Done():
			// process got halt , return that time without even watching the process
			fmt.Printf("worker %d stopping (context cancellations) \n", id)
			return
		case job, ok := <-wp.jobs:
			if !ok {
				fmt.Println("closing the process")
				return
			}
			fmt.Printf("Worker %d processing Job %d\n", id, job.ID)
			time.Sleep(100 * time.Millisecond) // Simulate work
			// Send result
			wp.results <- Result{JobID: job.ID, Sum: job.Num1 + job.Num2}
		}

	}
}

func (wp *WorkerPool) AddJob(j Job) {
	wp.jobs <- j
}

func (wp *WorkerPool) Stop() {
	close(wp.jobs)
	wp.wg.Wait()
	close(wp.results)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool := NewWorkerPool(10, 10)

	// Start 3 workers
	pool.Start(ctx, 3)

	// Dispatch 5 jobs
	go func() {
		for i := 1; i <= 5; i++ {
			pool.AddJob(Job{ID: i, Num1: i, Num2: i * 2})
		}
		// Standard: Explicitly stop the pool when we are done sending
		pool.Stop()
	}()

	// Consume results
	// Standard: Range over results allows us to read exactly as many as were sent
	// until the pool closes the results channel.
	for res := range pool.results {
		fmt.Printf("Received Result for Job %d: %d\n", res.JobID, res.Sum)
	}

	fmt.Println("All jobs finished. Exiting.")
}
