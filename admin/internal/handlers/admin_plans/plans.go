package admin_plans

import (
	"net/http"
	"vietpanel/admin/internal/app"
	"vietpanel/admin/internal/auth"
	"vietpanel/admin/internal/webtemplates"
)

type planRow struct {
	ID           int
	Name         string
	Domains      int
	Websites     int
	DBLimit      int
	DiskLimit    string
	CPU          string
	RAM          string
	Bandwidth    int
	FeatureSet   string
}

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /admin/plans", auth.RequireLogin(list(a)))
	mux.HandleFunc("GET /admin/plans/new", auth.RequireLogin(newPage(a)))
	mux.HandleFunc("POST /admin/plans/new", auth.RequireLogin(createAction(a)))
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, _ := a.DB.Query("SELECT id, name, domains_limit, websites_limit, db_limit, disk_limit, cpu, ram, bandwidth, feature_set FROM plans ORDER BY id")
		defer rows.Close()
		var plans []planRow
		for rows.Next() {
			var p planRow
			rows.Scan(&p.ID, &p.Name, &p.Domains, &p.Websites, &p.DBLimit, &p.DiskLimit, &p.CPU, &p.RAM, &p.Bandwidth, &p.FeatureSet)
			plans = append(plans, p)
		}
		webtemplates.Render(w, r, a, "admin_plans.html", webtemplates.PageData{
			Title: "Plans", ActiveNav: "plans", User: auth.FromContext(r),
			Extra: map[string]any{"Plans": plans},
		})
	}
}

func newPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		webtemplates.Render(w, r, a, "admin_plans_create.html", webtemplates.PageData{
			Title: "Create Plan", ActiveNav: "plans", User: auth.FromContext(r),
		})
	}
}

func createAction(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		name := r.FormValue("name")
		if name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		a.DB.Exec(`INSERT INTO plans (name, domains_limit, websites_limit, db_limit, disk_limit, cpu, ram, bandwidth, feature_set) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			name, r.FormValue("domains"), r.FormValue("websites"), r.FormValue("db_limit"), r.FormValue("disk_limit"),
			r.FormValue("cpu"), r.FormValue("ram"), r.FormValue("bandwidth"), r.FormValue("feature_set"))
		http.Redirect(w, r, "/admin/plans", http.StatusFound)
	}
}
