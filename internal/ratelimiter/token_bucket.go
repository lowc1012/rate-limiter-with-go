package ratelimiter

import (
    "context"

    "github.com/lowc1012/rate-limiter-with-go/internal/ratelimiter/algorithm"
    "github.com/redis/go-redis/v9"
)

// TokenBucketLimiter is a popular approach that regulates the flow of requests using a token bucket.
// Each request consumes a token from the bucket, and once the bucket is empty, no more requests are
// allowed until the bucket is refilled.
type TokenBucketLimiter struct {
    impl *algorithm.TokenBucket
}

// NewTokenBucketLimiter creates a new TokenBucketLimiter with the given refill rate and capacity
func NewTokenBucketLimiter(client *redis.Client, rate float64, capacity uint32) *TokenBucketLimiter {
    return &TokenBucketLimiter{
        impl: algorithm.NewTokenBucket(client, rate, capacity),
    }
}

func (l *TokenBucketLimiter) Type() Type {
    return TokenBucketLimiterType
}

func (l *TokenBucketLimiter) Run(ctx context.Context, req *Request) (*Result, error) {
    taken := l.impl.Take(ctx, req.Key, 1)
    if taken == 0 {
        return &Result{State: Deny, RequestLimit: l.impl.Capacity()}, nil
    }
    return &Result{State: Allow, RequestLimit: l.impl.Capacity()}, nil
}
