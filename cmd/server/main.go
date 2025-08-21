package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"forum-mvp/internal/app"

	_ "github.com/mattn/go-sqlite3"
)

/*
main configures the application and starts the HTTP server.
It wires up the database, templates and routes.
*/
func main() {
	addr := flag.String("addr", ":8080", "http listen address")
	dataDir := flag.String("data", "./data", "data directory for sqlite")
	tplDir := flag.String("templates", "./internal/web/templates", "templates dir")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatal(err)
	}

	dbPath := filepath.Join(*dataDir, "forum.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	schema, err := os.ReadFile("internal/db/schema.sql")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		log.Fatal(err)
	}

	tpls, err := loadTemplates(*tplDir)
	if err != nil {
		log.Fatal(err)
	}

	a := &app.App{DB: db, Templates: tpls, CookieName: "forum_session", SessionTTL: 7 * 24 * time.Hour}

	/*
	   Set up routes for the application.
	*/
	http.HandleFunc("/", a.HandleIndex)
	http.HandleFunc("/register", a.HandleRegister)
	http.HandleFunc("/login", a.HandleLogin)
	http.HandleFunc("/logout", a.HandleLogout)
	http.HandleFunc("/post", a.HandleShowPost)
	http.HandleFunc("/post/new", a.RequireAuth(a.HandleNewPost))
	http.HandleFunc("/comment/new", a.RequireAuth(a.HandleNewComment))
	http.HandleFunc("/like", a.RequireAuth(a.HandleLike))

	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, http.DefaultServeMux))
}
