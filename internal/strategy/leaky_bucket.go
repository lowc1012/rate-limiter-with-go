package strategy

import (
    "context"
    "errors"
    "fmt"
    "sync"
    "time"

    "github.com/redis/go-redis/v9"
)

type LeakyBucket struct {
    sync.Mutex
    client       *redis.Client
    rate         float64       // consume rate per second
    queue        chan struct{} // FIFO queue to store request
    stopChan     chan struct{}
    tokenCount   float64
    capacity     uint32    // maximum capacity of the bucket
    lastLeakTime time.Time // last time the bucket was refilled
}

func NewLeakyBucket(client *redis.Client, rate float64, capacity uint32) *LeakyBucket {
    return &LeakyBucket{
        client:       client,
        rate:         rate,
        queue:        make(chan struct{}, capacity),
        stopChan:     make(chan struct{}),
        tokenCount:   0,
        capacity:     capacity,
        lastLeakTime: time.Now(),
    }
}

func (b *LeakyBucket) Rate() float64 {
    return b.rate
}

func (b *LeakyBucket) Capacity() uint32 {
    return b.capacity
}

// Add adds a request to the bucket if it's not full
func (b *LeakyBucket) Add(ctx context.Context, ip string) error {
    key := b.getKey(ip)
    err := b.leak(ctx, ip)
    if err != nil {
        return err
    }

    tokenCount, err := b.client.HGet(ctx, key, "tokenCount").Float64()
    if err != nil && err != redis.Nil {
        return err
    }

    if tokenCount < float64(b.capacity) {
        _, err = b.client.HIncrByFloat(ctx, key, "tokenCount", 1).Result()
        return err
    }
    return fmt.Errorf("bucket is full")
}

func (b *LeakyBucket) Allow(ctx context.Context, ip string, n uint32) bool {
    key := b.getKey(ip)
    err := b.leak(ctx, ip)
    if err != nil {
        return false
    }

    tokenCount, err := b.client.HGet(ctx, key, "tokenCount").Float64()
    if err != nil && err != redis.Nil {
        return false
    }

    if tokenCount+float64(n) <= float64(b.capacity) {
        return true
    }
    return false
}

func (b *LeakyBucket) getKey(ip string) string {
    return ip
}

// leak consume a token in queue
func (b *LeakyBucket) leak(ctx context.Context, ip string) error {
    key := b.getKey(ip)
    now := time.Now().Unix()
    lastLeakTime, err := b.client.HGet(ctx, key, "lastLeakTime").Int64()
    if err != nil && !errors.Is(err, redis.Nil) {
        return err
    }

    elapsedTime := float64(now - lastLeakTime)
    tokensToLeak := elapsedTime * b.rate
    tokenCount, err := b.client.HGet(ctx, key, "tokenCount").Float64()
    if err != nil && !errors.Is(err, redis.Nil) {
        return err
    }

    tokenCount -= tokensToLeak
    if tokenCount < 0 {
        tokenCount = 0
    }

    _, err = b.client.HSet(ctx, key, "tokenCount", tokenCount, "lastLeakTime", now).Result()
    return err
}
