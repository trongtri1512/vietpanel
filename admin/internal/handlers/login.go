// Package handlers holds the admin panel's HTTP handlers.
package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"vietpanel/admin/internal/app"
	"vietpanel/admin/internal/auth"
	"vietpanel/admin/internal/db"
	"vietpanel/admin/internal/webtemplates"
)

// Login bundles the login/logout handlers and their shared dependencies,
// mirroring the original admin panel's flow.
type Login struct {
	App     *app.App
	Limiter *auth.PerIPLimiter

	mu             sync.Mutex
	failedAttempts map[string]int
}

func NewLogin(a *app.App) *Login {
	return &Login{App: a, Limiter: a.Limiter, failedAttempts: map[string]int{}}
}

func (l *Login) loginLogPath() string {
	return filepath.Join(l.App.Cfg.DataDir, "login.log")
}

func appendLogLine(path, line string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintln(f, line)
}

func (l *Login) renderLoginPage(w http.ResponseWriter, r *http.Request) {
	webtemplates.Render(w, r, "login.html", webtemplates.PageData{
		Title:   "Sign in",
		Next:    r.URL.Query().Get("next"),
		Flashes: auth.PopFlashes(w, r, l.App.Sessions),
	})
}

// ServeLoginPage handles GET /login.
func (l *Login) ServeLoginPage(w http.ResponseWriter, r *http.Request) {
	l.renderLoginPage(w, r)
}

// HandleLoginSubmit handles POST /login.
func (l *Login) HandleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	ip := auth.ClientIP(r)

	if !l.Limiter.Allow(ip) {
		appendLogLine(l.loginLogPath(), fmt.Sprintf("%s Rate limit for login exceeded from IP: %s", time.Now().Format("2006-01-02 15:04:05"), ip))
		l.mu.Lock()
		l.failedAttempts[ip]++
		l.mu.Unlock()
		auth.AddFlash(w, r, l.App.Sessions, "Too many failed login attempts. Please try again later.", "danger")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := db.UserByUsername(l.App.DB, username)
	if err != nil || !db.CheckPasswordHash(user.PasswordHash, password) {
		l.Limiter.Fail(ip)
		auth.AddFlash(w, r, l.App.Sessions, "Login failed. Please check your credentials.", "danger")
		l.renderLoginPage(w, r)
		return
	}

	if !user.IsActive {
		auth.AddFlash(w, r, l.App.Sessions, "Login failed. User is not active.", "danger")
		l.renderLoginPage(w, r)
		return
	}

	l.finalizeLogin(w, r, user, ip)
}

func (l *Login) finalizeLogin(w http.ResponseWriter, r *http.Request, user *db.User, ip string) {
	appendLogLine(l.loginLogPath(), fmt.Sprintf("%s %s %s", time.Now().Format("2006-01-02 15:04:05"), user.Username, ip))

	if err := l.App.Sessions.LoginUser(w, r, user); err != nil {
		log.Printf("Login session save failed for user %s (%s): %v", user.Username, ip, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	l.mu.Lock()
	delete(l.failedAttempts, ip)
	l.mu.Unlock()

	// next is always a same-origin relative path (RequireLogin constructs it
	// from the request path), but validate shape anyway.
	next := firstNonEmpty(r.FormValue("next"), r.URL.Query().Get("next"))
	if next != "" && strings.HasPrefix(next, "/") && !strings.HasPrefix(next, "//") {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// HandleLogout handles GET /logout.
func (l *Login) HandleLogout(w http.ResponseWriter, r *http.Request) {
	l.App.Sessions.LogoutUser(w, r)
	auth.AddFlash(w, r, l.App.Sessions, "Logged out successfully", "success")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
