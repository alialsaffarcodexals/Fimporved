package app

import (
	"net/http"
	"time"
)

/*
HandleShowPost renders a single post along with its categories and
comments. It also passes user session details to the template so the
page can tailor controls for logged in visitors.
*/
func (a *App) HandleShowPost(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	type Post struct {
		ID                    int64
		Title, Body, Username string
		CreatedAt             string
		Likes, Dislikes       int
		Categories            []string
	}
	var p Post
	var ts time.Time
	err := a.DB.QueryRow(`SELECT p.id, p.title, p.body, p.created_at, u.username,
        (SELECT COUNT(*) FROM likes l WHERE l.target_type='post' AND l.target_id=p.id AND l.value=1) AS likes,
        (SELECT COUNT(*) FROM likes l WHERE l.target_type='post' AND l.target_id=p.id AND l.value=-1) AS dislikes
        FROM posts p
        JOIN users u ON u.id=p.user_id
        WHERE p.id=?`, id).Scan(&p.ID, &p.Title, &p.Body, &ts, &p.Username, &p.Likes, &p.Dislikes)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	p.CreatedAt = ts.Format("2006-01-02 15:04")
	rows, _ := a.DB.Query(`SELECT c.name FROM categories c JOIN post_categories pc ON pc.category_id=c.id WHERE pc.post_id=? ORDER BY c.name`, id)
	defer rows.Close()
	for rows.Next() {
		var n string
		rows.Scan(&n)
		p.Categories = append(p.Categories, n)
	}

	crows, err := a.DB.Query(`SELECT c.id, c.body, c.created_at, u.username,
        (SELECT COUNT(*) FROM likes l WHERE l.target_type='comment' AND l.target_id=c.id AND l.value=1) AS likes,
        (SELECT COUNT(*) FROM likes l WHERE l.target_type='comment' AND l.target_id=c.id AND l.value=-1) AS dislikes
        FROM comments c
        JOIN users u ON u.id=c.user_id
        WHERE c.post_id=?
        ORDER BY c.created_at ASC`, id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	type Comment struct {
		ID                        int64
		Body, Username, CreatedAt string
		Likes, Dislikes           int
	}
	var comments []Comment
	for crows.Next() {
		var cmt Comment
		var t time.Time
		if err := crows.Scan(&cmt.ID, &cmt.Body, &t, &cmt.Username, &cmt.Likes, &cmt.Dislikes); err == nil {
			cmt.CreatedAt = t.Format("2006-01-02 15:04")
			comments = append(comments, cmt)
		}
	}
	uid, uname, logged := a.CurrentUser(r)
	a.render(w, "post_show.html", map[string]any{
		"View":     "post_show",
		"Post":     p,
		"Comments": comments,
		"LoggedIn": logged,
		"UserID":   uid,
		"Username": uname,
	})
}
