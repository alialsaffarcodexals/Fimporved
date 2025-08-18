package auth_test

import (
	"path/filepath"
	"testing"

	"database/sql"

	model "forum/internal/models"
	db "forum/internal/db"
	auth "forum/internal/auth"
)

func withDBT(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "test.db")
	dbConn, err := db.Open(p)
	if err != nil { t.Fatalf("open: %v", err) }
	if err := db.Migrate(dbConn, filepath.Join("sql", "schema.sql")); err != nil { t.Fatalf("migrate: %v", err) }
	if err := db.SeedDefaultCategories(dbConn); err != nil { t.Fatalf("seed: %v", err) }
	return dbConn
}

func TestVoteToggle(t *testing.T) {
	dbConn := withDBT(t)
	defer dbConn.Close()

	h,_ := auth.HashPassword("x")
	uid, err := auth.CreateUser(nil, dbConn, "u@example.com", "u", h)
	if err != nil { t.Fatalf("create user: %v", err) }
	pid, err := model.CreatePost(nil, dbConn, uid, "hello", "world", nil)
	if err != nil { t.Fatalf("create post: %v", err) }

	// like
	if err := model.TogglePostVote(nil, dbConn, uid, pid, 1); err != nil { t.Fatalf("like: %v", err) }
	// like again -> remove
	if err := model.TogglePostVote(nil, dbConn, uid, pid, 1); err != nil { t.Fatalf("unlike: %v", err) }
	// dislike
	if err := model.TogglePostVote(nil, dbConn, uid, pid, -1); err != nil { t.Fatalf("dislike: %v", err) }
	// switch to like
	if err := model.TogglePostVote(nil, dbConn, uid, pid, 1); err != nil { t.Fatalf("switch: %v", err) }
}
