package admin_bind9

import (
	"net/http"

	"vietpanel/admin/internal/app"
	"vietpanel/admin/internal/auth"
	"vietpanel/admin/internal/webtemplates"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /admin/dns", auth.RequireLogin(list(a)))
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		webtemplates.Render(w, r, a, "admin_dns.html", webtemplates.PageData{
			Title: "DNS", ActiveNav: "dns", User: auth.FromContext(r),
		})
	}
}