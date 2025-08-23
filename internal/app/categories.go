package app

// AllCategories returns the names of all categories sorted
// alphabetically. Categories are stored in their own table and can be
// attached to posts. If the query fails the error is returned to
// the caller.
func (a *App) AllCategories() ([]string, error) {
    rows, err := a.DB.Query(`SELECT name FROM categories ORDER BY name`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var out []string
    for rows.Next() {
        var name string
        if err := rows.Scan(&name); err != nil {
            return nil, err
        }
        out = append(out, name)
    }
    return out, nil
}