package admin_ftp

import (
	"net/http"

	"vietpanel/admin/internal/app"
	"vietpanel/admin/internal/auth"
	"vietpanel/admin/internal/webtemplates"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /admin/ftp", auth.RequireLogin(list(a)))
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		webtemplates.Render(w, r, a, "admin_ftp.html", webtemplates.PageData{
			Title: "FTP", ActiveNav: "ftp", User: auth.FromContext(r),
		})
	}
}