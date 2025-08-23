package server

// This file provides a helper function for loading HTML templates from
// disk. Each page template is parsed alongside the shared layout so
// that all pages inherit the same header and footer. The function
// returns a map keyed by filename for convenient lookup in handlers.

import (
    "html/template"
    "path/filepath"
)

// LoadTemplates walks the provided directory and parses every HTML
// template found within it, excluding the layout itself. Each page
// template is parsed together with the layout so that the templates
// share a common base. The returned map is keyed by the basename of
// the template file (e.g. "index.html"). If any template fails to
// parse, an error is returned.
func LoadTemplates(dir string) (map[string]*template.Template, error) {
    layout := filepath.Join(dir, "layout.html")
    pages, err := filepath.Glob(filepath.Join(dir, "*.html"))
    if err != nil {
        return nil, err
    }
    m := make(map[string]*template.Template)
    for _, page := range pages {
        // Skip the layout itself. Each page will include it when parsing.
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