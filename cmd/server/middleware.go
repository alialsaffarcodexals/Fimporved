package main

import (
	"fmt"
	"net/http"
	"time"

	"forum-mvp/internal/app"
)

/*
logRequest prints the method, path and how long the handler took.
It's handy while developing to see what's going on.
*/
func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start))
	})
}

/*
withCustomErrors wraps the default mux and provides friendly 404
and 500 pages when things go wrong.
*/
func withCustomErrors(next *http.ServeMux, app *app.App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-store")
				w.WriteHeader(http.StatusInternalServerError)
				if tpl, ok := app.Templates["500.html"]; ok {
					tpl.ExecuteTemplate(w, "500.html", appTemplateData(r, app))
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
				tpl.ExecuteTemplate(w, "404.html", appTemplateData(r, app))
			} else {
				http.NotFound(w, r)
			}
		}
	})
}

/*
responseWriter records the status code so we can check it
after the handler runs.
*/
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

/*
appTemplateData builds the common template data including
the logged-in user and category list.
*/
func appTemplateData(r *http.Request, app *app.App) map[string]any {
	uid, uname, logged := app.CurrentUser(r)
	cats, _ := app.AllCategories()
	return map[string]any{
		"LoggedIn":   logged,
		"UserID":     uid,
		"Username":   uname,
		"Categories": cats,
	}
}
