package app

import (
	"net/http"
	"time"
)

/*
HandleIndex renders the home page with recent posts and filter controls.
*/
func (a *App) HandleIndex(w http.ResponseWriter, r *http.Request) {
	q := `SELECT p.id, p.title, p.body, p.created_at, u.username,
                 (SELECT COUNT(*) FROM likes l WHERE l.target_type='post' AND l.target_id=p.id AND l.value=1) AS likes,
                 (SELECT COUNT(*) FROM likes l WHERE l.target_type='post' AND l.target_id=p.id AND l.value=-1) AS dislikes
          FROM posts p
          JOIN users u ON u.id = p.user_id
          WHERE 1=1`
	args := []any{}

	if cat := r.URL.Query().Get("category"); cat != "" {
		q += ` AND EXISTS (SELECT 1 FROM post_categories pc JOIN categories c ON c.id=pc.category_id WHERE pc.post_id=p.id AND c.name=?)`
		args = append(args, cat)
	}

	if r.URL.Query().Get("mine") == "1" {
		if uid, _, ok := a.CurrentUser(r); ok {
			q += ` AND p.user_id = ?`
			args = append(args, uid)
		} else {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
	}

	if r.URL.Query().Get("liked") == "1" {
		if uid, _, ok := a.CurrentUser(r); ok {
			q += ` AND p.id IN (SELECT target_id FROM likes WHERE target_type='post' AND user_id=? AND value=1)`
			args = append(args, uid)
		} else {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
	}

	q += ` ORDER BY p.created_at DESC LIMIT 50`

	rows, err := a.DB.Query(q, args...)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type Row struct {
		ID                    int64
		Title, Body, Username string
		CreatedAt             string
		Likes, Dislikes       int
	}
	var posts []Row
	for rows.Next() {
		var rrow Row
		var ts time.Time
		if err := rows.Scan(&rrow.ID, &rrow.Title, &rrow.Body, &ts, &rrow.Username, &rrow.Likes, &rrow.Dislikes); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		rrow.CreatedAt = ts.Format("2006-01-02 15:04")
		posts = append(posts, rrow)
	}

	cats, _ := a.AllCategories()

	uid, uname, logged := a.CurrentUser(r)
	a.render(w, "index.html", map[string]any{
		"View":       "index",
		"Posts":      posts,
		"Categories": cats,
		"LoggedIn":   logged,
		"UserID":     uid,
		"Username":   uname,
		"Filter": map[string]string{
			"category": r.URL.Query().Get("category"),
			"mine":     r.URL.Query().Get("mine"),
			"liked":    r.URL.Query().Get("liked"),
		},
	})
}
