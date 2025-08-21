package server

import (
        "net/http"

        "forum-mvp/internal/app"
)

/*
WithCustomErrors wraps the provided ServeMux to show friendly 404 and
500 pages.

Example:

    mux := http.NewServeMux()
    handler := server.WithCustomErrors(mux, app)
    http.ListenAndServe(":8080", handler)
*/
func WithCustomErrors(next *http.ServeMux, app *app.App) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                defer func() {
                        if err := recover(); err != nil {
                                w.Header().Set("Content-Type", "text/html; charset=utf-8")
                                w.Header().Set("Cache-Control", "no-store")
                                w.WriteHeader(http.StatusInternalServerError)
                                if tpl, ok := app.Templates["500.html"]; ok {
                                        tpl.ExecuteTemplate(w, "500.html", AppTemplateData(r, app))
                                } else {
                                        http.Error(w, "Internal Server Error", 500)
                                }
                        }
                }()

                rw := &responseWriter{ResponseWriter: w, statusCode: 200}
                next.ServeHTTP(rw, r)

                if rw.statusCode == http.StatusNotFound {
                        if tpl, ok := app.Templates["404.html"]; ok {
                                w.Header().Set("Content-Type", "text/html; charset=utf-8")
                                w.Header().Set("Cache-Control", "no-store")
                                tpl.ExecuteTemplate(w, "404.html", AppTemplateData(r, app))
                        } else {
                                http.NotFound(w, r)
                        }
                }
        })
}

/*
responseWriter records the status code so we can inspect it after the handler runs.
*/
type responseWriter struct {
        http.ResponseWriter
        statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
        rw.statusCode = code
        rw.ResponseWriter.WriteHeader(code)
}

