// Command license-server is VietPanel's self-hosted license validation service.
// It speaks the same wire protocol the panel's license check expects
// (POST form licensekey+ip, XML-ish <tag>value</tag> response with <status>Active</status>),
// so the panel can validate keys against your own infrastructure instead of a vendor.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

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

// licenseResponse mirrors the tag/value XML the panel's parseLicenseResponse understands.
type licenseResponse struct {
	Status string `xml:"status"`
	Plan   string `xml:"plan,omitempty"`
	Expiry string `xml:"expiry,omitempty"`
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

type issueRequest struct {
	Plan string `json:"plan"`
	IP   string `json:"ip,omitempty"`
	Note string `json:"note,omitempty"`
}

func issueHandler(store *Store, adminToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if adminToken == "" || r.Header.Get("Authorization") != "Bearer "+adminToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var req issueRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		plan := strings.ToLower(strings.TrimSpace(req.Plan))
		if plan == "" {
			plan = "enterprise"
		}
		k := &Key{
			Key:       fmt.Sprintf("%s-%s", plan, randomToken(16)),
			Plan:      plan,
			IP:        strings.TrimSpace(req.IP),
			Status:    "Active",
			CreatedAt: time.Now().UTC(),
			Note:      req.Note,
		}
		if err := store.Put(k); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(k)
	}
}

func listHandler(store *Store, adminToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if adminToken == "" || r.Header.Get("Authorization") != "Bearer "+adminToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		store.mu.RLock()
		defer store.mu.RUnlock()
		out := make([]*Key, 0, len(store.keys))
		for _, k := range store.keys {
			out = append(out, k)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func main() {
	dbPath := envOr("LICENSE_DB", "licenses.json")
	addr := envOr("LICENSE_ADDR", ":2095")
	adminToken := os.Getenv("LICENSE_ADMIN_TOKEN")

	store, err := LoadStore(dbPath)
	if err != nil {
		log.Fatalf("license-server: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") })
	mux.HandleFunc("POST /enterprise/index.php", validateHandler(store))
	mux.HandleFunc("POST /admin/keys", issueHandler(store, adminToken))
	mux.HandleFunc("GET /admin/keys", listHandler(store, adminToken))

	log.Printf("license-server listening on %s (db=%s)", addr, dbPath)
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
