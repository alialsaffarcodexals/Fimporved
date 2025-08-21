package app

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

/*
HandleLogin verifies user credentials and starts a new session when the
password matches. It also clears any old sessions for that user.
*/
func (a *App) HandleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.render(w, "login.html", map[string]any{"View": "login"})
		return

	case http.MethodPost:
		email := r.FormValue("email")
		pass := r.FormValue("password")

		var (
			id       int64
			hash     string
			username string
		)
		err := a.DB.QueryRow(`SELECT id, username, password_hash FROM users WHERE email=?`, email).
			Scan(&id, &username, &hash)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass)) != nil {
			w.WriteHeader(http.StatusUnauthorized)
			a.render(w, "login.html", map[string]any{
				"View":  "login",
				"Error": "invalid credentials",
			})
			return
		}

		tx, err := a.DB.Begin()
		if err != nil {
			http.Error(w, "failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer func() { _ = tx.Rollback() }()

		if _, err = tx.Exec(`DELETE FROM sessions WHERE user_id=?`, id); err != nil {
			http.Error(w, "failed to clear old sessions", http.StatusInternalServerError)
			return
		}

		sid := uuid.NewString()
		expiresAt := time.Now().Add(a.SessionTTL)

		if _, err = tx.Exec(`INSERT INTO sessions(id, user_id, expires_at) VALUES(?,?,?)`, sid, id, expiresAt); err != nil {
			http.Error(w, "failed to create session", http.StatusInternalServerError)
			return
		}

		if err = tx.Commit(); err != nil {
			http.Error(w, "failed to finalize login", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     a.CookieName,
			Value:    sid,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  expiresAt,
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
}
