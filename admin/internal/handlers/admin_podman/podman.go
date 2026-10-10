package admin_podman

import (
	"net/http"

	"vietpanel/admin/internal/app"
	"vietpanel/admin/internal/auth"
	"vietpanel/admin/internal/webtemplates"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /admin/containers", auth.RequireLogin(list(a)))
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		webtemplates.Render(w, r, a, "admin_containers.html", webtemplates.PageData{
			Title: "Containers", ActiveNav: "containers", User: auth.FromContext(r),
		})
	}
}