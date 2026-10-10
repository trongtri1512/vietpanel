// Package account implements login, logout and API login for the user panel.
package account

import (
	"net/http"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /login", auth.LoginPage(a))
	mux.HandleFunc("POST /login", auth.LoginPost(a))
	mux.HandleFunc("POST /logout", auth.LogoutPost(a))
	mux.HandleFunc("POST /api/login", auth.APILogin(a))
}
