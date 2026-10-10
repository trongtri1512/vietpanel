// Package dashboard implements the panel's landing page and the minimal API
// surface (whoami + endpoint registry) that P1 ships.
package dashboard

import (
	"encoding/json"
	"net/http"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	})
	mux.HandleFunc("GET /dashboard", auth.Require(page(a)))
	mux.HandleFunc("GET /api/whoami", auth.RequireAPI(a, whoami(a)))
	mux.HandleFunc("GET /api/endpoints", endpoints(a))
}

func page(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		web.Render(w, r, a, "user/dashboard.html", web.PageData{
			Title:     "Dashboard",
			ActiveNav: "dashboard",
			User:      auth.FromContext(r),
			Counts:    map[string]int{"websites": 0, "domains": 0, "databases": 0},
		})
	}
}

func whoami(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name, _ := r.Context().Value(auth.APIUserKey).(string)
		_ = json.NewEncoder(w).Encode(map[string]string{"username": name})
	}
}

func endpoints(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]string{
			{"method": "POST", "path": "/api/login", "description": "Get a JWT for API access"},
			{"method": "GET", "path": "/api/whoami", "description": "Current user info (JWT)"},
		})
	}
}
