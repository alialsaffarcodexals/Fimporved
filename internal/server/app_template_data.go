package server

import (
        "net/http"

        "forum-mvp/internal/app"
)

/*
AppTemplateData builds the common template data including the logged-in
user and category list.

Example:

    data := server.AppTemplateData(r, app)
*/
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

