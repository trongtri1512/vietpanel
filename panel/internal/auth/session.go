// Package auth handles sessions, CSRF, login/logout and JWT API access for
// the user panel.
package auth

import (
	"net/http"

	"github.com/gorilla/sessions"

	"vietpanel/panel/internal/app"
)

func session(r *http.Request, a *app.App) (*sessions.Session, error) {
	return a.Sessions.Get(r, a.Cfg.SessionName)
}

func SetSessionUser(w http.ResponseWriter, r *http.Request, a *app.App, u *app.User) error {
	s, err := session(r, a)
	if err != nil {
		return err
	}
	s.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		MaxAge:   7 * 86400,
		SameSite: http.SameSiteLaxMode,
		Secure:   a.Cfg.SecureCookies,
	}
	s.Values["uid"] = u.ID
	s.Values["username"] = u.Username
	return s.Save(r, w)
}

func ClearSession(w http.ResponseWriter, r *http.Request, a *app.App) {
	s, err := session(r, a)
	if err != nil {
		return
	}
	s.Options.MaxAge = -1
	_ = s.Save(r, w)
}

func GetUser(r *http.Request, a *app.App) *app.User {
	s, err := session(r, a)
	if err != nil {
		return nil
	}
	uid, ok := s.Values["uid"].(int64)
	if !ok {
		return nil
	}
	username, _ := s.Values["username"].(string)
	return &app.User{ID: uid, Username: username}
}
