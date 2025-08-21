package app

/*
AllCategories retrieves the list of category names ordered alphabetically.
*/
func (a *App) AllCategories() ([]string, error) {
	rows, err := a.DB.Query(`SELECT name FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		out = append(out, n)
	}
	return out, nil
}
