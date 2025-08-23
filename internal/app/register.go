package app

// This file defines the handler for user registration. Users must
// provide a unique email address, a username and a password. The
// password is hashed using bcrypt before storing it in the database.
// If the email or username is already taken the registration fails
// with an appropriate error message.

import (
    "net/http"

    "golang.org/x/crypto/bcrypt"
)

// HandleRegister renders the registration form on GET and processes
// new user registrations on POST. It expects the form fields
// `email`, `username` and `password`. On successful registration the
// user is redirected to the login page. On error the form is
// re‑rendered with an error message.
func (a *App) HandleRegister(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        // Render the registration form. Build the base template
        // context and include any error message provided in the
        // query string.
        data := a.baseData(r)
        if msg := r.URL.Query().Get("error"); msg != "" {
            data["Error"] = msg
        }
        tmpl := a.Templates["register.html"]
        tmpl.ExecuteTemplate(w, "register.html", data)
    case http.MethodPost:
        // Parse the form values.
        if err := r.ParseForm(); err != nil {
            http.Error(w, "unable to parse form", http.StatusBadRequest)
            return
        }
        email := r.Form.Get("email")
        username := r.Form.Get("username")
        password := r.Form.Get("password")
        if email == "" || username == "" || password == "" {
            http.Redirect(w, r, "/register?error=All fields are required", http.StatusSeeOther)
            return
        }
        // Hash the password. bcrypt.GenerateFromPassword returns an
        // error only if the cost parameter is invalid.
        hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
        if err != nil {
            http.Error(w, "password hashing failed", http.StatusInternalServerError)
            return
        }
        // Insert the user into the database. Use parameterized
        // statements to avoid injection. The UNIQUE constraints on
        // email and username will cause the Exec call to fail if
        // duplicates exist.
        _, err = a.DB.Exec(`INSERT INTO users(email, username, password_hash) VALUES(?,?,?)`, email, username, string(hash))
        if err != nil {
            // Determine if the error is due to uniqueness. SQLite
            // returns an error string containing "UNIQUE" for such
            // violations. We can check for this substring.
            msg := "Registration failed"
            if err.Error() != "" {
                if contains(err.Error(), "UNIQUE") {
                    msg = "Email or username already exists"
                }
            }
            http.Redirect(w, r, "/register?error="+msg, http.StatusSeeOther)
            return
        }
        // Registration successful; redirect to login page.
        http.Redirect(w, r, "/login", http.StatusSeeOther)
    default:
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
    }
}

// contains is a small helper to avoid importing strings package for a
// single Contains call. It returns true if substr appears in s.
func contains(s, substr string) bool {
    return len(substr) == 0 || (len(s) >= len(substr) && (index(s, substr) >= 0))
}

// index returns the index of substr in s or -1 if not found. It is
// implemented manually to avoid bringing in the full strings package.
func index(s, substr string) int {
    for i := 0; i+len(substr) <= len(s); i++ {
        if s[i:i+len(substr)] == substr {
            return i
        }
    }
    return -1
}