package app

import (
	"net/http"
	"time"
)

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
