package app

// This file defines the handler for user login. Users provide
// their email and password. If the credentials match an existing
// account a session is created and the user is redirected to the
// home page. On failure the form is re‑rendered with an error.

import (
    "database/sql"
    "net/http"

    "golang.org/x/crypto/bcrypt"
)

// HandleLogin renders the login form on GET and processes
// authentication on POST. It expects `email` and `password` form
// fields. On successful login a new session cookie is set. On
// failure an error message is displayed.
func (a *App) HandleLogin(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        data := a.baseData(r)
        if msg := r.URL.Query().Get("error"); msg != "" {
            data["Error"] = msg
        }
        tmpl := a.Templates["login.html"]
        tmpl.ExecuteTemplate(w, "login.html", data)
    case http.MethodPost:
        if err := r.ParseForm(); err != nil {
            http.Error(w, "unable to parse form", http.StatusBadRequest)
            return
        }
        email := r.Form.Get("email")
        password := r.Form.Get("password")
        if email == "" || password == "" {
            http.Redirect(w, r, "/login?error=Email and password are required", http.StatusSeeOther)
            return
        }
        // Look up the user by email. If not found return an error.
        var id int64
        var username, hash string
        err := a.DB.QueryRow(`SELECT id, username, password_hash FROM users WHERE email = ?`, email).Scan(&id, &username, &hash)
        if err == sql.ErrNoRows {
            http.Redirect(w, r, "/login?error=Invalid credentials", http.StatusSeeOther)
            return
        }
        if err != nil {
            http.Error(w, "database error", http.StatusInternalServerError)
            return
        }
        // Compare the provided password with the stored hash.
        if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
            http.Redirect(w, r, "/login?error=Invalid credentials", http.StatusSeeOther)
            return
        }
        // Credentials valid; create a session.
        if err := a.SetSession(w, id); err != nil {
            http.Error(w, "failed to create session", http.StatusInternalServerError)
            return
        }
        http.Redirect(w, r, "/", http.StatusSeeOther)
    default:
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
    }
}