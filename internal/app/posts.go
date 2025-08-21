package app

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

/*
HandleNewPost creates a new post for the logged-in user.
*/
func (a *App) HandleNewPost(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cats, _ := a.AllCategories()
		a.render(w, "post_new.html", map[string]any{"View": "post_new", "Categories": cats})
	case http.MethodPost:
		uid, _, ok := a.CurrentUser(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		title := r.FormValue("title")
		body := r.FormValue("body")
		if title == "" || body == "" {
			http.Error(w, "title/body required", 400)
			return
		}
		res, err := a.DB.Exec(`INSERT INTO posts(user_id, title, body) VALUES(?,?,?)`, uid, title, body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		pid, _ := res.LastInsertId()
		if err := r.ParseForm(); err == nil {
			seen := map[string]bool{}
			for _, cname := range r.Form["category"] {
				for _, part := range strings.Split(cname, ",") {
					c := strings.TrimSpace(part)
					if c == "" || seen[c] {
						continue
					}
					seen[c] = true
					var cid int64
					err := a.DB.QueryRow(`SELECT id FROM categories WHERE name=?`, c).Scan(&cid)
					if errors.Is(err, sql.ErrNoRows) {
						res, err := a.DB.Exec(`INSERT INTO categories(name) VALUES(?)`, c)
						if err == nil {
							cid, _ = res.LastInsertId()
						}
					}
					if cid != 0 {
						a.DB.Exec(`INSERT OR IGNORE INTO post_categories(post_id, category_id) VALUES(?,?)`, pid, cid)
					}
				}
			}
		}
		http.Redirect(w, r, "/post?id="+strconv.FormatInt(pid, 10), http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

/*
HandleShowPost displays a single post and its comments.
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

/*
HandleNewComment adds a comment to an existing post.
*/
func (a *App) HandleNewComment(w http.ResponseWriter, r *http.Request) {
	uid, _, ok := a.CurrentUser(r)
	if !ok {
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
		http.Error(w, "empty comment", 400)
		return
	}
	if _, err := a.DB.Exec(`INSERT INTO comments(post_id, user_id, body) VALUES(?,?,?)`, postID, uid, body); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/post?id="+postID, http.StatusSeeOther)
}

/*
HandleLike records a like or dislike for a post or comment.
*/
func (a *App) HandleLike(w http.ResponseWriter, r *http.Request) {
	uid, _, ok := a.CurrentUser(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	typ := r.FormValue("type")
	tid := r.FormValue("id")
	val := r.FormValue("value")
	_, err := a.DB.Exec(`INSERT INTO likes(user_id, target_type, target_id, value) VALUES(?,?,?,?)
                         ON CONFLICT(user_id, target_type, target_id) DO UPDATE SET value=excluded.value`, uid, typ, tid, val)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if typ == "post" {
		http.Redirect(w, r, "/post?id="+tid, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
	}
}
