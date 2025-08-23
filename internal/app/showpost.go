package app

// This file defines the handler for displaying a single post and its
// associated comments. It gathers all necessary data such as the
// author, categories, like/dislike counts and the current user's
// reaction. Comments are ordered by creation time.

import (
    "database/sql"
    "net/http"
    "strconv"
    "time"
)

// commentView holds the data needed to render a comment.
type commentView struct {
    ID           int64
    Body         string
    Author       string
    CreatedAt    time.Time
    LikeCount    int
    DislikeCount int
    MyReaction   int
}

// postView holds the data needed to render a post along with its
// comments.
type postView struct {
    ID           int64
    Title        string
    Body         string
    Author       string
    Categories   string
    CreatedAt    time.Time
    LikeCount    int
    DislikeCount int
    MyReaction   int
    Comments     []commentView
}

// HandleShowPost renders a single post page. If the post ID is
// missing or invalid a 404 page is shown. The page includes the
// post itself and its comments.
func (a *App) HandleShowPost(w http.ResponseWriter, r *http.Request) {
    idStr := r.URL.Query().Get("id")
    pid, err := strconv.ParseInt(idStr, 10, 64)
    if err != nil || pid <= 0 {
        http.NotFound(w, r)
        return
    }
    // Determine current user ID for personalised data.
    uid, _, _ := a.CurrentUser(r)
    // Query the post and its metadata.
    var p postView
    var cats sql.NullString
    var myReact sql.NullInt64
    row := a.DB.QueryRow(`SELECT
        p.id, p.title, p.body, p.created_at,
        u.username,
        GROUP_CONCAT(c.name),
        (SELECT COUNT(*) FROM likes WHERE target_type='post' AND target_id=p.id AND value=1) as like_count,
        (SELECT COUNT(*) FROM likes WHERE target_type='post' AND target_id=p.id AND value=-1) as dislike_count,
        COALESCE((SELECT value FROM likes WHERE target_type='post' AND target_id=p.id AND user_id=?), 0)
    FROM posts p
    JOIN users u ON p.user_id = u.id
    LEFT JOIN post_categories pc ON p.id = pc.post_id
    LEFT JOIN categories c ON pc.category_id = c.id
    WHERE p.id = ?
    GROUP BY p.id`, uid, pid)
    if err := row.Scan(&p.ID, &p.Title, &p.Body, &p.CreatedAt, &p.Author, &cats, &p.LikeCount, &p.DislikeCount, &myReact); err != nil {
        if err == sql.ErrNoRows {
            http.NotFound(w, r)
            return
        }
        http.Error(w, "database error", http.StatusInternalServerError)
        return
    }
    if cats.Valid {
        p.Categories = cats.String
    }
    if myReact.Valid {
        p.MyReaction = int(myReact.Int64)
    }
    // Query comments for this post.
    rows, err := a.DB.Query(`SELECT
        cm.id, cm.body, cm.created_at, u.username,
        (SELECT COUNT(*) FROM likes WHERE target_type='comment' AND target_id=cm.id AND value=1) as like_count,
        (SELECT COUNT(*) FROM likes WHERE target_type='comment' AND target_id=cm.id AND value=-1) as dislike_count,
        COALESCE((SELECT value FROM likes WHERE target_type='comment' AND target_id=cm.id AND user_id=?), 0)
    FROM comments cm
    JOIN users u ON cm.user_id = u.id
    WHERE cm.post_id = ?
    ORDER BY cm.created_at ASC`, uid, pid)
    if err != nil {
        http.Error(w, "database error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    for rows.Next() {
        var cmt commentView
        var mycReact sql.NullInt64
        if err := rows.Scan(&cmt.ID, &cmt.Body, &cmt.CreatedAt, &cmt.Author, &cmt.LikeCount, &cmt.DislikeCount, &mycReact); err != nil {
            http.Error(w, "database error", http.StatusInternalServerError)
            return
        }
        if mycReact.Valid {
            cmt.MyReaction = int(mycReact.Int64)
        }
        p.Comments = append(p.Comments, cmt)
    }
    data := a.baseData(r)
    data["Post"] = p
    tmpl := a.Templates["post_show.html"]
    tmpl.ExecuteTemplate(w, "post_show.html", data)
}