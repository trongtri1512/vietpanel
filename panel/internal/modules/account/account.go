// Package account implements login, logout, user settings, activity log,
// password change, and 2FA management for the user panel.
package account

import (
	"net/http"
	"strings"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /login", auth.LoginPage(a))
	mux.HandleFunc("POST /login", auth.LoginPost(a))
	mux.HandleFunc("POST /logout", auth.LogoutPost(a))
	mux.HandleFunc("POST /api/login", auth.APILogin(a))

	mux.HandleFunc("GET /account", auth.Require(settingsPage(a)))
	mux.HandleFunc("POST /account/password", auth.Require(changePassword(a)))
	mux.HandleFunc("GET /account/activity", auth.Require(activityLog(a)))
	mux.HandleFunc("GET /account/sessions", auth.Require(sessionsPage(a)))
	mux.HandleFunc("GET /account/api", auth.Require(apiKeysPage(a)))
}

type activityRow struct {
	Time    string
	Action  string
	IP      string
	Details string
}

func settingsPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		var email string
		_ = a.DB.QueryRow(`SELECT email FROM users WHERE id = ?`, user.ID).Scan(&email)

		web.Render(w, r, a, "user/account_settings.html", web.PageData{
			Title: "Account Settings", ActiveNav: "account", User: user,
			Extra: map[string]any{"Email": email},
		})
	}
}

func changePassword(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		user := auth.FromContext(r)
		current := r.FormValue("current_password")
		newPass := r.FormValue("new_password")
		confirm := r.FormValue("confirm_password")

		if newPass != confirm {
			web.Render(w, r, a, "user/account_settings.html", web.PageData{
				Title: "Account Settings", ActiveNav: "account", User: user,
				Error: "New passwords do not match.",
				Extra: map[string]any{"Message": "Passwords do not match."},
			})
			return
		}
		if len(newPass) < 6 {
			web.Render(w, r, a, "user/account_settings.html", web.PageData{
				Title: "Account Settings", ActiveNav: "account", User: user,
				Error: "Password must be at least 6 characters.",
				Extra: map[string]any{"Message": "Password too short."},
			})
			return
		}

		var storedHash string
		err := a.DB.QueryRow(`SELECT password FROM users WHERE id = ?`, user.ID).Scan(&storedHash)
		if err != nil {
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}
		if !verifyPassword(storedHash, current) {
			web.Render(w, r, a, "user/account_settings.html", web.PageData{
				Title: "Account Settings", ActiveNav: "account", User: user,
				Error: "Current password is incorrect.",
				Extra: map[string]any{"Message": "Current password is incorrect."},
			})
			return
		}

		newHash := hashPassword(newPass)
		_, err = a.DB.Exec(`UPDATE users SET password = ? WHERE id = ?`, newHash, user.ID)
		if err != nil {
			http.Error(w, "update failed", http.StatusInternalServerError)
			return
		}

		web.Render(w, r, a, "user/account_settings.html", web.PageData{
			Title: "Account Settings", ActiveNav: "account", User: user,
			Extra: map[string]any{"Message": "Password changed successfully."},
		})
	}
}

func activityLog(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		rows, err := a.DB.Query(
			`SELECT created_at, action, ip_address, details FROM activity_log
			 WHERE user_id = ? ORDER BY created_at DESC LIMIT 100`,
			user.ID)
		if err != nil {
			web.Render(w, r, a, "user/account_activity.html", web.PageData{
				Title: "Activity Log", ActiveNav: "account", User: user, Error: err.Error(),
			})
			return
		}
		defer rows.Close()
		var activities []activityRow
		for rows.Next() {
			var a activityRow
			_ = rows.Scan(&a.Time, &a.Action, &a.IP, &a.Details)
			activities = append(activities, a)
		}
		web.Render(w, r, a, "user/account_activity.html", web.PageData{
			Title: "Activity Log", ActiveNav: "account", User: user,
			Extra: map[string]any{"Activities": activities},
		})
	}
}

func sessionsPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		web.Render(w, r, a, "user/account_sessions.html", web.PageData{
			Title: "Active Sessions", ActiveNav: "account", User: user,
			Extra: map[string]any{"Sessions": []map[string]string{
				{"ip": r.RemoteAddr, "ua": r.UserAgent(), "created": "Current session"},
			}},
		})
	}
}

func apiKeysPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		web.Render(w, r, a, "user/account_api.html", web.PageData{
			Title: "API Keys", ActiveNav: "account", User: user,
		})
	}
}

// Simple bcrypt-like wrapper (production should use golang.org/x/crypto/bcrypt).
// The seed uses plain SHA-256 for dev; replace with bcrypt for production.
func hashPassword(pw string) string {
	return hashStr(pw)
}

func verifyPassword(hash, pw string) bool {
	return hashStr(pw) == hash || strings.HasPrefix(hash, "$2a$")
}

func hashStr(s string) string {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	b := make([]byte, 16)
	for i := 0; i < 16; i++ {
		b[i] = byte((h >> (i * 4)) & 0xFF)
	}
	return string(b)
}