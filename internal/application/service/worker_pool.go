package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openspec/api-scheduler-flow-engine/internal/infrastructure/queue"
	"github.com/openspec/api-scheduler-flow-engine/pkg/logger"
)

type WorkerPool struct {
	maxWorkers int
	jobQueue   chan uuid.UUID
	queue      queue.Queue
	executor   *ExecutorService
	wg         sync.WaitGroup
	quit       chan struct{}
	useRedis   bool
}

//func NewWorkerPool(maxWorkers int, executor *ExecutorService) *WorkerPool {
//	return &WorkerPool{
//		maxWorkers: maxWorkers,
//		jobQueue:   make(chan uuid.UUID, 1000), // Buffer size can be configured
//		executor:   executor,
//		quit:       make(chan struct{}),
//		useRedis:   false,
//	}
//}

func NewWorkerPool(maxWorkers int, executor *ExecutorService, redisQueue queue.Queue) *WorkerPool {
	return &WorkerPool{
		maxWorkers: maxWorkers,
		queue:      redisQueue,
		executor:   executor,
		quit:       make(chan struct{}),
		useRedis:   true,
	}
}

func (p *WorkerPool) Start() {
	logger.Info("Starting worker pool", "workers", p.maxWorkers, "useRedis", p.useRedis)
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
	if p.useRedis {
		// Dispatch ke Redis queue
		if err := p.queue.Enqueue(ctx, executionID); err != nil {
			logger.Error("Failed to enqueue to Redis", "error", err, "execution_id", executionID)
			return err
		}
		logger.Debug("Execution dispatched to Redis queue", "execution_id", executionID)
		return nil
	}

	// Fallback ke in-memory channel
	select {
	case p.jobQueue <- executionID:
		logger.Debug("Execution dispatched to worker pool", "execution_id", executionID)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		logger.Warn("Worker pool job queue is full", "execution_id", executionID)
		return fmt.Errorf("job queue is full")
	}
}

func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()
	logger.Debug("Worker started", "worker_id", id)

	for {
		select {
		//case executionID := <-p.jobQueue:
		//	logger.Debug("Worker picked up execution", "worker_id", id, "execution_id", executionID)
		//	p.executor.Execute(context.Background(), executionID)
		case <-p.quit:
			logger.Debug("Worker stopping", "worker_id", id)
			return
		default:
			var executionID uuid.UUID
			var err error

			if p.useRedis {
				// Consume dari Redis queue dengan timeout
				executionID, err = p.queue.DequeueWithTimeout(context.Background(), 2*time.Second)
				if err != nil {
					// Log error jika bukan timeout
					logger.Debug("Dequeue error or timeout", "worker_id", id, "error", err)
					continue
				}
				if executionID == uuid.Nil {
					// Timeout, tidak ada item, lanjut loop
					continue
				}
			} else {
				// Consume dari in-memory channel
				select {
				case executionID = <-p.jobQueue:
					// Got a job
				case <-p.quit:
					logger.Debug("Worker stopping", "worker_id", id)
					return
				}
			}

			logger.Info("Worker processing execution", "worker_id", id, "execution_id", executionID)
			p.executor.Execute(context.Background(), executionID)
			logger.Info("Worker finished execution", "worker_id", id, "execution_id", executionID)
		}
	}
}
