package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// HashPassword hashes with bcrypt cost 12.
func HashPassword(plain string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(plain), 12)
}

func CheckPassword(hash []byte, plain string) error {
	return bcrypt.CompareHashAndPassword(hash, []byte(plain))
}

// CreateUser inserts a user.
func CreateUser(ctx context.Context, db *sql.DB, email, username string, passwordHash []byte) (int64, error) {
	res, err := db.ExecContext(ctx, "INSERT INTO users(email, username, password_hash) VALUES(?,?,?)", email, username, string(passwordHash))
	if err != nil { return 0, err }
	return res.LastInsertId()
}

type User struct {
	ID int64
	Email string
	Username string
}

func GetUserByEmail(ctx context.Context, db *sql.DB, email string) (*User, []byte, error) {
	row := db.QueryRowContext(ctx, "SELECT id, email, username, password_hash FROM users WHERE email = ?", email)
	var u User
	var ph string
	if err := row.Scan(&u.ID, &u.Email, &u.Username, &ph); err != nil {
		if err == sql.ErrNoRows { return nil, nil, ErrInvalidCredentials }
		return nil, nil, err
	}
	return &u, []byte(ph), nil
}

// UpsertSession ensures exactly one active session per user. Returns new session ID.
func UpsertSession(ctx context.Context, db *sql.DB, userID int64, ttl time.Duration) (string, error) {
	sid := uuid.New().String()
	exp := time.Now().Add(ttl).UTC()
	_, err := db.ExecContext(ctx, `
		INSERT INTO sessions(id, user_id, expires_at) VALUES(?,?,?)
		ON CONFLICT(user_id) DO UPDATE SET id = excluded.id, expires_at = excluded.expires_at, created_at = CURRENT_TIMESTAMP
	`, sid, userID, exp)
	if err != nil { return "", err }
	return sid, nil
}

func DeleteSessionByID(ctx context.Context, db *sql.DB, id string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", id)
	return err
}

func GetUserBySession(ctx context.Context, db *sql.DB, sessionID string, now time.Time) (*User, error) {
	row := db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.username
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.id = ? AND s.expires_at > ?
	`, sessionID, now.UTC())
	var u User
	if err := row.Scan(&u.ID, &u.Email, &u.Username); err != nil {
		if err == sql.ErrNoRows { return nil, nil }
		return nil, err
	}
	return &u, nil
}
