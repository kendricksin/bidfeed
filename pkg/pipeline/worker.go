// pkg/pipeline/worker.go

package pipeline

import (
	"context"
	"log"
	"sync"
)

// Job represents a unit of work to be processed
type Job interface {
	Process(ctx context.Context) error
	ID() string
}

// WorkerPool manages a pool of workers for processing jobs
type WorkerPool struct {
	numWorkers int
	jobs       chan Job
	results    chan error
	wg         sync.WaitGroup
	logger     *log.Logger
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(numWorkers int, queueSize int, logger *log.Logger) *WorkerPool {
	return &WorkerPool{
		numWorkers: numWorkers,
		jobs:       make(chan Job, queueSize),
		results:    make(chan error, queueSize),
		logger:     logger,
	}
}

// Start initializes the worker pool and begins processing jobs
func (p *WorkerPool) Start(ctx context.Context) {
	// Start the workers
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
}

// Submit adds a new job to the pool
func (p *WorkerPool) Submit(job Job) {
	p.jobs <- job
}

// Stop gracefully shuts down the worker pool
func (p *WorkerPool) Stop() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}

// Results returns the channel for receiving job results
func (p *WorkerPool) Results() <-chan error {
	return p.results
}

// worker processes jobs from the pool
func (p *WorkerPool) worker(ctx context.Context, id int) {
	defer p.wg.Done()

	for job := range p.jobs {
		select {
		case <-ctx.Done():
			p.logger.Printf("Worker %d stopping due to context cancellation", id)
			return
		default:
			p.logger.Printf("Worker %d processing job %s", id, job.ID())
			err := job.Process(ctx)
			if err != nil {
				p.logger.Printf("Worker %d encountered error processing job %s: %v", id, job.ID(), err)
			}
			p.results <- err
		}
	}
}

// ProcessingJob implements the Job interface for feed entry processing
type ProcessingJob struct {
	jobID      string
	ProcessFn  func(ctx context.Context) error
	identifier string
}

// NewProcessingJob creates a new processing job
func NewProcessingJob(id string, fn func(ctx context.Context) error) *ProcessingJob {
	return &ProcessingJob{
		jobID:      id,
		ProcessFn:  fn,
		identifier: id,
	}
}

// Process executes the job's processing function
func (j *ProcessingJob) Process(ctx context.Context) error {
	return j.ProcessFn(ctx)
}

// ID returns the job's identifier
func (j *ProcessingJob) ID() string {
	return j.identifier
}
