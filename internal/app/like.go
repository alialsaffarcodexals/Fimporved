package app

// This file defines the handler for liking and disliking posts or
// comments. The handler expects query parameters specifying the
// target type (either `post` or `comment`), the target ID and the
// value (1 for like, -1 for dislike). Submitting the same value
// twice removes the existing reaction, effectively toggling it off.

import (
    sql "database/sql"
    "net/http"
    "strconv"
)

// HandleLike processes like/dislike actions. It requires the user
// to be authenticated (enforced by the RequireAuth middleware). On
// completion it redirects back to the post page containing the
// target so that the user sees the updated reaction counts.
func (a *App) HandleLike(w http.ResponseWriter, r *http.Request) {
    // Only allow POST or GET for likes. Using GET simplifies
    // template links but in a real application you may prefer POST
    // for CSRF protection.
    targetType := r.URL.Query().Get("type")
    if targetType != "post" && targetType != "comment" {
        http.Error(w, "invalid target type", http.StatusBadRequest)
        return
    }
    idStr := r.URL.Query().Get("id")
    targetID, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil || targetID <= 0 {
        http.Error(w, "invalid target id", http.StatusBadRequest)
        return
    }
    valStr := r.URL.Query().Get("value")
    v, err := strconv.Atoi(valStr)
    if err != nil || (v != 1 && v != -1) {
        http.Error(w, "invalid reaction value", http.StatusBadRequest)
        return
    }
    // Determine the user from the session.
    uid, _, ok := a.CurrentUser(r)
    if !ok {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }
    // Check for existing like on this target by this user.
    var existingID int64
    var existingValue int
    row := a.DB.QueryRow(`SELECT id, value FROM likes WHERE user_id=? AND target_type=? AND target_id=?`, uid, targetType, targetID)
    switch err := row.Scan(&existingID, &existingValue); err {
    case nil:
        // Already exists. If the same value is being sent, remove the
        // like (toggle off). Otherwise update the value.
        if existingValue == v {
            _, _ = a.DB.Exec(`DELETE FROM likes WHERE id = ?`, existingID)
        } else {
            _, _ = a.DB.Exec(`UPDATE likes SET value = ? WHERE id = ?`, v, existingID)
        }
    case sql.ErrNoRows:
        // No existing record; insert a new like.
        _, _ = a.DB.Exec(`INSERT INTO likes(user_id, target_type, target_id, value) VALUES(?,?,?,?)`, uid, targetType, targetID, v)
    default:
        http.Error(w, "database error", http.StatusInternalServerError)
        return
    }
    // Determine the post ID to redirect to. When liking a post it's
    // the target ID itself. When liking a comment we need to look up
    // the parent post. Accept an optional "post_id" parameter to
    // avoid a lookup. If absent we query the comments table.
    var postID int64
    if targetType == "post" {
        postID = targetID
    } else {
        if pidStr := r.URL.Query().Get("post_id"); pidStr != "" {
            postID, _ = strconv.ParseInt(pidStr, 10, 64)
        } else {
            // Query comments table for the parent post ID.
            row := a.DB.QueryRow(`SELECT post_id FROM comments WHERE id = ?`, targetID)
            if err := row.Scan(&postID); err != nil {
                http.Error(w, "comment not found", http.StatusBadRequest)
                return
            }
        }
    }
    http.Redirect(w, r, "/post?id="+strconv.FormatInt(postID, 10), http.StatusSeeOther)
}