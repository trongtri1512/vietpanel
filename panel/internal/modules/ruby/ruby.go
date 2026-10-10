// Package ruby installs a Ruby application into a domain via a docker-compose service + reverse proxy - the actual logic lives in internal/modules/appinstall, shared with the nodejs and python packages
package ruby

import (
	"net/http"

	appctx "vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/modules/appinstall"
)

// Register wires the /ruby/install route onto mux.
func Register(mux *http.ServeMux, a *appctx.App) {
	requireLogin := func(h http.HandlerFunc) http.Handler {
		return auth.RequireLogin(a, "ruby")(h)
	}
	mux.Handle("GET /ruby/install", requireLogin(func(w http.ResponseWriter, r *http.Request) {
		appinstall.HandleInstallPage(appinstall.Ruby, a, w, r)
	}))
	mux.Handle("POST /ruby/install", requireLogin(func(w http.ResponseWriter, r *http.Request) {
		appinstall.HandleInstallPage(appinstall.Ruby, a, w, r)
	}))
}
