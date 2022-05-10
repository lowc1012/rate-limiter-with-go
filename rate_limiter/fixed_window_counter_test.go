package rate_limiter

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func TestFixedWindowStrategy_Run(t *testing.T) {
	var now = time.Date(2022, 5, 10, 9, 15, 0, 0, time.UTC)

	var tests = []struct {
		name       string
		runs       int
		request    *Request
		lastResult *Result
		lastErr    error
		advance    time.Duration
	}{
		{
			name: "returns Allow for request under limit",
			runs: 50,
			request: &Request{
				Key:      "user",
				Limit:    60,
				Duration: time.Minute,
			},
			lastResult: &Result{
				State:         Allow,
				TotalRequests: 50,
				ExpiredAt:     time.Date(2022, 5, 10, 9, 16, 0, 0, time.UTC),
			},
		},
		{
			name: "returns Deny for request under limit",
			runs: 51,
			request: &Request{
				Key:      "user",
				Limit:    50,
				Duration: time.Minute,
			},
			lastResult: &Result{
				State:         Deny,
				TotalRequests: 51,
				ExpiredAt:     time.Date(2022, 5, 10, 9, 16, 0, 0, time.UTC),
			},
		},
		{
			name: "key expires and start again (returns Allow)",
			runs: 100,
			request: &Request{
				Key:      "user",
				Limit:    100,
				Duration: time.Minute,
			},
			lastResult: &Result{
				State:         Allow,
				TotalRequests: 40,
				ExpiredAt:     time.Date(2022, 5, 10, 9, 17, 0, 0, time.UTC),
			},
			advance: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := miniredis.Run()
			assert.Nil(t, err)
			defer server.Close()

			client := redis.NewClient(&redis.Options{
				Addr: server.Addr(),
			})
			defer client.Close()

			strategy := NewFixedWindowStrategy(client, func() time.Time {
				return now
			})

			var lastResult *Result
			var lastErr error
			for i := 0; i < tt.runs; i++ {
				lastResult, lastErr = strategy.Run(context.Background(), tt.request)
				if tt.advance != 0 {
					server.FastForward(tt.advance)
					now = now.Add(tt.advance)
				}
			}

			assert.Equal(t, tt.lastResult, lastResult)
			if lastErr != nil {
				assert.Equal(t, tt.lastErr, lastErr)
			} else {
				assert.Nil(t, lastErr)
			}
		})
	}
}
