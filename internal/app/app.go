package app

import (
	"database/sql"
	"html/template"

	"time"
)

/*
App bundles shared resources like the database connection, template set,
and session settings. Handlers receive a pointer to this struct so they can
work with those shared pieces.
*/

type App struct {
	DB         *sql.DB
	Templates  map[string]*template.Template
	CookieName string
	SessionTTL time.Duration
}

