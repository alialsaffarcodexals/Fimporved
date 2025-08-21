package app

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

/*
HandleRegister shows the registration form or creates a new user when data is posted.
*/
func (a *App) HandleRegister(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.render(w, "register.html", map[string]any{"View": "register"})
	case http.MethodPost:
		email := r.FormValue("email")
		username := r.FormValue("username")
		pass := r.FormValue("password")
		if email == "" || username == "" || pass == "" {
			http.Error(w, "missing fields", http.StatusBadRequest)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = a.DB.Exec(`INSERT INTO users(email, username, password_hash) VALUES(?,?,?)`, email, username, string(hash))
		if err != nil {
			if isUniqueErr(err) {
				w.WriteHeader(http.StatusUnauthorized)
				a.render(w, "register.html", map[string]any{
					"View":  "register",
					"Error": "email or username already taken",
				})
				return
			}
			http.Error(w, err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

/*
isUniqueErr looks for the SQLite unique constraint error message.
*/
func isUniqueErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

/*
HandleLogin verifies credentials and starts a new session for the user.
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

/*
HandleLogout clears the session cookie and removes it from the database.
*/
func (a *App) HandleLogout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(a.CookieName)
	if err == nil {
		a.DB.Exec(`DELETE FROM sessions WHERE id=?`, c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: a.CookieName, Value: "", Path: "/", Expires: time.Unix(0, 0), MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
