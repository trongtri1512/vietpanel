// Command license-server is VietPanel's self-hosted license validation + management service.
//
// Machine API (what the panel calls):
//
//	POST /enterprise/index.php   form licensekey+ip -> XML <status>Active|Invalid</status>
//
// Management API (Bearer LICENSE_ADMIN_TOKEN):
//
//	POST /api/keys   {"plan":"enterprise","ip":"","note":""} -> created key
//	GET  /api/keys   -> all keys
//
// Web UI (session cookie login with LICENSE_ADMIN_TOKEN):
//
//	GET  /           dashboard (key list + issue/revoke forms)
//	POST /login      password = LICENSE_ADMIN_TOKEN
package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------- key store

type Key struct {
	Key       string    `json:"key"`
	Plan      string    `json:"plan"` // enterprise | noc | lifetime
	IP        string    `json:"ip,omitempty"`
	Status    string    `json:"status"` // Active | Revoked
	CreatedAt time.Time `json:"created_at"`
	Note      string    `json:"note,omitempty"`
}

type Store struct {
	mu   sync.RWMutex
	path string
	keys map[string]*Key
}

func LoadStore(path string) (*Store, error) {
	s := &Store{path: path, keys: map[string]*Key{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, s.save()
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, &s.keys); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return s, nil
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.keys, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) Get(key string) (*Key, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k, ok := s.keys[key]
	return k, ok
}

func (s *Store) Put(k *Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[k.Key] = k
	return s.save()
}

func (s *Store) All() []*Key {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Key, 0, len(s.keys))
	for _, k := range s.keys {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// ------------------------------------------------------------ validation API

// licenseResponse mirrors the tag/value XML the panel's parseLicenseResponse understands.
type licenseResponse struct {
	Status string `xml:"status"`
	Plan   string `xml:"plan,omitempty"`
}

func writeXML(w http.ResponseWriter, status, plan string) {
	w.Header().Set("Content-Type", "application/xml")
	body, _ := xml.Marshal(licenseResponse{Status: status, Plan: plan})
	fmt.Fprint(w, xml.Header+string(body))
}

func validateHandler(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			writeXML(w, "Invalid", "")
			return
		}
		licenseKey := strings.TrimSpace(r.Form.Get("licensekey"))
		ip := strings.TrimSpace(r.Form.Get("ip"))

		k, ok := store.Get(licenseKey)
		if !ok || k.Status != "Active" {
			log.Printf("LICENSE - deny unknown/revoked key=%q ip=%q", licenseKey, ip)
			writeXML(w, "Invalid", "")
			return
		}
		// bind key to first-seen IP; subsequent mismatches are denied
		if k.IP == "" {
			k.IP = ip
			_ = store.Put(k)
		} else if ip != "" && k.IP != ip {
			log.Printf("LICENSE - deny ip mismatch key=%q bound=%q got=%q", licenseKey, k.IP, ip)
			writeXML(w, "Invalid", "")
			return
		}
		log.Printf("LICENSE - allow key=%q plan=%s ip=%q", licenseKey, k.Plan, ip)
		writeXML(w, "Active", k.Plan)
	}
}

// --------------------------------------------------------------- management

type issueRequest struct {
	Plan string `json:"plan"`
	IP   string `json:"ip,omitempty"`
	Note string `json:"note,omitempty"`
}

func newKey(req issueRequest) *Key {
	plan := strings.ToLower(strings.TrimSpace(req.Plan))
	if plan == "" {
		plan = "enterprise"
	}
	return &Key{
		Key:       fmt.Sprintf("%s-%s", plan, randomToken(16)),
		Plan:      plan,
		IP:        strings.TrimSpace(req.IP),
		Status:    "Active",
		CreatedAt: time.Now().UTC(),
		Note:      req.Note,
	}
}

func bearerOK(r *http.Request, token string) bool {
	return token != "" && r.Header.Get("Authorization") == "Bearer "+token
}

func apiIssue(store *Store, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !bearerOK(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var req issueRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		k := newKey(req)
		if err := store.Put(k); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(k)
	}
}

