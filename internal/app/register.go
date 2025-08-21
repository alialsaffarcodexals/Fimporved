package app

import (
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

/*
HandleRegister shows the registration form or creates a new account when
valid data is posted. Duplicate usernames or emails result in a friendly
error message.
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
