package app

import "net/http"

// HandleNewComment processes a form submission to create a new comment.
//
// Only authenticated users may comment on posts. The form must include
// both a `post_id` identifying the parent post and a `body` with the
// comment text. Comments with an empty body are rejected with a
// Bad Request error. After inserting the comment into the database the
// user is redirected back to the post page.
func (a *App) HandleNewComment(w http.ResponseWriter, r *http.Request) {
    uid, _, ok := a.CurrentUser(r)
    if !ok {
        // If the user is not logged in, send them to the login page.
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    postID := r.FormValue("post_id")
    body := r.FormValue("body")
    if body == "" {
        http.Error(w, "empty comment", http.StatusBadRequest)
        return
    }
    if _, err := a.DB.Exec(`INSERT INTO comments(post_id, user_id, body) VALUES(?,?,?)`, postID, uid, body); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    http.Redirect(w, r, "/post?id="+postID, http.StatusSeeOther)
}