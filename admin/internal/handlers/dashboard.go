package handlers

import (
	"net/http"
	"os"

	"vietpanel/admin/internal/app"
	"vietpanel/admin/internal/auth"
	"vietpanel/admin/internal/webtemplates"
)

// Dashboard renders the admin landing page with counts pulled from the user
// panel's MySQL store when one is configured.
func Dashboard(a *app.App) http.HandlerFunc {
	return auth.RequireLogin(func(w http.ResponseWriter, r *http.Request) {
		counts := map[string]int{"users": 0, "domains": 0, "plans": 0}
		if a.PanelDB != nil {
			var n int
			if err := a.PanelDB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n); err == nil {
				counts["users"] = n
			}
			if err := a.PanelDB.QueryRow(`SELECT COUNT(*) FROM domains`).Scan(&n); err == nil {
				counts["domains"] = n
			}
		}
		hostname, _ := os.Hostname()
		webtemplates.Render(w, r, a, "dashboard.html", webtemplates.PageData{
			Title:     "Dashboard",
			ActiveNav: "dashboard",
			User:      auth.FromContext(r),
			Counts:    counts,
			Hostname:  hostname,
			Version:   "v0.1.0",
		})
	})
}

func UsersList(a *app.App) http.HandlerFunc {
	return auth.RequireLogin(func(w http.ResponseWriter, r *http.Request) {
		webtemplates.Render(w, r, a, "users_list.html", webtemplates.PageData{
			Title:     "Users",
			ActiveNav: "users",
			User:      auth.FromContext(r),
		})
	})
}
