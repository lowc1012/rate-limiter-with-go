package rate_limiter

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/lowc1012/rate-limiter-with-go/log"
)

var _ Strategy = &slidingWindowStrategy{}

const (
	sortedSetMax = "+inf"
	sortedSetMin = "-inf"
)

type slidingWindowStrategy struct {
	client *redis.Client
	now    func() time.Time
}

func NewSlidingWindowStrategy(client *redis.Client, now func() time.Time) *slidingWindowStrategy {
	return &slidingWindowStrategy{
		client,
		now,
	}
}

func (s *slidingWindowStrategy) Run(ctx context.Context, r *Request) (*Result, error) {

	now := s.now()
	expiresAt := now.Add(r.Duration)
	minimum := now.Add(-r.Duration)

	// Check whether the user has been already over the limit or not
	result, err := s.client.ZCount(ctx, r.Key, strconv.FormatInt(minimum.UnixMilli(), 10), sortedSetMax).Uint64()
	if err == nil && result >= r.Limit {
		return &Result{
			Deny,
			result,
			expiresAt,
		}, nil
	}

	// Remove expired requests
	p := s.client.Pipeline()
	removeByScore := p.ZRemRangeByScore(ctx, r.Key, "0", strconv.FormatInt(minimum.UnixMilli(), 10))

	// assign uuid to each request
	id := uuid.New()

	// Add the current request (member) to the sorted set stored at Key
	addResult := p.ZAdd(ctx, r.Key, &redis.Z{
		Member: id.String(),
		Score:  float64(now.UnixMilli()),
	})

	// Count all the member (non-expired requests) in the sorted set
	count := p.ZCount(ctx, r.Key, sortedSetMin, sortedSetMax)

	if _, err = p.Exec(ctx); err != nil {
		log.Logger().Error(fmt.Sprintf("Failed to execute sorted set pipeline for key %v", r.Key))
		return nil, err
	}

	if err = removeByScore.Err(); err != nil {
		log.Logger().Error(fmt.Sprintf("Failed to remove items from key %v", r.Key))
		return nil, err
	}

	if err = addResult.Err(); err != nil {
		log.Logger().Error(fmt.Sprintf("Failed add item to key %v", r.Key))
		return nil, err
	}

	countResult, err := count.Result()
	if err != nil {
		log.Logger().Error(fmt.Sprintf("Failed to count items for key %v", r.Key))
		return nil, err
	}

	numRequest := uint64(countResult)
	if numRequest > r.Limit {
		return &Result{
			Deny,
			numRequest,
			expiresAt,
		}, nil
	}

	return &Result{
		Allow,
		numRequest,
		expiresAt,
	}, nil
}
