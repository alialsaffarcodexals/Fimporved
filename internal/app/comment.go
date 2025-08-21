package app

import "net/http"

/*
HandleNewComment inserts a comment for a post. Only authenticated users
may comment, and the handler expects a post_id and body in the form
payload.
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