func apiList(store *Store, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !bearerOK(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(store.All())
	}
}

// ------------------------------------------------------------- web UI auth

func cookieValue(token string, expiry time.Time) string {
	mac := hmac.New(sha256.New, []byte(token))
	fmt.Fprintf(mac, "%d", expiry.Unix())
	return strconv.FormatInt(expiry.Unix(), 10) + "." + hex.EncodeToString(mac.Sum(nil))
}

func cookieValid(cookie, token string) bool {
	parts := strings.SplitN(cookie, ".", 2)
	if len(parts) != 2 || token == "" {
		return false
	}
	expiry, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Unix(expiry, 0).Before(time.Now()) {
		return false
	}
	return hmac.Equal([]byte(cookieValue(token, time.Unix(expiry, 0))), []byte(cookie))
}

const sessionCookie = "vietpanel_license_session"

func requireWeb(token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil || !cookieValid(c.Value, token) {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

// ------------------------------------------------------------------ web UI

//go:embed all:templates
var templateFS embed.FS

var pages = template.Must(template.ParseFS(templateFS, "templates/*.html"))

type dashData struct {
	Keys        []*Key
	Active      int
	ValidateURL string
}

func render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("UI - render %s: %v", name, err)
	}
}

func webLogin(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		errMsg := ""
		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			if token != "" && r.Form.Get("token") == token {
				expiry := time.Now().Add(12 * time.Hour)
				http.SetCookie(w, &http.Cookie{
					Name:     sessionCookie,
					Value:    cookieValue(token, expiry),
					Path:     "/",
					Expires:  expiry,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			errMsg = "Invalid token."
		}
		render(w, "login.html", map[string]any{"Error": errMsg})
	}
}

func webLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func webDashboard(store *Store, validateURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keys := store.All()
		active := 0
		for _, k := range keys {
			if k.Status == "Active" {
				active++
			}
		}
		render(w, "keys.html", dashData{Keys: keys, Active: active, ValidateURL: validateURL})
	}
}

func webIssue(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		k := newKey(issueRequest{
			Plan: r.Form.Get("plan"),
			IP:   r.Form.Get("ip"),
			Note: r.Form.Get("note"),
		})
		_ = store.Put(k)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func webRevoke(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if k, ok := store.Get(r.Form.Get("key")); ok {
			k.Status = "Revoked"
			_ = store.Put(k)
		}
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// -------------------------------------------------------------------- main

func main() {
	dbPath := envOr("LICENSE_DB", "licenses.json")
	addr := envOr("LICENSE_ADDR", ":2095")
	adminToken := os.Getenv("LICENSE_ADMIN_TOKEN")
	// ValidateURL is shown in the UI so admins can copy it into the panel config.
	validateURL := envOr("LICENSE_PUBLIC_URL", "http://127.0.0.1:2095") + "/enterprise/index.php"

	store, err := LoadStore(dbPath)
	if err != nil {
		log.Fatalf("license-server: %v", err)
	}
	if adminToken == "" {
		log.Print("license-server: WARNING - LICENSE_ADMIN_TOKEN not set; management UI/API disabled")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") })
	mux.HandleFunc("POST /enterprise/index.php", validateHandler(store))
	mux.HandleFunc("POST /api/keys", apiIssue(store, adminToken))
	mux.HandleFunc("GET /api/keys", apiList(store, adminToken))

	mux.HandleFunc("GET /{$}", requireWeb(adminToken, webDashboard(store, validateURL)))
	mux.HandleFunc("/login", webLogin(adminToken))
	mux.HandleFunc("POST /logout", webLogout())
	mux.HandleFunc("POST /keys/issue", requireWeb(adminToken, webIssue(store)))
	mux.HandleFunc("POST /keys/revoke", requireWeb(adminToken, webRevoke(store)))

	log.Printf("license-server listening on %s (db=%s, ui=http://localhost%s/)", addr, dbPath, addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("randomToken: %v", err)
	}
	return hex.EncodeToString(b)
}
