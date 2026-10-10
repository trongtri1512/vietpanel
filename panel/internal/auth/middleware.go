package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/csrf"

	"vietpanel/panel/internal/app"
)

type ctxKey int

const userKey ctxKey = 1

// APIUserKey carries the JWT username on API requests.
const APIUserKey ctxKey = 2

// LoadUser attaches the session user (if any) to every request.
func LoadUser(a *app.App, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u := GetUser(r, a); u != nil {
			r = r.WithContext(context.WithValue(r.Context(), userKey, u))
		}
		next.ServeHTTP(w, r)
	})
}

func FromContext(r *http.Request) *app.User {
	u, _ := r.Context().Value(userKey).(*app.User)
	return u
}

// Require redirects anonymous users to /login, carrying the original path
// in ?next= so the login flow can return them there.
func Require(next http.HandlerFunc) http.HandlerFunc {
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

// CSRF protects all routes except /api/* which use JWT auth instead.
func CSRF(a *app.App, next http.Handler) http.Handler {
	protect := csrf.Protect(
		a.Cfg.SecretKey,
		csrf.CookieName(a.Cfg.SessionName+"_CSRF"),
		csrf.Path("/"),
		csrf.Secure(a.Cfg.SecureCookies),
	)(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
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
