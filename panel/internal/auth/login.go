package auth

import (
	"log"
	"net/http"
	"strings"
	"time"

	"vietpanel/panel/internal/app"
	core "vietpanel/panel/internal/core/db"
	"vietpanel/panel/internal/web"
)

var loginLimiter = newRateLimiter(5, 10*time.Minute)

func LoginPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if FromContext(r) != nil {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
			return
		}
		web.Render(w, r, a, "user/login.html", web.PageData{
			Title: "Sign in",
			Next:  r.URL.Query().Get("next"),
		})
	}
}

func LoginPost(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !loginLimiter.allow(ip) {
			web.Render(w, r, a, "user/login.html", web.PageData{
				Title: "Sign in",
				Error: "Too many failed attempts. Please try again in a few minutes.",
			})
			return
		}
		username := r.FormValue("username")
		password := r.FormValue("password")
		id, name, email, err := core.VerifyUser(a.DB, username, password)
		if err != nil {
			loginLimiter.record(ip)
			log.Printf("auth: failed login for %q from %s", username, ip)
			core.LogActivity(a.DB, username, "login_failed", ip)
			web.Render(w, r, a, "user/login.html", web.PageData{
				Title: "Sign in",
				Error: "Invalid username or password.",
			})
			return
		}
		u := &app.User{ID: id, Username: name, Email: email}
		if err := SetSessionUser(w, r, a, u); err != nil {
			log.Printf("auth: session save: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		core.LogActivity(a.DB, name, "login", ip)

		// next is always a same-origin relative path (Require constructs it
		// from the request path), but validate shape anyway.
		next := firstNonEmpty(r.FormValue("next"), r.URL.Query().Get("next"))
		if next != "" && strings.HasPrefix(next, "/") && !strings.HasPrefix(next, "//") {
			http.Redirect(w, r, next, http.StatusFound)
			return
		}
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func LogoutPost(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u := FromContext(r); u != nil {
			core.LogActivity(a.DB, u.Username, "logout", clientIP(r))
		}
		ClearSession(w, r, a)
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}
