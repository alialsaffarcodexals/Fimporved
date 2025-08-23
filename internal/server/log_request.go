package server

// This middleware logs each HTTP request with its method, path and
// duration. It is useful for debugging and performance analysis.

import (
    "fmt"
    "net/http"
    "time"
)

// LogRequest wraps an http.Handler and prints the request method,
// URL path and time taken to process the request. The format is
// similar to: `GET /index 12.345ms`.
func LogRequest(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        fmt.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start))
    })
}