package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	retryKeyPrefix = "flow_engine:retry"
	retryKeyTTL    = 24 * time.Hour
)

// Queue defines the contract for job queue operations.
type Queue interface {
	Enqueue(ctx context.Context, executionID uuid.UUID) error
	Dequeue(ctx context.Context) (uuid.UUID, error)
	DequeueWithTimeout(ctx context.Context, timeout time.Duration) (uuid.UUID, error)
	Close() error
}

// RetryTracker defines the contract for tracking retry state in Redis.
// Data ini visible di RedisInsight sebagai Hash per execution step.
type RetryTracker interface {
	TrackRetryAttempt(ctx context.Context, executionID uuid.UUID, stepOrder int, attempt, maxRetries int, action, lastError string, nextRetryAt time.Time) error
	TrackRetryCompleted(ctx context.Context, executionID uuid.UUID, stepOrder int) error
	TrackRetryFailed(ctx context.Context, executionID uuid.UUID, stepOrder int, lastError string) error
	DeleteRetryState(ctx context.Context, executionID uuid.UUID, stepOrder int) error
}

// RedisQueue implements both Queue and RetryTracker.
type RedisQueue struct {
	client   *redis.Client
	queueKey string
}

func NewRedisQueue(redisURL string) (*RedisQueue, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisQueue{
		client:   client,
		queueKey: "flow_engine:execution_queue",
	}, nil
}

// ── Queue implementation ──────────────────────────────────────────────────────

func (q *RedisQueue) Enqueue(ctx context.Context, executionID uuid.UUID) error {
	return q.client.LPush(ctx, q.queueKey, executionID.String()).Err()
}

func (q *RedisQueue) Dequeue(ctx context.Context) (uuid.UUID, error) {
	res, err := q.client.BRPop(ctx, 0, q.queueKey).Result()
	if err != nil {
		return uuid.Nil, err
	}
	if len(res) < 2 {
		return uuid.Nil, fmt.Errorf("empty result from brpop")
	}
	return uuid.Parse(res[1])
}

func (q *RedisQueue) DequeueWithTimeout(ctx context.Context, timeout time.Duration) (uuid.UUID, error) {
	res, err := q.client.BRPop(ctx, timeout, q.queueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return uuid.Nil, nil
		}
		return uuid.Nil, err
	}
	if len(res) < 2 {
		return uuid.Nil, fmt.Errorf("empty result from brpop")
	}
	return uuid.Parse(res[1])
}

func (q *RedisQueue) Close() error {
	return q.client.Close()
}

func (q *RedisQueue) Len(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, q.queueKey).Result()
}

// ── RetryTracker implementation ───────────────────────────────────────────────

// retryKey menghasilkan Redis key format:
// flow_engine:retry:{execution_id}:{step_order}
func retryKey(executionID uuid.UUID, stepOrder int) string {
	return fmt.Sprintf("%s:%s:%d", retryKeyPrefix, executionID.String(), stepOrder)
}

// TrackRetryAttempt menulis state retry ke Redis Hash.
// Visible di RedisInsight sebagai key: flow_engine:retry:{execution_id}:{step_order}
func (q *RedisQueue) TrackRetryAttempt(
	ctx context.Context,
	executionID uuid.UUID,
	stepOrder int,
	attempt, maxRetries int,
	action, lastError string,
	nextRetryAt time.Time,
) error {
	key := retryKey(executionID, stepOrder)
	now := time.Now().UTC().Format(time.RFC3339)

	fields := map[string]interface{}{
		"execution_id":    executionID.String(),
		"step_order":      stepOrder,
		"action":          action,
		"attempt":         attempt,
		"max_retries":     maxRetries,
		"status":          "retrying",
		"last_error":      lastError,
		"last_attempt_at": now,
		"next_retry_at":   nextRetryAt.UTC().Format(time.RFC3339),
	}

	pipe := q.client.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, retryKeyTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// TrackRetryCompleted menandai step berhasil setelah retry.
func (q *RedisQueue) TrackRetryCompleted(
	ctx context.Context,
	executionID uuid.UUID,
	stepOrder int,
) error {
	key := retryKey(executionID, stepOrder)
	now := time.Now().UTC().Format(time.RFC3339)

	pipe := q.client.Pipeline()
	pipe.HSet(ctx, key, map[string]interface{}{
		"status":       "completed",
		"completed_at": now,
		"last_error":   "",
	})
	// Perpendek TTL ke 1 jam setelah completed — tidak perlu lama di Redis
	pipe.Expire(ctx, key, time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

// TrackRetryFailed menandai step gagal setelah semua retry habis.
func (q *RedisQueue) TrackRetryFailed(
	ctx context.Context,
	executionID uuid.UUID,
	stepOrder int,
	lastError string,
) error {
	key := retryKey(executionID, stepOrder)
	now := time.Now().UTC().Format(time.RFC3339)

	pipe := q.client.Pipeline()
	pipe.HSet(ctx, key, map[string]interface{}{
		"status":        "failed",
		"failed_at":     now,
		"last_error":    lastError,
		"next_retry_at": "",
	})
	pipe.Expire(ctx, key, retryKeyTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// DeleteRetryState menghapus retry state dari Redis (opsional, untuk cleanup manual).
func (q *RedisQueue) DeleteRetryState(
	ctx context.Context,
	executionID uuid.UUID,
	stepOrder int,
) error {
	return q.client.Del(ctx, retryKey(executionID, stepOrder)).Err()
}
