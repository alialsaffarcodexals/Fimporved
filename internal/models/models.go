package models

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Category struct {
	ID int64
	Name string
	Slug string
}

type Post struct {
	ID int64
	UserID int64
	Title string
	Content string
	CreatedAt time.Time
	Author string
	Likes int
	Dislikes int
	Categories []Category
}

type Comment struct {
	ID int64
	PostID int64
	UserID int64
	Body string
	CreatedAt time.Time
	Author string
	Likes int
	Dislikes int
}

var ErrNotFound = errors.New("not found")

func GetCategories(ctx context.Context, db *sql.DB) ([]Category, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, name, slug FROM categories ORDER BY name ASC")
	if err != nil { return nil, err }
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug); err != nil { return nil, err }
		out = append(out, c)
	}
	return out, rows.Err()
}

type ListFilters struct {
	CategorySlug string
	MineUserID int64
	LikedByUserID int64
	Limit int
	Offset int
}

// ListPosts returns posts with aggregates and optional filters.
func ListPosts(ctx context.Context, db *sql.DB, f ListFilters) ([]Post, error) {
	if f.Limit <= 0 { f.Limit = 50 }
	args := []any{}
	where := "WHERE 1=1"
	if f.CategorySlug != "" {
		where += " AND EXISTS (SELECT 1 FROM post_categories pc JOIN categories c ON c.id=pc.category_id WHERE pc.post_id=p.id AND c.slug = ?)"
		args = append(args, f.CategorySlug)
	}
	if f.MineUserID > 0 {
		where += " AND p.user_id = ?"
		args = append(args, f.MineUserID)
	}
	if f.LikedByUserID > 0 {
		where += " AND EXISTS (SELECT 1 FROM post_votes pvx WHERE pvx.post_id = p.id AND pvx.user_id = ? AND pvx.value = 1)"
		args = append(args, f.LikedByUserID)
	}
	q := `
		SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.username,
			COALESCE(SUM(CASE WHEN pv.value = 1 THEN 1 ELSE 0 END), 0) AS likes,
			COALESCE(SUM(CASE WHEN pv.value = -1 THEN 1 ELSE 0 END), 0) AS dislikes
		FROM posts p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN post_votes pv ON pv.post_id = p.id
		` + where + `
		GROUP BY p.id
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?
	`
	args = append(args, f.Limit, f.Offset)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.CreatedAt, &p.Author, &p.Likes, &p.Dislikes); err != nil { return nil, err }
		out = append(out, p)
	}
	if err := rows.Err(); err != nil { return nil, err }
	// Attach categories for each post
	for i := range out {
		cats, err := GetCategoriesForPost(ctx, db, out[i].ID)
		if err != nil { return nil, err }
		out[i].Categories = cats
	}
	return out, nil
}

func GetCategoriesForPost(ctx context.Context, db *sql.DB, postID int64) ([]Category, error) {
	rows, err := db.QueryContext(ctx, `SELECT c.id, c.name, c.slug FROM categories c JOIN post_categories pc ON pc.category_id = c.id WHERE pc.post_id = ? ORDER BY c.name`, postID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug); err != nil { return nil, err }
		out = append(out, c)
	}
	return out, rows.Err()
}

