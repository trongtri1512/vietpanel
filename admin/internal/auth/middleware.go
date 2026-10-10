package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/csrf"

	"vietpanel/admin/internal/db"
)

type ctxKey int

const userKey ctxKey = 1

// LoadAdmin attaches the session user (if any) to every request.
func LoadAdmin(m *Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u := m.CurrentUser(r); u != nil {
			r = r.WithContext(context.WithValue(r.Context(), userKey, u))
		}
		next.ServeHTTP(w, r)
	})
}

func FromContext(r *http.Request) *db.User {
	u, _ := r.Context().Value(userKey).(*db.User)
	return u
}

// RequireLogin redirects anonymous users to /login, carrying the original
// path in ?next= (always a same-origin relative path by construction).
func RequireLogin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if FromContext(r) == nil {
			target := "/login"
			if r.URL.Path != "/" {
				target += "?next=" + url.QueryEscape(r.URL.Path)
			}
			http.Redirect(w, r, target, http.StatusFound)
			return
		}
		next(w, r)
	}
}

// CSRF protects all routes; the admin panel has no cookie-free API surface
// yet, so nothing is excluded.
func CSRF(secret []byte, sessionName string, secure bool, next http.Handler) http.Handler {
	protect := csrf.Protect(
		secret,
		csrf.CookieName(sessionName+"_CSRF"),
		csrf.Path("/"),
		csrf.Secure(secure),
	)(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// gorilla/csrf assumes HTTPS unless told otherwise, but the panel
		// usually runs plain HTTP (dev, or behind Caddy's TLS-terminating
		// proxy), so mark the request as plaintext when the client-facing
		// scheme isn't https.
		if requestScheme(r) != "https" {
			r = csrf.PlaintextHTTPRequest(r)
		}
		protect.ServeHTTP(w, r)
	})
}

// requestScheme returns the client-facing scheme, trusting X-Forwarded-Proto
// first since a TLS-terminating proxy makes r.TLS nil even over https.
func requestScheme(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return strings.ToLower(strings.TrimSpace(strings.SplitN(proto, ",", 2)[0]))
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}
