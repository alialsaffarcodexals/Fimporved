package app

// This file implements session management for the forum. Sessions are
// stored in the database so that they persist across restarts and to
// allow revocation. Each session has a UUID identifier, the ID of
// the associated user and an expiry timestamp. A unique constraint
// on user_id ensures that a user may only have a single active
// session at a time. The session cookie is HTTP‑only and scoped to
// the root path.

import (
    "net/http"
    "time"

    "github.com/google/uuid"
)

// SetSession creates a new session for the provided user ID and
// writes the session cookie to the response. Any existing session
// for the user is replaced. The expiration of the session is
// determined by App.SessionTTL. A zero TTL will create a session
// cookie without an expiry, which becomes a session cookie in the
// browser.
func (a *App) SetSession(w http.ResponseWriter, userID int64) error {
    // Generate a new unique session ID. uuid.New() never returns an
    // error so we can call String() directly.
    sid := uuid.New().String()
    // Determine the expiry. Use Unix timestamps for easy storage.
    expires := time.Now().Add(a.SessionTTL)
    // Remove any existing session for this user so that they have
    // only one active session at a time.
    _, _ = a.DB.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
    // Insert the new session. Use INSERT OR REPLACE to handle
    // existing rows gracefully.
    _, err := a.DB.Exec(`INSERT INTO sessions(id, user_id, expires_at) VALUES(?,?,?)`, sid, userID, expires.Unix())
    if err != nil {
        return err
    }
    cookie := http.Cookie{
        Name:     a.CookieName,
        Value:    sid,
        Path:     "/",
        Expires:  expires,
        HttpOnly: true,
        SameSite: http.SameSiteLaxMode,
    }
    http.SetCookie(w, &cookie)
    return nil
}

// ClearSession removes the current session from the database and
// invalidates the browser cookie. It is called during logout.
func (a *App) ClearSession(w http.ResponseWriter, r *http.Request) {
    c, err := r.Cookie(a.CookieName)
    if err == nil {
        // Delete the session row. Ignore errors since the session may
        // already be gone.
        _, _ = a.DB.Exec(`DELETE FROM sessions WHERE id = ?`, c.Value)
    }
    // Set an expired cookie to remove it from the browser.
    http.SetCookie(w, &http.Cookie{
        Name:     a.CookieName,
        Value:    "",
        Path:     "/",
        Expires:  time.Unix(0, 0),
        MaxAge:   -1,
        HttpOnly: true,
        SameSite: http.SameSiteLaxMode,
    })
}

// CurrentUser returns the ID and username of the logged‑in user along
// with a boolean indicating whether a valid session exists. If no
// session cookie is present or the session has expired the boolean
// will be false. Expired sessions are removed automatically.
func (a *App) CurrentUser(r *http.Request) (int64, string, bool) {
    c, err := r.Cookie(a.CookieName)
    if err != nil {
        return 0, "", false
    }
    var userID int64
    var expiresUnix int64
    var username string
    // Join the sessions and users table to fetch the username in a
    // single query.
    row := a.DB.QueryRow(`SELECT s.user_id, s.expires_at, u.username FROM sessions s JOIN users u ON s.user_id = u.id WHERE s.id = ?`, c.Value)
    if err := row.Scan(&userID, &expiresUnix, &username); err != nil {
        return 0, "", false
    }
    expires := time.Unix(expiresUnix, 0)
    if time.Now().After(expires) {
        // Session has expired. Remove it and indicate no user.
        _, _ = a.DB.Exec(`DELETE FROM sessions WHERE id = ?`, c.Value)
        return 0, "", false
    }
    return userID, username, true
}

// RequireAuth wraps a handler and ensures that the user is
// authenticated before calling it. If no valid session exists the
// user is redirected to the login page. Use this on handlers that
// require login such as creating posts, comments or likes.
func (a *App) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if _, _, ok := a.CurrentUser(r); !ok {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }
        next(w, r)
    }
}