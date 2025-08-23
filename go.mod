module forum

go 1.20

require (
	// UUID library for generating unique session IDs.
	github.com/google/uuid v1.3.0
	// SQLite driver for Go. Required to talk to our local database.
	github.com/mattn/go-sqlite3 v1.14.17
	// Crypto package containing bcrypt for secure password hashing.
	golang.org/x/crypto v0.17.0
)
