package admin_emails

import (
	"net/http"
	"vietpanel/admin/internal/app"
	"vietpanel/admin/internal/auth"
	"vietpanel/admin/internal/webtemplates"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /admin/emails", auth.RequireLogin(page(a)))
}

func page(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		webtemplates.Render(w, r, a, "admin_emails.html", webtemplates.PageData{
			Title: "emails", ActiveNav: "emails", User: auth.FromContext(r),
		})
	}
}
