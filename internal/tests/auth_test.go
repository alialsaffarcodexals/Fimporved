package auth_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	auth "forum/internal/auth"
	db "forum/internal/db"
)

func TestHashAndCheckPassword(t *testing.T) {
	h, err := auth.HashPassword("s3cret!123")
	if err != nil { t.Fatalf("hash err: %v", err) }
	if err := auth.CheckPassword(h, "s3cret!123"); err != nil { t.Fatalf("check failed: %v", err) }
	if err := auth.CheckPassword(h, "wrong"); err == nil { t.Fatalf("expected failure") }
}

func withDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "test.db")
	dbConn, err := db.Open(p)
	if err != nil { t.Fatalf("open: %v", err) }
	if err := db.Migrate(dbConn, filepath.Join("sql", "schema.sql")); err != nil { t.Fatalf("migrate: %v", err) }
	if err := db.SeedDefaultCategories(dbConn); err != nil { t.Fatalf("seed: %v", err) }
	return dbConn
}

func TestSessionCreateAndExpiry(t *testing.T) {
	dbConn := withDB(t)
	defer dbConn.Close()

	// Create a user
	h, _ := auth.HashPassword("pass1234")
	id, err := auth.CreateUser(nil, dbConn, "a@example.com", "alice", h)
	if err != nil { t.Fatalf("create user: %v", err) }

	// Create session with short TTL
	sid, err := auth.UpsertSession(nil, dbConn, id, time.Second)
	if err != nil { t.Fatalf("session: %v", err) }

	if u, err := auth.GetUserBySession(nil, dbConn, sid, time.Now()); err != nil || u == nil {
		t.Fatalf("expected valid session")
	}

	// After expiry
	time.Sleep(1200 * time.Millisecond)
	if u, _ := auth.GetUserBySession(nil, dbConn, sid, time.Now()); u != nil {
		t.Fatalf("expected expired session to be invalid")
	}
	_ = os.RemoveAll("data")
}
