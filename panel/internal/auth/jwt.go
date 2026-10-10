package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"vietpanel/panel/internal/app"
	core "vietpanel/panel/internal/core/db"
)

// APILogin issues a JWT for API access; no session cookie involved.
func APILogin(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		id, name, _, err := core.VerifyUser(a.DB, req.Username, req.Password)
		if err != nil {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		claims := jwt.MapClaims{
			"uid":      id,
			"username": name,
			"exp":      time.Now().Add(24 * time.Hour).Unix(),
		}
		tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.Cfg.SecretKey)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"token": tok})
	}
}

// RequireAPI guards /api/* routes with a Bearer JWT and stashes the username
// in the request context.
func RequireAPI(a *app.App, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(strings.TrimPrefix(h, "Bearer "), claims, func(t *jwt.Token) (any, error) {
			return a.Cfg.SecretKey, nil
		})
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		username, _ := claims["username"].(string)
		r = r.WithContext(context.WithValue(r.Context(), APIUserKey, username))
		next(w, r)
	}
}
