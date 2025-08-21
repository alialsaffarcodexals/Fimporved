package app

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

/*
HandleNewPost creates a fresh post for the logged in user and stores
associated categories. It expects title and body fields in the form
submission and optional comma separated category names.
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
