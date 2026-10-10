package admin_caddy

import (
	"net/http"

	"vietpanel/admin/internal/app"
	"vietpanel/admin/internal/auth"
	"vietpanel/admin/internal/webtemplates"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /admin/caddy", auth.RequireLogin(list(a)))
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		webtemplates.Render(w, r, a, "admin_caddy.html", webtemplates.PageData{
			Title: "Caddy", ActiveNav: "caddy", User: auth.FromContext(r),
		})
	}
}