func CreatePost(ctx context.Context, db *sql.DB, userID int64, title, content string, categoryIDs []int64) (int64, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil { return 0, err }
	defer func(){ _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, "INSERT INTO posts(user_id, title, content) VALUES(?,?,?)", userID, title, content)
	if err != nil { return 0, err }
	id, err := res.LastInsertId()
	if err != nil { return 0, err }
	for _, cid := range categoryIDs {
		if _, err := tx.ExecContext(ctx, "INSERT INTO post_categories(post_id, category_id) VALUES(?,?)", id, cid); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil { return 0, err }
	return id, nil
}

func GetPost(ctx context.Context, db *sql.DB, id int64) (*Post, error) {
	row := db.QueryRowContext(ctx, `
		SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.username,
			COALESCE(SUM(CASE WHEN pv.value = 1 THEN 1 ELSE 0 END), 0) AS likes,
			COALESCE(SUM(CASE WHEN pv.value = -1 THEN 1 ELSE 0 END), 0) AS dislikes
		FROM posts p JOIN users u ON u.id = p.user_id
		LEFT JOIN post_votes pv ON pv.post_id = p.id
		WHERE p.id = ?
		GROUP BY p.id
	`, id)
	var p Post
	if err := row.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.CreatedAt, &p.Author, &p.Likes, &p.Dislikes); err != nil {
		if err == sql.ErrNoRows { return nil, ErrNotFound }
		return nil, err
	}
	cats, err := GetCategoriesForPost(ctx, db, id)
	if err != nil { return nil, err }
	p.Categories = cats
	return &p, nil
}

func ListComments(ctx context.Context, db *sql.DB, postID int64) ([]Comment, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT c.id, c.post_id, c.user_id, c.body, c.created_at, u.username,
			COALESCE(SUM(CASE WHEN cv.value = 1 THEN 1 ELSE 0 END), 0) AS likes,
			COALESCE(SUM(CASE WHEN cv.value = -1 THEN 1 ELSE 0 END), 0) AS dislikes
		FROM comments c JOIN users u ON u.id = c.user_id
		LEFT JOIN comment_votes cv ON cv.comment_id = c.id
		WHERE c.post_id = ?
		GROUP BY c.id
		ORDER BY c.created_at ASC
	`, postID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Body, &c.CreatedAt, &c.Author, &c.Likes, &c.Dislikes); err != nil { return nil, err }
		out = append(out, c)
	}
	return out, rows.Err()
}

func AddComment(ctx context.Context, db *sql.DB, postID, userID int64, body string) (int64, error) {
	res, err := db.ExecContext(ctx, "INSERT INTO comments(post_id, user_id, body) VALUES(?,?,?)", postID, userID, body)
	if err != nil { return 0, err }
	return res.LastInsertId()
}

// Toggle vote: if same value exists, delete; otherwise upsert to new value.
func TogglePostVote(ctx context.Context, db *sql.DB, userID, postID int64, value int) error {
	var cur int
	err := db.QueryRowContext(ctx, "SELECT value FROM post_votes WHERE user_id=? AND post_id=?", userID, postID).Scan(&cur)
	if err == nil {
		if cur == value {
			_, err := db.ExecContext(ctx, "DELETE FROM post_votes WHERE user_id=? AND post_id=?", userID, postID)
			return err
		}
		_, err := db.ExecContext(ctx, "UPDATE post_votes SET value=? WHERE user_id=? AND post_id=?", value, userID, postID)
		return err
	}
	if err != sql.ErrNoRows { return err }
	_, err = db.ExecContext(ctx, "INSERT INTO post_votes(user_id, post_id, value) VALUES(?,?,?)", userID, postID, value)
	return err
}

func ToggleCommentVote(ctx context.Context, db *sql.DB, userID, commentID int64, value int) error {
	var cur int
	err := db.QueryRowContext(ctx, "SELECT value FROM comment_votes WHERE user_id=? AND comment_id=?", userID, commentID).Scan(&cur)
	if err == nil {
		if cur == value {
			_, err := db.ExecContext(ctx, "DELETE FROM comment_votes WHERE user_id=? AND comment_id=?", userID, commentID)
			return err
		}
		_, err := db.ExecContext(ctx, "UPDATE comment_votes SET value=? WHERE user_id=? AND comment_id=?", value, userID, commentID)
		return err
	}
	if err != sql.ErrNoRows { return err }
	_, err = db.ExecContext(ctx, "INSERT INTO comment_votes(user_id, comment_id, value) VALUES(?,?,?)", userID, commentID, value)
	return err
}
