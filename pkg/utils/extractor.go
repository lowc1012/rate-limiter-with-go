package utils

import (
    "fmt"
    "net/http"
    "strings"
)

// Extractor represents the way we will extract a key from an HTTP request, this could be
// a value from a header, request path, method used, user authentication information, any information that
// is available at the HTTP request that wouldn't cause side effects if it was collected (this object shouldn't
// read the body of the request).
type Extractor interface {
    Extract(r *http.Request) (string, error)
}

type httpHeaderExtractor struct {
    headers []string
}

// NewHTTPHeadersExtractor creates a new HTTP header extractor
func NewHTTPHeadersExtractor(headers ...string) Extractor {
    return &httpHeaderExtractor{headers: headers}
}

// Extract extracts a collection of http headers and joins them to build the key that will be used for
// rate limiting. You should use headers that are guaranteed to be unique for a client.
func (h *httpHeaderExtractor) Extract(r *http.Request) (string, error) {
    values := make([]string, 0, len(h.headers))

    for _, key := range h.headers {
        // if we can't find a value for the headers, give up and return an error.
        if value := strings.TrimSpace(r.Header.Get(key)); value == "" {
            return "", fmt.Errorf("the header %v must have a value set", key)
        } else {
            values = append(values, value)
        }
    }

    return strings.Join(values, "-"), nil
}
