package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Open opens (and creates parent dir for) the SQLite DB at path.
func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_foreign_keys=on", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	// Reasonable limits
	db.SetMaxOpenConns(1) // SQLite works best with 1 writer
	db.SetConnMaxLifetime(0)
	return db, nil
}

// Migrate runs the schema SQL file.
func Migrate(db *sql.DB, schemaPath string) error {
	b, err := os.ReadFile(schemaPath)
	if err != nil { return err }
	_, err = db.Exec(string(b))
	return err
}

// SeedDefaultCategories inserts a few categories if none exist.
func SeedDefaultCategories(db *sql.DB) error {
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&n); err != nil {
		return err
	}
	if n > 0 { return nil }
	cats := []struct{ name, slug string }{
		{"general", "general"},
		{"help", "help"},
		{"random", "random"},
		{"announcements", "announcements"},
		{"show-and-tell", "show-and-tell"},
	}
	tx, err := db.Begin()
	if err != nil { return err }
	defer func(){ _ = tx.Rollback() }()
	for _, c := range cats {
		if _, err := tx.Exec("INSERT INTO categories(name, slug) VALUES(?,?)", c.name, c.slug); err != nil {
			return err
		}
	}
	return tx.Commit()
}

var ErrNotFound = errors.New("not found")

// WithTimeout gives a context with reasonable timeout for DB ops.
func WithTimeout() (context.Context, context.CancelFunc) { return context.WithTimeout(context.Background(), 5*time.Second) }
