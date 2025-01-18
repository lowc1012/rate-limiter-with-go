package strategy

import (
    "context"
    "errors"
    "fmt"
    "sync"
    "time"

    "github.com/redis/go-redis/v9"
)

type tokenBucketRecord struct {
    TokenCount float64 `redis:"tokenCount"`
    LastFilled int64   `redis:"lastFilled"`
}

type TokenBucket struct {
    sync.Mutex
    client    *redis.Client
    rate      float64 // refill rate per second
    capacity  uint32  // max number of tokens
    keyPrefix string  // key prefix bucket records
}

// NewTokenBucket creates a new TokenBucket with the given refill rate and capacity
func NewTokenBucket(client *redis.Client, rate float64, capacity uint32) *TokenBucket {
    return &TokenBucket{
        client:    client,
        rate:      rate,
        capacity:  capacity,
        keyPrefix: "token_bucket:",
    }
}

func (b *TokenBucket) Capacity() uint32 {
    return b.capacity
}

// Take removes tokens from the bucket and returns the number of tokens taken
func (b *TokenBucket) Take(ctx context.Context, ip string, amount uint32) uint32 {
    b.Lock()
    defer b.Unlock()

    key := b.getKey(ip)
    b.refill(ctx, key, time.Now())

    var bucketRec tokenBucketRecord
    err := b.client.HGetAll(ctx, key).Scan(&bucketRec)
    if err != nil || bucketRec.TokenCount < float64(amount) {
        return 0
    }

    bucketRec.TokenCount -= float64(amount)
    b.client.HSet(ctx, key, map[string]interface{}{
        "tokenCount": bucketRec.TokenCount,
        "lastFilled": bucketRec.LastFilled,
    }).Err()
    return amount
}

// refill refills tokens to the bucket based on the refill rate
func (b *TokenBucket) refill(ctx context.Context, key string, current time.Time) {
    var bucketRec tokenBucketRecord
    err := b.client.HGetAll(ctx, key).Scan(&bucketRec)
    if err != nil {
        if !errors.Is(err, redis.Nil) {
            panic(fmt.Sprintf("failed to get bucket record from Redis: %v", err))
        }
    }

    elapsedTime := current.Sub(time.Unix(bucketRec.LastFilled, 0))
    bucketRec.TokenCount += elapsedTime.Seconds() * b.rate
    if bucketRec.TokenCount > float64(b.capacity) {
        bucketRec.TokenCount = float64(b.capacity)
    }
    bucketRec.LastFilled = current.Unix()
    b.client.HSet(ctx, key, bucketRec).Err()
}

// getKey returns the key for the bucket record
func (b *TokenBucket) getKey(ip string) string {
    return b.keyPrefix + ip
}
