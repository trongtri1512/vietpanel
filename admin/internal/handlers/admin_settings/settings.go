package admin_settings

import (
	"net/http"
	"vietpanel/admin/internal/app"
	"vietpanel/admin/internal/auth"
	"vietpanel/admin/internal/webtemplates"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /admin/settings", auth.RequireLogin(page(a)))
}

func page(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		webtemplates.Render(w, r, a, "admin_settings.html", webtemplates.PageData{
			Title: "Settings", ActiveNav: "settings", User: auth.FromContext(r),
		})
	}
}
