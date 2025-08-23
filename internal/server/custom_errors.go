package server

// This middleware decorates a ServeMux with friendly error pages.
// It intercepts panics to return a 500 page and records the status
// code of responses so that 404 and 400 pages can be rendered via
// templates. Other status codes pass through unchanged.

import (
    "net/http"
    "forum/internal/app"
)

// WithCustomErrors wraps the provided ServeMux and uses the
// templates stored on the App to render custom error pages. If a
// panic occurs during request handling a 500 page is shown. If the
// handler writes a 404 or 400 status code the corresponding error
// page is rendered. All other responses are passed through.
func WithCustomErrors(next *http.ServeMux, app *app.App) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                // Render a 500 page if available. Otherwise fall back to
                // the default text. We do not expose the panic value to
                // the user for security reasons.
                w.Header().Set("Content-Type", "text/html; charset=utf-8")
                w.Header().Set("Cache-Control", "no-store")
                w.WriteHeader(http.StatusInternalServerError)
                if tpl, ok := app.Templates["500.html"]; ok {
                    tpl.ExecuteTemplate(w, "500.html", AppTemplateData(r, app))
                } else {
                    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                }
            }
        }()

        // Wrap the ResponseWriter to capture the status code written by
        // handlers. If WriteHeader is not called explicitly the code
        // defaults to 200.
        rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        next.ServeHTTP(rw, r)

        // Render custom pages for 404 and 400 codes. The handler
        // might have already written some body, but we'll ignore it
        // and replace with our own template.
        switch rw.statusCode {
        case http.StatusNotFound:
            if tpl, ok := app.Templates["404.html"]; ok {
                w.Header().Set("Content-Type", "text/html; charset=utf-8")
                w.Header().Set("Cache-Control", "no-store")
                tpl.ExecuteTemplate(w, "404.html", AppTemplateData(r, app))
            } else {
                http.NotFound(w, r)
            }
        case http.StatusBadRequest:
            if tpl, ok := app.Templates["400.html"]; ok {
                w.Header().Set("Content-Type", "text/html; charset=utf-8")
                w.Header().Set("Cache-Control", "no-store")
                tpl.ExecuteTemplate(w, "400.html", AppTemplateData(r, app))
            } else {
                http.Error(w, "Bad Request", http.StatusBadRequest)
            }
        }
    })
}

// responseWriter wraps an http.ResponseWriter and records the status
// code written. It forwards Write and WriteHeader calls to the
// underlying writer.
type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}