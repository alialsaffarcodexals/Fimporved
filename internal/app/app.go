package app

import (
	"database/sql"
	"html/template"
	"net/http"
	"time"
)

// App bundles together the shared dependencies used by HTTP handlers.
//
// Rather than passing the database, parsed templates and session
// configuration into every handler function individually, we wrap them
// inside a struct and provide methods on that struct. This makes
// testing easier and keeps handler signatures clean.
type App struct {
    // DB is the opened SQLite database. It is safe for concurrent use
    // by multiple goroutines.
    DB *sql.DB
    // Templates holds all parsed HTML templates keyed by filename.
    Templates map[string]*template.Template
    // CookieName is the name of the session cookie we set on login.
    CookieName string
    // SessionTTL controls how long a session remains valid before
    // expiring. A zero or negative duration effectively disables
    // sessions.
    SessionTTL time.Duration
}

// baseData returns the common template data used on every page.
// It includes whether the user is logged in, their ID and username
// and the list of available categories. Any errors retrieving the
// categories are ignored and result in an empty slice.
func (a *App) baseData(r *http.Request) map[string]any {
    uid, uname, logged := a.CurrentUser(r)
    cats, _ := a.AllCategories()
    return map[string]any{
        "LoggedIn":   logged,
        "UserID":     uid,
        "Username":   uname,
        "Categories": cats,
    }
}