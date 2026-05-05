package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/pkg/logger"
)

type WorkerPool struct {
	maxWorkers int
	jobQueue   chan uuid.UUID
	executor   *ExecutorService
	wg         sync.WaitGroup
	quit       chan struct{}
}

func NewWorkerPool(maxWorkers int, executor *ExecutorService) *WorkerPool {
	return &WorkerPool{
		maxWorkers: maxWorkers,
		jobQueue:   make(chan uuid.UUID, 1000), // Buffer size can be configured
		executor:   executor,
		quit:       make(chan struct{}),
	}
}

func (p *WorkerPool) Start() {
	logger.Info("Starting worker pool", "workers", p.maxWorkers)
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

func (p *WorkerPool) Stop() {
	logger.Info("Stopping worker pool...")
	close(p.quit)
	p.wg.Wait()
	logger.Info("Worker pool stopped")
}

func (p *WorkerPool) Dispatch(ctx context.Context, executionID uuid.UUID) error {
	select {
	case p.jobQueue <- executionID:
		logger.Debug("Execution dispatched to worker pool", "execution_id", executionID)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		// If job queue is full and we can't enqueue within 5 seconds, we would ideally fall back to Redis queue.
		// For now, just block or return error if channel is full.
		// Since buffer is 1000, it's unlikely to block unless severely overloaded.
		p.jobQueue <- executionID // block until space is available
		return nil
	}
}

func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()
	logger.Debug("Worker started", "worker_id", id)

	for {
		select {
		case executionID := <-p.jobQueue:
			logger.Debug("Worker picked up execution", "worker_id", id, "execution_id", executionID)
			p.executor.Execute(context.Background(), executionID)
		case <-p.quit:
			logger.Debug("Worker stopping", "worker_id", id)
			return
		}
	}
}
