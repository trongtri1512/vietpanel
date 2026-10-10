// Package auth handles admin sessions, flashes, CSRF and the per-IP login
// rate limiter, mirroring the original admin panel's flows.
package auth

import (
	"net/http"

	"github.com/gorilla/sessions"

	"vietpanel/admin/internal/db"
)

// Manager wraps the session store with the admin session cookie.
type Manager struct {
	store  sessions.Store
	name   string
	secure bool
}

func NewManager(name string, secret []byte, secure bool) *Manager {
	return &Manager{store: sessions.NewCookieStore(secret), name: name, secure: secure}
}

func (m *Manager) Get(r *http.Request) (*sessions.Session, error) {
	return m.store.Get(r, m.name)
}

// LoginUser stores the admin identity in the session cookie (31 days, like
// the original).
func (m *Manager) LoginUser(w http.ResponseWriter, r *http.Request, u *db.User) error {
	s, err := m.Get(r)
	if err != nil {
		return err
	}
	s.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		MaxAge:   31 * 86400,
		SameSite: http.SameSiteLaxMode,
		Secure:   m.secure,
	}
	s.Values["uid"] = u.ID
	s.Values["username"] = u.Username
	s.Values["role"] = u.Role
	return s.Save(r, w)
}

func (m *Manager) LogoutUser(w http.ResponseWriter, r *http.Request) {
	s, err := m.Get(r)
	if err != nil {
		return
	}
	s.Options.MaxAge = -1
	_ = s.Save(r, w)
}

func (m *Manager) CurrentUser(r *http.Request) *db.User {
	s, err := m.Get(r)
	if err != nil {
		return nil
	}
	uid, ok := s.Values["uid"].(int64)
	if !ok {
		return nil
	}
	username, _ := s.Values["username"].(string)
	role, _ := s.Values["role"].(string)
	return &db.User{ID: uid, Username: username, Role: role}
}

// Flash is a one-shot message shown on the next rendered page.
type Flash struct {
	Type    string // "danger" | "success"
	Message string
}

func AddFlash(w http.ResponseWriter, r *http.Request, m *Manager, message, typ string) {
	s, err := m.Get(r)
	if err != nil {
		return
	}
	s.AddFlash(Flash{Type: typ, Message: message}, "flashes")
	_ = s.Save(r, w)
}

func PopFlashes(w http.ResponseWriter, r *http.Request, m *Manager) []Flash {
	s, err := m.Get(r)
	if err != nil {
		return nil
	}
	raw := s.Flashes("flashes")
	if len(raw) == 0 {
		return nil
	}
	out := make([]Flash, 0, len(raw))
	for _, f := range raw {
		if fl, ok := f.(Flash); ok {
			out = append(out, fl)
		}
	}
	delete(s.Values, "flashes")
	_ = s.Save(r, w)
	return out
}
