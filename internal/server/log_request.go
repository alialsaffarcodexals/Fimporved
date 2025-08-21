package server

import (
        "fmt"
        "net/http"
        "time"
)

/*
LogRequest prints the HTTP method, path and how long the handler took.

Example:

    mux := http.NewServeMux()
    handler := server.LogRequest(mux)
    http.ListenAndServe(":8080", handler)
*/
func LogRequest(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                start := time.Now()
                next.ServeHTTP(w, r)
                fmt.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start))
        })
}

