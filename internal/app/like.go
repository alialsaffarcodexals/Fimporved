package app

import "net/http"

/*
HandleLike records a like or dislike for either a post or a comment.
It upserts the user's preference and redirects back to the relevant
page once complete.
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
