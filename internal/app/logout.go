package app

// This file implements user logout. Logging out removes the current
// session from the database and clears the cookie so that further
// requests are treated as anonymous.

import "net/http"

// HandleLogout terminates the user session and redirects to the
// home page. It accepts both GET and POST for convenience.
func (a *App) HandleLogout(w http.ResponseWriter, r *http.Request) {
    // We allow both GET and POST to log out. In practice you may
    // prefer POST to avoid accidental logouts triggered by crawlers.
    a.ClearSession(w, r)
    http.Redirect(w, r, "/", http.StatusSeeOther)
}