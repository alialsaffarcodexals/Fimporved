package auth_test

import (
	"path/filepath"
	"testing"

	model "forum/internal/models"
	db "forum/internal/db"
	auth "forum/internal/auth"
)

func TestCategoryFilterQuery(t *testing.T) {
	dbConn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil { t.Fatal(err) }
	defer dbConn.Close()
	if err := db.Migrate(dbConn, filepath.Join("sql", "schema.sql")); err != nil { t.Fatal(err) }
	if err := db.SeedDefaultCategories(dbConn); err != nil { t.Fatal(err) }

	h,_ := auth.HashPassword("pw")
	uid, _ := auth.CreateUser(nil, dbConn, "x@example.com", "x", h)

	// find a category id for "general"
	var cid int64
	if err := dbConn.QueryRow("SELECT id FROM categories WHERE slug='general'").Scan(&cid); err != nil { t.Fatal(err) }

	pid, err := model.CreatePost(nil, dbConn, uid, "hello", "body", []int64{cid})
	if err != nil { t.Fatal(err) }

	// Like it
	if err := model.TogglePostVote(nil, dbConn, uid, pid, 1); err != nil { t.Fatal(err) }

	// Check filters
	posts, err := model.ListPosts(nil, dbConn, model.ListFilters{CategorySlug:"general"})
	if err != nil { t.Fatal(err) }
	if len(posts) != 1 { t.Fatalf("expected 1 post, got %d", len(posts)) }

	posts, err = model.ListPosts(nil, dbConn, model.ListFilters{MineUserID: uid})
	if err != nil || len(posts) != 1 { t.Fatalf("mine filter failed") }

	posts, err = model.ListPosts(nil, dbConn, model.ListFilters{LikedByUserID: uid})
	if err != nil || len(posts) != 1 { t.Fatalf("liked filter failed") }
}
