package server

// This file defines a helper for constructing the common data passed to
// all HTML templates. It extracts the current logged‑in user from the
// request via the App and fetches the list of categories from the
// database. Individual handlers can embed additional fields into
// the returned map before executing a template.

import (
    "net/http"

    "forum/internal/app"
)

// AppTemplateData builds the common template data including the
// logged‑in user and category list. It is meant to be called at the
// beginning of each handler to ensure consistent data across pages.
// Errors fetching the categories are ignored; in that case the
// Categories field will be nil. Handlers can override these values
// by writing into the returned map before passing it to ExecuteTemplate.
func AppTemplateData(r *http.Request, app *app.App) map[string]any {
    uid, uname, logged := app.CurrentUser(r)
    cats, _ := app.AllCategories()
    return map[string]any{
        "LoggedIn":   logged,
        "UserID":     uid,
        "Username":   uname,
        "Categories": cats,
    }
}