package main

// The main package sets up the database, parses HTML templates and wires
// together HTTP routes. This entry point stays intentionally small to
// emphasize how the application is composed from smaller packages.

import (
    "database/sql"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "time"

    // Import our internal packages.  Note that the module name declared in
    // go.mod is `forum`, so any packages inside the repository can be
    // referenced as `forum/internal/...`.
    "forum/internal/app"
    "forum/internal/server"

    // Register the sqlite3 driver. Without the blank import the driver
    // doesn't register itself and sql.Open would fail.
    _ "github.com/mattn/go-sqlite3"
)

/*
main configures the application and starts the HTTP server.

It parses a few flags that allow the caller to customise the HTTP
listen address, the location where the SQLite database file is
created and the directory containing our HTML templates. Next it
ensures the data directory exists, opens the database and runs our
schema creation script. Finally it loads the HTML templates, wires
together the HTTP handlers and starts the web server.
*/
func main() {
    // Define command‑line flags. These allow the developer to override
    // defaults without changing code. The `addr` flag controls the
    // listening address, `data` controls where the SQLite file is
    // stored and `templates` points at the directory containing our
    // template files.
    addr := flag.String("addr", ":8080", "http listen address")
    dataDir := flag.String("data", "./data", "data directory for sqlite")
    tplDir := flag.String("templates", "./internal/web/templates", "templates dir")
    flag.Parse()

    // Create the data directory if it doesn't already exist. The
    // permissions here (0755) allow the owner to read/write and others
    // to read. If this fails we'll abort the application because it
    // won't be able to persist any data.
    if err := os.MkdirAll(*dataDir, 0o755); err != nil {
        log.Fatalf("unable to create data directory: %v", err)
    }

    // Open (or create) the SQLite database. The `database/sql` package
    // handles connection pooling for us. Note the driver name `sqlite3`
    // comes from the blank import above.
    dbPath := filepath.Join(*dataDir, "forum.db")
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        log.Fatalf("unable to open database: %v", err)
    }
    defer db.Close()

    // Read the schema SQL from our embedded file and execute it. The
    // schema contains `CREATE TABLE` statements wrapped in IF NOT
    // EXISTS clauses so it's safe to run on every startup.
    schema, err := os.ReadFile("internal/db/schema.sql")
    if err != nil {
        log.Fatalf("failed reading schema: %v", err)
    }
    if _, err := db.Exec(string(schema)); err != nil {
        log.Fatalf("failed executing schema: %v", err)
    }

    // Parse all templates in the provided directory. The server
    // package walks the directory, loads the shared layout and parses
    // each page into a single Template object. The returned map is
    // keyed by filename (e.g. index.html).
    tpls, err := server.LoadTemplates(*tplDir)
    if err != nil {
        log.Fatalf("failed loading templates: %v", err)
    }

    // Build the application context. All HTTP handlers receive a
    // pointer to this struct so they can access the shared database,
    // templates and session configuration. CookieName is the name of
    // the session cookie and SessionTTL determines how long a login
    // session should live.
    appCtx := &app.App{
        DB:         db,
        Templates:  tpls,
        CookieName: "forum_session",
        SessionTTL: 7 * 24 * time.Hour, // one week
    }

    // Set up the HTTP routes. We use a ServeMux rather than
    // http.DefaultServeMux so that no third party packages can insert
    // handlers without us noticing. Some routes are wrapped in the
    // RequireAuth middleware to ensure the user is logged in before
    // proceeding.
    mux := http.NewServeMux()
    mux.HandleFunc("/", appCtx.HandleIndex)
    mux.HandleFunc("/register", appCtx.HandleRegister)
    mux.HandleFunc("/login", appCtx.HandleLogin)
    mux.HandleFunc("/logout", appCtx.HandleLogout)
    mux.HandleFunc("/post", appCtx.HandleShowPost)
    mux.HandleFunc("/post/new", appCtx.RequireAuth(appCtx.HandleNewPost))
    mux.HandleFunc("/comment/new", appCtx.RequireAuth(appCtx.HandleNewComment))
    mux.HandleFunc("/like", appCtx.RequireAuth(appCtx.HandleLike))
    // Serve static assets such as CSS and images from the
    // internal/web/static directory. The files are served under the
    // /static/ prefix.
    mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./internal/web/static"))))

    // Wrap the mux in our middleware. WithCustomErrors will trap
    // panics (500) and intercept 404 responses, rendering the
    // appropriate error pages. LogRequest records each request in the
    // terminal along with the duration.
    handler := server.LogRequest(server.WithCustomErrors(mux, appCtx))

    // Start the HTTP server. ListenAndServe returns only when the
    // server shuts down or fails to bind. We print a friendly message
    // letting the user know where to point their browser.
    fmt.Printf("listening on %s\n", *addr)
    if err := http.ListenAndServe(*addr, handler); err != nil {
        log.Fatalf("server error: %v", err)
    }
}