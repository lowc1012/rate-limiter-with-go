package ratelimiter

import (
    "context"
)

type Request struct {
    Key string
}

type State uint32

const (
    Deny State = iota
    Allow
)

type Result struct {
    State            State
    RequestLimit     uint32
    RemainingTimeSec uint32
}

// Type defines the type of rate limiter.
type Type uint32

const (
    TokenBucketLimiterType Type = iota
    LeakyBucketLimiterType
    FixedWindowLimiterType
    SlidingWindowLimiterType
)

// RateLimiter defines the interface for a rate limiter.
type RateLimiter interface {
    Run(ctx context.Context, req *Request) (*Result, error)
    Type() Type
}
