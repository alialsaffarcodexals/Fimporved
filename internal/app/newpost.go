package app

// This file defines the handler for creating a new post. Only
// authenticated users may access this handler (enforced by the
// RequireAuth middleware in main.go). Posts consist of a title,
// body and one or more categories.

import (
    "net/http"
    "strings"
    "strconv"
)

// HandleNewPost displays the new post form on GET and inserts a
// new post on POST. It expects the form fields `title`, `body`
// and `categories` (multi‑select). At least one category must be
// chosen. The handler redirects to the newly created post on
// success.
func (a *App) HandleNewPost(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        data := a.baseData(r)
        tmpl := a.Templates["post_new.html"]
        tmpl.ExecuteTemplate(w, "post_new.html", data)
    case http.MethodPost:
        uid, _, ok := a.CurrentUser(r)
        if !ok {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }
        if err := r.ParseForm(); err != nil {
            http.Error(w, "unable to parse form", http.StatusBadRequest)
            return
        }
        title := strings.TrimSpace(r.Form.Get("title"))
        body := strings.TrimSpace(r.Form.Get("body"))
        cats := r.Form["categories"]
        if title == "" || body == "" || len(cats) == 0 {
            http.Error(w, "all fields are required", http.StatusBadRequest)
            return
        }
        // Insert the post and get its ID.
        res, err := a.DB.Exec(`INSERT INTO posts(user_id, title, body) VALUES(?,?,?)`, uid, title, body)
        if err != nil {
            http.Error(w, "database error", http.StatusInternalServerError)
            return
        }
        pid, err := res.LastInsertId()
        if err != nil {
            http.Error(w, "failed to retrieve post id", http.StatusInternalServerError)
            return
        }
        // Associate the post with categories. We lookup the ID for
        // each category name to avoid inserting invalid names.
        for _, name := range cats {
            var cid int64
            err := a.DB.QueryRow(`SELECT id FROM categories WHERE name = ?`, name).Scan(&cid)
            if err != nil {
                // Ignore invalid categories.
                continue
            }
            _, _ = a.DB.Exec(`INSERT OR IGNORE INTO post_categories(post_id, category_id) VALUES(?,?)`, pid, cid)
        }
        http.Redirect(w, r, "/post?id="+strconv.FormatInt(pid, 10), http.StatusSeeOther)
    default:
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
    }
}