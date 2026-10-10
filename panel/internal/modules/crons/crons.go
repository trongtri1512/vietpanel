package crons

import (
	"net/http"
	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /crons", auth.Require(list(a)))
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		web.Render(w, r, a, "user/crons.html", web.PageData{
			Title: "Cron Jobs", ActiveNav: "crons", User: auth.FromContext(r),
		})
	}
}
