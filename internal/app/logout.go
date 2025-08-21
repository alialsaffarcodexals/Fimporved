package app

import (
	"net/http"
	"time"
)

/*
HandleLogout clears the user's session both server side and client side
by deleting the record and expiring the cookie immediately.
*/
func (a *App) HandleLogout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(a.CookieName)
	if err == nil {
		a.DB.Exec(`DELETE FROM sessions WHERE id=?`, c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: a.CookieName, Value: "", Path: "/", Expires: time.Unix(0, 0), MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
