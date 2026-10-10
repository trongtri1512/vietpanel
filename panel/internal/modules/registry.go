// Package modules wires feature modules onto the mux. Every module exposes a
// Register(mux, app) function; enabled_modules config filtering lands in P7.
package modules

import (
	"net/http"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/modules/account"
	"vietpanel/panel/internal/modules/dashboard"
)

func Register(mux *http.ServeMux, a *app.App) {
	account.Register(mux, a)
	dashboard.Register(mux, a)
}
