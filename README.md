# rate-limiter-with-go

## Rate limiting algorithms
This implementation supports the following rate limiting strategies:

- Token bucket
- Leaky bucket
- Fixed window (TODO)
- Sliding window (TODO)

## Storage
This implementation supports the following storage backends:
- Redis
- In-memory (TODO)

## Run this example
    go run main.go

    # execute this instruction many times
    curl --header "X-Forwarded-For: 127.0.0.1" localhost:8080/api/v1/hello