package admin_mail

import (
	"net/http"

	"vietpanel/admin/internal/app"
	"vietpanel/admin/internal/auth"
	"vietpanel/admin/internal/webtemplates"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /admin/mail", auth.RequireLogin(list(a)))
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		webtemplates.Render(w, r, a, "admin_mail.html", webtemplates.PageData{
			Title: "Mail", ActiveNav: "mail", User: auth.FromContext(r),
		})
	}
}