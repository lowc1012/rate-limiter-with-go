package main

import (
    "net/http"

    "github.com/lowc1012/rate-limiter-with-go/internal/log"
    "github.com/lowc1012/rate-limiter-with-go/internal/ratelimiter"
    "github.com/lowc1012/rate-limiter-with-go/internal/utils"
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

func HelloHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello, World!"))
}

func main() {
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    mux := http.NewServeMux()
    mux.HandleFunc("/api/v1/hello", HelloHandler)

    // create a rate limiter with a specific strategy
    limiter := ratelimiter.NewTokenBucketLimiter(redisClient, 0.5, 3)

    config := &ratelimiter.Config{
        Extractor: utils.NewHTTPHeadersExtractor("X-Forwarded-For"),
        Limiter:   limiter,
    }

    wrappedMux := ratelimiter.NewHTTPRateLimiterHandler(mux, config)

    // use wrappedMux instead of mux as root handler
    log.Logger().Info("Run a server listening to localhost:8080")
    err := http.ListenAndServe("localhost:8080", wrappedMux)
    if err != nil {
        log.Logger().Fatal("Failed to serve handler", zap.Error(err))
    }
}
