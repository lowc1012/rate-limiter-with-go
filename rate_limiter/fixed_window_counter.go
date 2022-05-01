package rate_limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/lowc1012/rate-limiter-with-go/log"
	"go.uber.org/zap"
)

// ensure that counterStrategy satisfies an interface Strategy
var _ Strategy = &counterStrategy{}

type counterStrategy struct {
	client  *redis.Client
	timeNow func() time.Time
}

func NewCounterStrategy(c *redis.Client, now func() time.Time) *counterStrategy {
	return &counterStrategy{
		c,
		now,
	}
}

func (c *counterStrategy) Run(ctx context.Context, r *Request) (*Result, error) {
	// Using Redis pipeline to optimize network performance
	p := c.client.Pipeline()
	incrResult := p.Incr(ctx, r.Key)
	ttlResult := p.TTL(ctx, r.Key)

	// make sure all errors are handled
	if _, err := p.Exec(ctx); err != nil {
		log.Logger().Error("Failed to execute increase to key", zap.Error(err))
		return nil, err
	}

	// get current window count
	totalRequests, err := incrResult.Result()
	if err != nil {
		log.Logger().Error("Failed to increase key", zap.Error(err))
		return nil, err
	}

	var ttlDuration time.Duration
	duration, err := ttlResult.Result()
	if err != nil || duration == -1 {
		// returns duration = -1 if the key exists but has no associated expire.
		// returns duration = -2 if the key does not exist.
		ttlDuration = r.Duration
		if err := c.client.Expire(ctx, r.Key, r.Duration).Err(); err != nil {
			log.Logger().Error("Failed to set an expiration to key", zap.Error(err))
			return nil, err
		}
	} else {
		ttlDuration = duration
	}

	expiresAt := c.timeNow().Add(ttlDuration)
	log.Logger().Info(fmt.Sprintf("The key [%s] will expire", r.Key),
		zap.String("ttlDuration", ttlDuration.String()))

	if requests := uint64(totalRequests); requests > r.Limit {
		return &Result{
			State:         Deny,
			TotalRequests: requests,
			ExpiredAt:     expiresAt,
		}, nil
	} else {
		return &Result{
			State:         Allow,
			TotalRequests: requests,
			ExpiredAt:     expiresAt,
		}, nil
	}
}
