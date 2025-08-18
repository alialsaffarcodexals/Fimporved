package http

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	authpkg "forum/internal/auth"
	model "forum/internal/models"
)

type App struct {
	DB           *sql.DB
	Templates    map[string]*template.Template
	SessionTTL   time.Duration
	CookieSecure bool
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	// Static
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	mux.HandleFunc("/", a.handleHome)
	mux.HandleFunc("/register", a.handleRegister)
	mux.HandleFunc("/login", a.handleLogin)
	mux.HandleFunc("/logout", a.handleLogout)

	mux.HandleFunc("/post/new", a.requireAuth(a.handleNewPost))
	mux.HandleFunc("/post/create", a.requireAuth(a.handleCreatePost)) // internal redirect from /post/new form

	mux.HandleFunc("/post/", a.handlePostDetail) // also comment/like endpoints under this path
	mux.HandleFunc("/me/posts", a.requireAuth(a.handleMyPosts))
	mux.HandleFunc("/me/likes", a.requireAuth(a.handleMyLikes))
	mux.HandleFunc("/comment/", a.requireAuth(a.handleCommentVote))

	mw := &Middleware{DB: a.DB, SessionTTL: a.SessionTTL}
	var h http.Handler = mux
	h = mw.LoadUser(h)
	h = mw.Logging(h)
	return h
}

func (a *App) render(w http.ResponseWriter, r *http.Request, name string, data any) {
	tmpl, ok := a.Templates[name]
	if !ok {
		log.Printf("template %s not found", name)
		http.Error(w, "template error", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, "template error", 500)
	}
}

func (a *App) errorPage(w http.ResponseWriter, r *http.Request, status int, msg string) {
	w.WriteHeader(status)
	a.render(w, r, "error.gohtml", map[string]any{"Status": status, "Message": msg, "User": UserFromContext(r.Context())})
}

func (a *App) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			next(w, r) // allow GET to render forms guarded elsewhere
			return
		}
		u := UserFromContext(r.Context())
		if u == nil {
			a.errorPage(w, r, http.StatusUnauthorized, "Please log in to continue.")
			return
		}
		next(w, r)
	}
}

// ---------- Handlers ----------

