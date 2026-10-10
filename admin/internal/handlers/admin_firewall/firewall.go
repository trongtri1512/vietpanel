package admin_firewall

import (
	"net/http"
	"vietpanel/admin/internal/app"
	"vietpanel/admin/internal/auth"
	"vietpanel/admin/internal/webtemplates"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /admin/firewall", auth.RequireLogin(page(a)))
}

func page(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		webtemplates.Render(w, r, a, "admin_firewall.html", webtemplates.PageData{
			Title: "firewall", ActiveNav: "firewall", User: auth.FromContext(r),
		})
	}
}
