package main

import (
	"html/template"
	"path/filepath"
)

/*
loadTemplates walks the directory and parses every template
alongside the shared layout. It returns a map keyed by
filename so handlers can pick the right template.
*/
func loadTemplates(dir string) (map[string]*template.Template, error) {
	layout := filepath.Join(dir, "layout.html")
	pages, err := filepath.Glob(filepath.Join(dir, "*.html"))
	if err != nil {
		return nil, err
	}
	m := make(map[string]*template.Template)
	for _, page := range pages {
		if filepath.Base(page) == "layout.html" {
			continue
		}
		t, err := template.ParseFiles(layout, page)
		if err != nil {
			return nil, err
		}
		m[filepath.Base(page)] = t
	}
	return m, nil
}
