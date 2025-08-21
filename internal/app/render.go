package app

import "net/http"

/*
render writes common headers and executes the named template.
*/
func (a *App) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	/* Helpful in dev to avoid stale pages */
	w.Header().Set("Cache-Control", "no-store")
	tpl, ok := a.Templates[name]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := tpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
