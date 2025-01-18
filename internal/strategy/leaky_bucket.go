package strategy

import (
    "context"
    "errors"
    "fmt"
    "strconv"
    "sync"
    "time"

    "github.com/redis/go-redis/v9"
)

type bucketRecord struct {
    TokenCount   float64 `redis:"tokenCount"`
    LastLeakTime int64   `redis:"lastLeakTime"`
}

type LeakyBucket struct {
    sync.Mutex
    client       *redis.Client
    rate         float64       // consume rate per second
    queue        chan struct{} // FIFO queue to store request
    tokenCount   float64
    capacity     uint32    // maximum capacity of the bucket
    lastLeakTime time.Time // last time the bucket was refilled
}

func NewLeakyBucket(client *redis.Client, rate float64, capacity uint32) *LeakyBucket {
    return &LeakyBucket{
        client:       client,
        rate:         rate,
        queue:        make(chan struct{}, capacity),
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
    b.Lock()
    defer b.Unlock()

    key := b.getKey(ip)
    err := b.leak(ctx, ip)
    if err != nil {
        return err
    }

    var bucketRec bucketRecord
    err = b.client.HGetAll(ctx, key).Scan(&bucketRec)
    if err != nil && !errors.Is(err, redis.Nil) {
        return err
    }

    if uint32(bucketRec.TokenCount) < b.capacity {
        _, err = b.client.HIncrByFloat(ctx, key, "tokenCount", 1).Result()
        return err
    }
    return fmt.Errorf("bucket is full")
}

func (b *LeakyBucket) getKey(ip string) string {
    return ip
}

// leak consume a token in queue
func (b *LeakyBucket) leak(ctx context.Context, ip string) error {
    key := b.getKey(ip)
    now := time.Now().Unix()
    var bucketRec bucketRecord
    err := b.client.HGetAll(ctx, key).Scan(&bucketRec)
    if err != nil && !errors.Is(err, redis.Nil) {
        return err
    }

    elapsedTime := float64(now - bucketRec.LastLeakTime)
    tokensToLeak := elapsedTime * b.rate

    bucketRec.TokenCount -= tokensToLeak
    if bucketRec.TokenCount < 0 {
        bucketRec.TokenCount = 0
    }

    hash := []string{
        "tokenCount", strconv.FormatFloat(bucketRec.TokenCount, 'f', -1, 64),
        "lastLeakTime", strconv.FormatInt(now, 10),
    }
    _, err = b.client.HSet(ctx, key, hash).Result()
    return err
}
