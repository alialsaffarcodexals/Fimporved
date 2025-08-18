package http

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	authpkg "forum/internal/auth"
)

type ctxKey int
const userKey ctxKey = 1

type Middleware struct {
	DB *sql.DB
	SessionTTL time.Duration
}

func (m *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func (m *Middleware) LoadUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err == nil {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()
			u, err := authpkg.GetUserBySession(ctx, m.DB, cookie.Value, time.Now())
			if err == nil && u != nil {
				r = r.WithContext(context.WithValue(r.Context(), userKey, u))
			} else {
				// Clear invalid/expired cookie
				c := &http.Cookie{Name: "session_id", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode}
				http.SetCookie(w, c)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func UserFromContext(ctx context.Context) *authpkg.User {
	v := ctx.Value(userKey)
	if v == nil { return nil }
	if u, ok := v.(*authpkg.User); ok { return u }
	return nil
}