func (a *App) handleHome(w http.ResponseWriter, r *http.Request) {
	u := UserFromContext(r.Context())
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	mine := r.URL.Query().Get("mine") == "1" || r.URL.Query().Get("mine") == "true"
	liked := r.URL.Query().Get("liked") == "1" || r.URL.Query().Get("liked") == "true"

	filters := model.ListFilters{CategorySlug: category, Limit: 50}
	if u != nil && mine {
		filters.MineUserID = u.ID
	}
	if u != nil && liked {
		filters.LikedByUserID = u.ID
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	posts, err := model.ListPosts(ctx, a.DB, filters)
	if err != nil {
		a.errorPage(w, r, 500, "failed to list posts")
		return
	}

	cats, err := model.GetCategories(ctx, a.DB)
	if err != nil {
		a.errorPage(w, r, 500, "failed to load categories")
		return
	}

	a.render(w, r, "home.gohtml", map[string]any{
		"User":           u,
		"Posts":          posts,
		"Categories":     cats,
		"FilterCategory": category,
		"FilterMine":     mine,
		"FilterLiked":    liked,
	})
}

func (a *App) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		a.render(w, r, "register.gohtml", map[string]any{"User": UserFromContext(r.Context())})
		return
	}
	if r.Method != http.MethodPost {
		a.errorPage(w, r, 405, "method not allowed")
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	username := strings.TrimSpace(r.FormValue("username"))
	password := strings.TrimSpace(r.FormValue("password"))

	if !validEmail(email) || len(username) < 3 || len(password) < 6 {
		a.errorPage(w, r, 400, "Invalid input. Ensure a valid email, username >= 3, password >= 6.")
		return
	}
	hash, err := authpkg.HashPassword(password)
	if err != nil {
		a.errorPage(w, r, 500, "failed to hash password")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	_, err = authpkg.CreateUser(ctx, a.DB, email, username, hash)
	if err != nil {
		// likely unique constraint
		a.errorPage(w, r, 400, "Email or username already taken.")
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		a.render(w, r, "login.gohtml", map[string]any{"User": UserFromContext(r.Context())})
		return
	}
	if r.Method != http.MethodPost {
		a.errorPage(w, r, 405, "method not allowed")
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	password := strings.TrimSpace(r.FormValue("password"))

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	u, ph, err := authpkg.GetUserByEmail(ctx, a.DB, email)
	if err != nil {
		a.errorPage(w, r, 401, "Invalid email or password.")
		return
	}
	if err := authpkg.CheckPassword(ph, password); err != nil {
		a.errorPage(w, r, 401, "Invalid email or password.")
		return
	}

	sid, err := authpkg.UpsertSession(ctx, a.DB, u.ID, a.SessionTTL)
	if err != nil {
		a.errorPage(w, r, 500, "failed to create session")
		return
	}

	cookie := &http.Cookie{Name: "session_id", Value: sid, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode}
	cookie.Secure = a.CookieSecure
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.errorPage(w, r, 405, "method not allowed")
		return
	}
	cookie, err := r.Cookie("session_id")
	if err == nil {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		_ = authpkg.DeleteSessionByID(ctx, a.DB, cookie.Value)
	}
	c := &http.Cookie{Name: "session_id", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode}
	c.Secure = a.CookieSecure
	http.SetCookie(w, c)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) handleNewPost(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.handleCreatePost(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	cats, err := model.GetCategories(ctx, a.DB)
	if err != nil {
		a.errorPage(w, r, 500, "failed to load categories")
		return
	}
	a.render(w, r, "new_post.gohtml", map[string]any{"User": UserFromContext(r.Context()), "Categories": cats})
}

func (a *App) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	u := UserFromContext(r.Context())
	if u == nil {
		a.errorPage(w, r, 401, "login required")
		return
	}
	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))
	if title == "" || content == "" || len(title) > 2000 || len(content) > 10000 {
		a.errorPage(w, r, 400, "Title and content are required (sane lengths).")
		return
	}
	var catIDs []int64
	for key, vals := range r.PostForm {
		if !strings.HasPrefix(key, "cat_") {
			continue
		}
		if len(vals) == 0 || vals[0] != "1" {
			continue
		}
		idStr := strings.TrimPrefix(key, "cat_")
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			catIDs = append(catIDs, id)
		}
	}
	if len(catIDs) == 0 {
		a.errorPage(w, r, 400, "Select at least one category.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	pid, err := model.CreatePost(ctx, a.DB, u.ID, title, content, catIDs)
	if err != nil {
		a.errorPage(w, r, 500, "failed to create post")
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/post/%d", pid), http.StatusSeeOther)
}

func (a *App) handlePostDetail(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "post" {
		a.errorPage(w, r, 404, "not found")
		return
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		a.errorPage(w, r, 404, "not found")
		return
	}

	if len(parts) == 2 && r.Method == http.MethodGet {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		p, err := model.GetPost(ctx, a.DB, id)
		if err != nil {
			if errors.Is(err, model.ErrNotFound) {
				a.errorPage(w, r, 404, "post not found")
				return
			}
			a.errorPage(w, r, 500, "failed to load post")
			return
		}
		comments, err := model.ListComments(ctx, a.DB, id)
		if err != nil {
			a.errorPage(w, r, 500, "failed to load comments")
			return
		}
		cats, _ := model.GetCategories(ctx, a.DB) // for sidebar
		a.render(w, r, "post_detail.gohtml", map[string]any{"User": UserFromContext(r.Context()), "Post": p, "Comments": comments, "Categories": cats})
		return
	}

	// sub-actions
	if len(parts) >= 3 && r.Method == http.MethodPost {
		u := UserFromContext(r.Context())
		if u == nil {
			a.errorPage(w, r, 401, "login required")
			return
		}
		switch parts[2] {
		case "comment":
			body := strings.TrimSpace(r.FormValue("body"))
			if body == "" {
				a.errorPage(w, r, 400, "comment cannot be empty")
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			if _, err := model.AddComment(ctx, a.DB, id, u.ID, body); err != nil {
				a.errorPage(w, r, 500, "failed to add comment")
				return
			}
			http.Redirect(w, r, fmt.Sprintf("/post/%d", id), http.StatusSeeOther)
			return
		case "like":
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			if err := model.TogglePostVote(ctx, a.DB, u.ID, id, 1); err != nil {
				a.errorPage(w, r, 500, "failed to like")
				return
			}
			http.Redirect(w, r, fmt.Sprintf("/post/%d", id), http.StatusSeeOther)
			return
		case "dislike":
			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			if err := model.TogglePostVote(ctx, a.DB, u.ID, id, -1); err != nil {
				a.errorPage(w, r, 500, "failed to dislike")
				return
			}
			http.Redirect(w, r, fmt.Sprintf("/post/%d", id), http.StatusSeeOther)
			return
		}
	}

	a.errorPage(w, r, 404, "not found")
}

func (a *App) handleMyPosts(w http.ResponseWriter, r *http.Request) {
	u := UserFromContext(r.Context())
	if u == nil {
		a.errorPage(w, r, 401, "login required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	cats, _ := model.GetCategories(ctx, a.DB)
	posts, err := model.ListPosts(ctx, a.DB, model.ListFilters{MineUserID: u.ID, Limit: 50})
	if err != nil {
		a.errorPage(w, r, 500, "failed to list posts")
		return
	}
	a.render(w, r, "home.gohtml", map[string]any{"User": u, "Posts": posts, "Categories": cats, "FilterMine": true})
}

func (a *App) handleMyLikes(w http.ResponseWriter, r *http.Request) {
	u := UserFromContext(r.Context())
	if u == nil {
		a.errorPage(w, r, 401, "login required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	cats, _ := model.GetCategories(ctx, a.DB)
	posts, err := model.ListPosts(ctx, a.DB, model.ListFilters{LikedByUserID: u.ID, Limit: 50})
	if err != nil {
		a.errorPage(w, r, 500, "failed to list posts")
		return
	}
	a.render(w, r, "home.gohtml", map[string]any{"User": u, "Posts": posts, "Categories": cats, "FilterLiked": true})
}

// Utility

func validEmail(e string) bool {
	if len(e) < 3 || len(e) > 254 {
		return false
	}
	if strings.Count(e, "@") != 1 {
		return false
	}
	parts := strings.Split(e, "@")
	if len(parts[0]) == 0 || len(parts[1]) < 3 {
		return false
	}
	if !strings.Contains(parts[1], ".") {
		return false
	}
	return true
}

// ---------- App bootstrap ----------

func NewApp(db *sql.DB, sessionTTL time.Duration, cookieSecure bool) (*App, error) {
	// Parse templates
	funcs := template.FuncMap{
		"joinCats": func(cats []model.Category) string {
			var slugs []string
			for _, c := range cats {
				slugs = append(slugs, c.Slug)
			}
			return strings.Join(slugs, ", ")
		},
		"formatTime": func(t time.Time) string {
			return t.Local().Format("2006-01-02 15:04")
		},
	}
	tmplDir := path.Join("internal", "views", "templates")
	entries, err := os.ReadDir(tmplDir)
	if err != nil {
		return nil, err
	}
	templates := make(map[string]*template.Template)
	base := path.Join(tmplDir, "base.gohtml")
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == "base.gohtml" || !strings.HasSuffix(name, ".gohtml") {
			continue
		}
		t, err := template.New("base").Funcs(funcs).ParseFiles(base, path.Join(tmplDir, name))
		if err != nil {
			return nil, err
		}
		templates[name] = t
	}
	return &App{DB: db, Templates: templates, SessionTTL: sessionTTL, CookieSecure: cookieSecure}, nil
}

func StartServer(addr string, app *App) error {
	server := &http.Server{
		Addr:         addr,
		Handler:      app.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("listening on %s", addr)
	return server.ListenAndServe()
}

// Helpers for tests
func NowUTC() time.Time { return time.Now().UTC() }

func MustGetEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

// Extra routes for comment likes/dislikes need to be registered at root level.
// We'll provide a small adapter here.
func init() {}

// In routes(), add handlers for comment actions (without a GET page).
func (a *App) handleCommentVote(w http.ResponseWriter, r *http.Request) {
	// Path: /comment/{id}/like or /comment/{id}/dislike
	if r.Method != http.MethodPost {
		a.errorPage(w, r, 405, "method not allowed")
		return
	}
	u := UserFromContext(r.Context())
	if u == nil {
		a.errorPage(w, r, 401, "login required")
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "comment" {
		a.errorPage(w, r, 404, "not found")
		return
	}
	cid, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		a.errorPage(w, r, 400, "bad id")
		return
	}
	action := parts[2]
	val := 0
	if action == "like" {
		val = 1
	} else if action == "dislike" {
		val = -1
	} else {
		a.errorPage(w, r, 404, "not found")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := model.ToggleCommentVote(ctx, a.DB, u.ID, cid, val); err != nil {
		a.errorPage(w, r, 500, "failed to vote")
		return
	}
	// Redirect back to referer if present, else home
	ref := r.Referer()
	if ref == "" {
		ref = "/"
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}
