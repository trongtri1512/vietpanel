// Package dns provides DNS zone management: list zones, view/edit records,
// add/delete records (A, AAAA, CNAME, MX, TXT, NS, SRV), and bulk operations.
package dns

import (
	"fmt"
	"net/http"
	"strings"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /dns", auth.Require(listZones(a)))
	mux.HandleFunc("GET /dns/zone", auth.Require(zoneView(a)))
	mux.HandleFunc("POST /dns/record/new", auth.Require(addRecord(a)))
	mux.HandleFunc("POST /dns/record/edit", auth.Require(editRecord(a)))
	mux.HandleFunc("POST /dns/record/delete", auth.Require(deleteRecord(a)))
}

type dnsZone struct {
	ID      int
	Domain  string
	Records int
}

type dnsRecord struct {
	ID       int
	Type     string
	Name     string
	Value    string
	TTL      int
	Priority int
}

var recordTypes = []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS", "SRV", "CAA"}

func listZones(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		rows, err := a.DB.Query(
			`SELECT d.domain_id, d.domain_url, COALESCE(zc.record_count, 0)
			 FROM domains d
			 LEFT JOIN (SELECT domain_id, COUNT(*) as record_count FROM dns_records GROUP BY domain_id) zc
			   ON zc.domain_id = d.domain_id
			 WHERE d.user_id = ?
			 ORDER BY d.domain_url`, user.ID)
		if err != nil {
			web.Render(w, r, a, "user/dns.html", web.PageData{
				Title: "DNS Zones", ActiveNav: "dns", User: user, Error: err.Error(),
			})
			return
		}
		defer rows.Close()
		var zones []dnsZone
		for rows.Next() {
			var z dnsZone
			_ = rows.Scan(&z.ID, &z.Domain, &z.Records)
			zones = append(zones, z)
		}
		web.Render(w, r, a, "user/dns.html", web.PageData{
			Title: "DNS Zones", ActiveNav: "dns", User: user,
			Extra: map[string]any{"Zones": zones, "RecordTypes": recordTypes},
		})
	}
}

func zoneView(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Redirect(w, r, "/dns", http.StatusFound)
			return
		}
		var domain string
		err := a.DB.QueryRow(`SELECT domain_url FROM domains WHERE domain_id = ? AND user_id = ?`,
			id, user.ID).Scan(&domain)
		if err != nil {
			http.Redirect(w, r, "/dns", http.StatusFound)
			return
		}
		rows, _ := a.DB.Query(
			`SELECT id, record_type, name, value, COALESCE(ttl, 3600), COALESCE(priority, 0)
			 FROM dns_records WHERE domain_id = ? ORDER BY record_type, name`, id)
		defer rows.Close()
		var records []dnsRecord
		for rows.Next() {
			var rec dnsRecord
			_ = rows.Scan(&rec.ID, &rec.Type, &rec.Name, &rec.Value, &rec.TTL, &rec.Priority)
			records = append(records, rec)
		}
		web.Render(w, r, a, "user/dns_zone.html", web.PageData{
			Title: "DNS: " + domain, ActiveNav: "dns", User: user,
			Extra: map[string]any{
				"Zone":    dnsZone{ID: toInt(id), Domain: domain, Records: len(records)},
				"Records": records,
				"Types":   recordTypes,
			},
		})
	}
}

func addRecord(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		user := auth.FromContext(r)
		domainID := r.FormValue("domain_id")
		recType := strings.ToUpper(r.FormValue("type"))
		name := r.FormValue("name")
		value := r.FormValue("value")
		ttl := r.FormValue("ttl")
		priority := r.FormValue("priority")

		var ownerID int64
		err := a.DB.QueryRow(`SELECT user_id FROM domains WHERE domain_id = ?`, domainID).Scan(&ownerID)
		if err != nil || ownerID != user.ID {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if ttl == "" {
			ttl = "3600"
		}
		if priority == "" {
			priority = "0"
		}
		_, _ = a.DB.Exec(
			`INSERT INTO dns_records (domain_id, record_type, name, value, ttl, priority)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			domainID, recType, name, value, ttl, priority)
		http.Redirect(w, r, fmt.Sprintf("/dns/zone?id=%s", domainID), http.StatusFound)
	}
}

func editRecord(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		recID := r.FormValue("id")
		domainID := r.FormValue("domain_id")
		value := r.FormValue("value")
		ttl := r.FormValue("ttl")
		priority := r.FormValue("priority")
		_, _ = a.DB.Exec(`UPDATE dns_records SET value = ?, ttl = ?, priority = ? WHERE id = ?`,
			value, ttl, priority, recID)
		http.Redirect(w, r, fmt.Sprintf("/dns/zone?id=%s", domainID), http.StatusFound)
	}
}

func deleteRecord(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		recID := r.FormValue("id")
		domainID := r.FormValue("domain_id")
		_, _ = a.DB.Exec(`DELETE FROM dns_records WHERE id = ?`, recID)
		http.Redirect(w, r, fmt.Sprintf("/dns/zone?id=%s", domainID), http.StatusFound)
	}
}

func toInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}