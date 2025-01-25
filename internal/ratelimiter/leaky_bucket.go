package ratelimiter

import (
    "context"

    "github.com/lowc1012/rate-limiter-with-go/internal/ratelimiter/algorithm"
    "github.com/redis/go-redis/v9"
)

type LeakyBucketLimiter struct {
    impl *algorithm.LeakyBucket
}

func (l *LeakyBucketLimiter) Run(ctx context.Context, req *Request) (*Result, error) {
    err := l.impl.Add(ctx, req.Key)
    if err != nil {
        return &Result{
            State:            Deny,
            RequestLimit:     l.impl.Capacity(),
            RemainingTimeSec: uint32(l.impl.Rate()),
        }, nil
    }
    return &Result{State: Allow, RequestLimit: l.impl.Capacity(), RemainingTimeSec: 0}, nil
}

func (l *LeakyBucketLimiter) Type() Type {
    return LeakyBucketLimiterType
}

func NewLeakyBucketLimiter(client *redis.Client, rate float64, capacity uint32) *LeakyBucketLimiter {
    return &LeakyBucketLimiter{impl: algorithm.NewLeakyBucket(client, rate, capacity)}
}
