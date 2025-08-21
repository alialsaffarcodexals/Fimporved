package app

import (
	"database/sql"
	"html/template"
	"net/http"
	"time"
)

// App holds core services shared across handlers.
type App struct {
	DB         *sql.DB
	Templates  map[string]*template.Template
	CookieName string
	SessionTTL time.Duration
}

/*
CurrentUser checks the session cookie and returns the user id and
username when a valid session is found.
*/
func (a *App) CurrentUser(r *http.Request) (id int64, username string, ok bool) {
	c, err := r.Cookie(a.CookieName)
	if err != nil {
		return 0, "", false
	}
	var userID int64
	var expires time.Time
	err = a.DB.QueryRow(`SELECT user_id, expires_at FROM sessions WHERE id=?`, c.Value).Scan(&userID, &expires)
	if err != nil {
		return 0, "", false
	}
	if time.Now().After(expires) {
		a.DB.Exec(`DELETE FROM sessions WHERE id=?`, c.Value)
		return 0, "", false
	}
	err = a.DB.QueryRow(`SELECT username FROM users WHERE id=?`, userID).Scan(&username)
	if err != nil {
		return 0, "", false
	}
	return userID, username, true
}

/*
RequireAuth ensures the user is logged in before calling the next handler.
*/
func (a *App) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := a.CurrentUser(r); !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	}
}

/*
render writes common headers and executes the named template.
*/
func (a *App) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	/* Helpful in dev to avoid stale pages */
	w.Header().Set("Cache-Control", "no-store")
	tpl, ok := a.Templates[name]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := tpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
