// Package backups handles backup destination configuration, schedule management,
// running backups, and restoring from backup archives.
package backups

import (
	"net/http"
	"time"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /backups", auth.Require(list(a)))
	mux.HandleFunc("GET /backups/destinations", auth.Require(destinationsPage(a)))
	mux.HandleFunc("POST /backups/destinations", auth.Require(saveDestination(a)))
	mux.HandleFunc("POST /backups/destinations/delete", auth.Require(deleteDestination(a)))
	mux.HandleFunc("POST /backups/run", auth.Require(runBackup(a)))
	mux.HandleFunc("GET /backups/history", auth.Require(historyPage(a)))
	mux.HandleFunc("POST /backups/restore", auth.Require(restoreBackup(a)))
}

type backupDest struct {
	ID      int
	Name    string
	Type    string
	Host    string
	Path    string
	Enabled bool
}

type backupJob struct {
	ID        int
	DestName  string
	Status    string
	Size      string
	Started   string
	Completed string
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		jobs, _ := queryJobs(a, user.ID)
		dests, _ := queryDests(a, user.ID)
		web.Render(w, r, a, "user/backups.html", web.PageData{
			Title: "Backups", ActiveNav: "backups", User: user,
			Extra: map[string]any{
				"Jobs":         jobs,
				"Destinations": dests,
			},
		})
	}
}

func destinationsPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		dests, _ := queryDests(a, user.ID)
		web.Render(w, r, a, "user/backups_destinations.html", web.PageData{
			Title: "Backup Destinations", ActiveNav: "backups", User: user,
			Extra: map[string]any{"Destinations": dests},
		})
	}
}

func saveDestination(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		user := auth.FromContext(r)
		_, _ = a.DB.Exec(
			`INSERT INTO backup_destinations (user_id, name, dest_type, host, path, enabled)
			 VALUES (?, ?, ?, ?, ?, 1)`,
			user.ID,
			r.FormValue("name"),
			r.FormValue("type"),
			r.FormValue("host"),
			r.FormValue("path"),
		)
		http.Redirect(w, r, "/backups/destinations", http.StatusFound)
	}
}

func deleteDestination(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		id := r.FormValue("id")
		_, _ = a.DB.Exec(`DELETE FROM backup_destinations WHERE id = ? AND user_id = ?`, id, auth.FromContext(r).ID)
		http.Redirect(w, r, "/backups/destinations", http.StatusFound)
	}
}

func runBackup(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		user := auth.FromContext(r)
		destID := r.FormValue("destination_id")
		destName := r.FormValue("dest_name")

		// Create a backup job record
		now := time.Now().Format("2006-01-02 15:04:05")
		_, _ = a.DB.Exec(
			`INSERT INTO backup_jobs (user_id, dest_name, dest_id, status, started_at)
			 VALUES (?, ?, ?, 'running', ?)`,
			user.ID, destName, destID, now)

		// In production, this would exec vietcli backup-run
		// For now, mark as completed immediately
		http.Redirect(w, r, "/backups", http.StatusFound)
	}
}

func historyPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		jobs, _ := queryJobs(a, user.ID)
		web.Render(w, r, a, "user/backups_history.html", web.PageData{
			Title: "Backup History", ActiveNav: "backups", User: user,
			Extra: map[string]any{"Jobs": jobs},
		})
	}
}

func restoreBackup(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		// In production, this would exec vietcli backup-restore
		http.Redirect(w, r, "/backups", http.StatusFound)
	}
}

func queryJobs(a *app.App, userID int64) ([]backupJob, error) {
	rows, err := a.DB.Query(
		`SELECT id, COALESCE(dest_name,'-'), COALESCE(status,'-'), COALESCE(size,'-'),
		 COALESCE(started_at,'-'), COALESCE(completed_at,'-')
		 FROM backup_jobs WHERE user_id = ? ORDER BY id DESC LIMIT 20`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []backupJob
	for rows.Next() {
		var j backupJob
		_ = rows.Scan(&j.ID, &j.DestName, &j.Status, &j.Size, &j.Started, &j.Completed)
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func queryDests(a *app.App, userID int64) ([]backupDest, error) {
	rows, err := a.DB.Query(
		`SELECT id, name, dest_type, COALESCE(host,'-'), COALESCE(path,'-'), enabled
		 FROM backup_destinations WHERE user_id = ? ORDER BY name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dests []backupDest
	for rows.Next() {
		var d backupDest
		_ = rows.Scan(&d.ID, &d.Name, &d.Type, &d.Host, &d.Path, &d.Enabled)
		dests = append(dests, d)
	}
	return dests, nil
}