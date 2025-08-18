package main

import (
	
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	httpapp "forum/internal/http"
	dbpkg "forum/internal/db"
)

func main() {
	port := httpapp.MustGetEnv("PORT", "8080")
	dbPath := httpapp.MustGetEnv("DB_PATH", "data/forum.db")
	ttlHoursStr := httpapp.MustGetEnv("SESSION_TTL_HOURS", "24")
	cookieSecure := httpapp.MustGetEnv("COOKIE_SECURE", "false") == "true"

	var ttlHours int
	if n, err := strconv.Atoi(ttlHoursStr); err == nil && n > 0 { ttlHours = n } else { ttlHours = 24 }

	db, err := dbpkg.Open(dbPath)
	if err != nil { log.Fatalf("open db: %v", err) }
	defer db.Close()

	if err := dbpkg.Migrate(db, "sql/schema.sql"); err != nil { log.Fatalf("migrate: %v", err) }
	if err := dbpkg.SeedDefaultCategories(db); err != nil { log.Fatalf("seed: %v", err) }

	app, err := httpapp.NewApp(db, time.Duration(ttlHours)*time.Hour, cookieSecure)
	if err != nil { log.Fatalf("init app: %v", err) }

	addr := fmt.Sprintf(":%s", port)
	if err := httpapp.StartServer(addr, app); err != nil {
		log.Println("server stopped:", err)
		os.Exit(1)
	}
}
