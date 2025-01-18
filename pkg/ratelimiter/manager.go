package ratelimiter

import (
    "fmt"
    "net/http"
    "strconv"

    "github.com/lowc1012/rate-limiter-with-go/pkg/utils"
)

var (
    stateStrings = map[State]string{
        Allow: "Allow",
        Deny:  "Deny",
    }
)

const (
    rateLimitMaxRequests = "X-Ratelimit-Max-Requests"
    rateLimitState       = "X-Ratelimit-State"
    rateLimitRetryAfter  = "X-Ratelimit-Retry-After"
)

// Config defines the configuration for the rate limiter handler.
type Config struct {
    Extractor utils.Extractor
    Limiter   RateLimiter
}

type httpRateLimiterHandler struct {
    handler http.Handler
    config  *Config
}

// NewHTTPRateLimiterHandler wraps an existing http.Handler object performing rate limiting before
// sending the request to the wrapped handler. If any errors happen while trying to rate limit a request
// or if the request is denied, the rate limiting handler will send a response to the client and will not
// call the wrapped handler.
func NewHTTPRateLimiterHandler(originalHandler http.Handler, config *Config) http.Handler {
    return &httpRateLimiterHandler{
        handler: originalHandler,
        config:  config,
    }
}

func (h *httpRateLimiterHandler) writeResponse(writer http.ResponseWriter, status int, msg string, args ...interface{}) {
    writer.Header().Set("Content-Type", "text/plain")
    writer.WriteHeader(status)
    if _, err := writer.Write([]byte(fmt.Sprintf(msg, args...))); err != nil {
        fmt.Printf("failed to write body to HTTP request: %v", err)
    }
}

// ServeHTTP performs rate limiting with the configuration it was provided and if there were no errors
// and the request was allowed it is sent to the wrapped handler. It also adds rate limiting headers that will be
// sent to the client to make it aware of what state it is in terms of rate limiting.
func (h *httpRateLimiterHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
    key, err := h.config.Extractor.Extract(request)
    if err != nil {
        h.writeResponse(writer, http.StatusBadRequest, "failed to collect rate limiting key from request: %v", err)
        return
    }

    // run the rate limiting
    result, err := h.config.Limiter.Run(request.Context(), &Request{
        Key: key,
    })

    if err != nil {
        h.writeResponse(writer, http.StatusInternalServerError, "failed to run rate limiting for request: %v", err)
        return
    }

    // set the rate limiting headers both on allow or deny results so the client knows what is going on
    writer.Header().Set(rateLimitMaxRequests, strconv.FormatUint(uint64(result.RequestLimit), 10))
    writer.Header().Set(rateLimitState, stateStrings[result.State])
    writer.Header().Set(rateLimitRetryAfter, strconv.Itoa(int(result.RemainingTimeSec)))

    // when the state is Deny, just return a 429 response to the client and stop the request handling flow
    if result.State == Deny {
        h.writeResponse(writer, http.StatusTooManyRequests, "you have sent too many requests to this service, slow down please")
        return
    }

    // if the request was not denied we assume it was allowed and call the wrapped handler.
    // by leaving this to the end we make sure the wrapped handler is only called once and doesn't have to worry
    // about any rate limiting at all (it doesn't even have to know there was rate limiting happening for this request)
    // as we have already set the headers, so when the handler flushes the response the headers above will be sent.
    h.handler.ServeHTTP(writer, request)
}
