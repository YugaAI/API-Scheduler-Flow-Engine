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
	once       sync.Once
	useRedis   bool

	// ctx dan cancel digunakan untuk propagate cancellation ke execution yang sedang berjalan.
	// Saat Stop() dipanggil, cancel() dipanggil — semua execution aktif mendapat sinyal cancel.
	ctx    context.Context
	cancel context.CancelFunc
}

func NewWorkerPool(maxWorkers int, executor *ExecutorService, redisQueue queue.Queue) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		maxWorkers: maxWorkers,
		queue:      redisQueue,
		executor:   executor,
		quit:       make(chan struct{}),
		once:       sync.Once{},
		useRedis:   true,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (p *WorkerPool) Start() {
	logger.Info("Starting worker pool", "workers", p.maxWorkers, "useRedis", p.useRedis)
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop idempotent — aman dipanggil berkali-kali.
// cancel() menyebar ke semua execution yang sedang berjalan via context.
func (p *WorkerPool) Stop() {
	p.once.Do(func() {
		logger.Info("Stopping worker pool...")
		p.cancel()      // sinyal ke semua execution aktif untuk berhenti
		close(p.quit)   // sinyal ke semua goroutine worker untuk exit loop
	})
	p.wg.Wait()
	logger.Info("Worker pool stopped")
}

func (p *WorkerPool) Dispatch(ctx context.Context, executionID uuid.UUID) error {
	if p.useRedis {
		if err := p.queue.Enqueue(ctx, executionID); err != nil {
			logger.Error("Failed to enqueue to Redis", "error", err, "execution_id", executionID)
			return err
		}
		logger.Debug("Execution dispatched to Redis queue", "execution_id", executionID)
		return nil
	}

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
		// Cek quit signal SEBELUM blocking dequeue
		select {
		case <-p.quit:
			logger.Debug("Worker stopping", "worker_id", id)
			return
		default:
		}

		if p.useRedis {
			executionID, err := p.queue.DequeueWithTimeout(context.Background(), 2*time.Second)
			if err != nil {
				select {
				case <-p.quit:
					logger.Debug("Worker stopping", "worker_id", id)
					return
				default:
					logger.Debug("Dequeue timeout or error", "worker_id", id, "error", err)
					continue
				}
			}
			if executionID == uuid.Nil {
				continue
			}

			logger.Info("Worker processing execution", "worker_id", id, "execution_id", executionID)

			// Propagate p.ctx ke executor — saat Stop() dipanggil,
			// p.cancel() akan menyebar ke semua execution yang sedang berjalan
			p.executor.Execute(p.ctx, executionID)

			logger.Info("Worker finished execution", "worker_id", id, "execution_id", executionID)

		} else {
			select {
			case executionID := <-p.jobQueue:
				logger.Info("Worker processing execution", "worker_id", id, "execution_id", executionID)
				p.executor.Execute(p.ctx, executionID)
				logger.Info("Worker finished execution", "worker_id", id, "execution_id", executionID)
			case <-p.quit:
				logger.Debug("Worker stopping", "worker_id", id)
				return
			}
		}
	}
}
