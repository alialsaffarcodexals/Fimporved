package app

// This file implements the handler for the forum home page. The
// index lists all posts ordered by creation time and provides
// optional filtering by category, posts authored by the current
// user and posts liked by the current user. Anonymous visitors can
// see all posts but cannot access the "mine" or "liked" filters.

import (
    "database/sql"
    "net/http"
    "strings"
    "time"
)

// postListing holds the data necessary to render a post in the
// index. Categories is a comma‑separated string rather than a
// slice to simplify template rendering without range loops.
type postListing struct {
    ID           int64
    Title        string
    Body         string
    Author       string
    Categories   string
    LikeCount    int
    DislikeCount int
    MyReaction   int
    CreatedAt    time.Time
}

// HandleIndex renders the list of posts with optional filters. The
// query parameters recognised are:
//   category=<name>  – only posts containing this category
//   filter=mine      – only posts authored by the logged‑in user
//   filter=liked     – only posts liked by the logged‑in user
// Filters "mine" and "liked" are ignored when the user is not
// authenticated.
func (a *App) HandleIndex(w http.ResponseWriter, r *http.Request) {
    // Read filter parameters from the query string.
    category := r.URL.Query().Get("category")
    filter := r.URL.Query().Get("filter")
    // Determine current user. uid is zero when anonymous.
    uid, _, logged := a.CurrentUser(r)
    // Build the SQL query incrementally. We'll assemble WHERE
    // clauses and arguments based on the requested filters.
    var where []string
    var args []any
    // When a category is specified we join through post_categories
    // below. We'll add a WHERE clause after the join.
    if category != "" {
        where = append(where, "c.name = ?")
        args = append(args, category)
    }
    // If the user requests the "mine" filter and is logged in we
    // restrict posts to those authored by them.
    if filter == "mine" && logged {
        where = append(where, "p.user_id = ?")
        args = append(args, uid)
    }
    // If the user requests the "liked" filter and is logged in we
    // join the likes table to find posts the user has liked. We
    // implement this as an inner join on likes with value=1.
    likedJoin := ""
    if filter == "liked" && logged {
        likedJoin = "JOIN likes l2 ON l2.target_type='post' AND l2.target_id=p.id AND l2.user_id=? AND l2.value=1"
        args = append(args, uid)
    }
    // Build the base query. We join users and categories via
    // post_categories. We use GROUP_CONCAT to aggregate category names
    // into a single string.
    query := `SELECT
        p.id, p.title, p.body, p.created_at,
        u.username,
        GROUP_CONCAT(DISTINCT c.name) as categories,
        (SELECT COUNT(*) FROM likes WHERE target_type='post' AND target_id=p.id AND value=1) as like_count,
        (SELECT COUNT(*) FROM likes WHERE target_type='post' AND target_id=p.id AND value=-1) as dislike_count,
        COALESCE((SELECT value FROM likes WHERE target_type='post' AND target_id=p.id AND user_id=?), 0) as my_reaction
    FROM posts p
    JOIN users u ON p.user_id = u.id
    LEFT JOIN post_categories pc ON p.id = pc.post_id
    LEFT JOIN categories c ON pc.category_id = c.id `
    // Insert the liked join if necessary.
    if likedJoin != "" {
        query += likedJoin + " "
    }
    // Add WHERE clauses if any.
    if len(where) > 0 {
        query += "WHERE " + strings.Join(where, " AND ") + " "
    }
    query += "GROUP BY p.id ORDER BY p.created_at DESC"
    // The first argument for my_reaction is the user ID. When not logged
    // in we pass zero which yields no reaction.
    args = append([]any{uid}, args...)
    rows, err := a.DB.Query(query, args...)
    if err != nil {
        http.Error(w, "database error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    var posts []postListing
    for rows.Next() {
        var p postListing
        var cats sql.NullString
        var myReact sql.NullInt64
        if err := rows.Scan(&p.ID, &p.Title, &p.Body, &p.CreatedAt, &p.Author, &cats, &p.LikeCount, &p.DislikeCount, &myReact); err != nil {
            http.Error(w, "database error", http.StatusInternalServerError)
            return
        }
        if cats.Valid {
            p.Categories = cats.String
        }
        if myReact.Valid {
            p.MyReaction = int(myReact.Int64)
        }
        // Truncate the body for preview. If the body is longer than
        // 200 runes slice it and append ellipsis.
        const maxPreview = 200
        if len([]rune(p.Body)) > maxPreview {
            runes := []rune(p.Body)
            p.Body = string(runes[:maxPreview]) + "..."
        }
        posts = append(posts, p)
    }
    data := a.baseData(r)
    data["Posts"] = posts
    data["SelectedCategory"] = category
    data["SelectedFilter"] = filter
    tmpl := a.Templates["index.html"]
    tmpl.ExecuteTemplate(w, "index.html", data)
}