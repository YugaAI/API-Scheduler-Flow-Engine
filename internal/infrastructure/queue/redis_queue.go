package queue

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	client *redis.Client
	queueKey string
}

func NewRedisQueue(redisURL string) (*RedisQueue, error) {
	// Parse redis URL, for simplicity we assume localhost:6379 or similar simple host:port
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

func (q *RedisQueue) Enqueue(ctx context.Context, executionID uuid.UUID) error {
	return q.client.LPush(ctx, q.queueKey, executionID.String()).Err()
}

func (q *RedisQueue) Dequeue(ctx context.Context) (uuid.UUID, error) {
	// BRPOP blocks until an item is available or timeout (0 = wait forever)
	res, err := q.client.BRPop(ctx, 0, q.queueKey).Result()
	if err != nil {
		return uuid.Nil, err
	}
	// res[0] is the key, res[1] is the value
	if len(res) < 2 {
		return uuid.Nil, fmt.Errorf("empty result from brpop")
	}

	return uuid.Parse(res[1])
}
