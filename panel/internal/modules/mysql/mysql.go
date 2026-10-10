// Package mysql handles database management: list, create, and delete MySQL
// databases for the logged-in user.
package mysql

import (
	"net/http"
	"strings"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /databases", auth.Require(list(a)))
	mux.HandleFunc("GET /databases/new", auth.Require(createPage(a)))
	mux.HandleFunc("POST /databases/new", auth.Require(createAction(a)))
	mux.HandleFunc("GET /databases/detail", auth.Require(detail(a)))
	mux.HandleFunc("POST /databases/delete", auth.Require(deleteAction(a)))
}

type dbRow struct {
	ID     int
	Name   string
	User   string
	Host   string
	Port   string
	Engine string
	Size   string
}

func (d dbRow) ConnectionString() string {
	return d.User + ":" + d.Name + "@" + d.Host + ":" + d.Port + "/" + d.Name
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := a.DB.Query(`SELECT id, db_name, db_user FROM databases WHERE user_id = ? ORDER BY db_name`,
			auth.FromContext(r).ID)
		if err != nil {
			web.Render(w, r, a, "user/databases.html", web.PageData{
				Title: "Databases", ActiveNav: "databases", Error: err.Error(),
			})
			return
		}
		defer rows.Close()
		var dbs []dbRow
		for rows.Next() {
			var d dbRow
			if err := rows.Scan(&d.ID, &d.Name, &d.User); err != nil {
				continue
			}
			d.Host = "127.0.0.1"
			d.Port = "3306"
			d.Engine = "MySQL"
			d.Size = "-"
			dbs = append(dbs, d)
		}
		web.Render(w, r, a, "user/databases.html", web.PageData{
			Title: "Databases", ActiveNav: "databases", User: auth.FromContext(r),
			Extra: map[string]any{"Databases": dbs},
		})
	}
}

func createPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		web.Render(w, r, a, "user/databases_create.html", web.PageData{
			Title: "New Database", ActiveNav: "databases", User: auth.FromContext(r),
		})
	}
}

func createAction(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		user := auth.FromContext(r)
		dbName := strings.TrimSpace(r.FormValue("db_name"))
		dbUser := strings.TrimSpace(r.FormValue("db_user"))
		if dbName == "" || dbUser == "" {
			web.Render(w, r, a, "user/databases_create.html", web.PageData{
				Title: "New Database", ActiveNav: "databases", User: user,
				Error: "Database name and user are required.",
			})
			return
		}
		_, err := a.DB.Exec(`INSERT INTO databases (db_name, db_user, user_id) VALUES (?, ?, ?)`,
			dbName, dbUser, user.ID)
		if err != nil {
			web.Render(w, r, a, "user/databases_create.html", web.PageData{
				Title: "New Database", ActiveNav: "databases", User: user,
				Error: "Failed to create database: " + err.Error(),
			})
			return
		}
		http.Redirect(w, r, "/databases", http.StatusFound)
	}
}

func detail(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Redirect(w, r, "/databases", http.StatusFound)
			return
		}
		var d dbRow
		err := a.DB.QueryRow(`SELECT id, db_name, db_user FROM databases WHERE id = ? AND user_id = ?`,
			id, auth.FromContext(r).ID).Scan(&d.ID, &d.Name, &d.User)
		if err != nil {
			http.Redirect(w, r, "/databases", http.StatusFound)
			return
		}
		d.Host = "127.0.0.1"
		d.Port = "3306"
		d.Engine = "MySQL"
		d.Size = "-"
		web.Render(w, r, a, "user/databases_detail.html", web.PageData{
			Title: "DB: " + d.Name, ActiveNav: "databases", User: auth.FromContext(r),
			Extra: map[string]any{"DB": d},
		})
	}
}

func deleteAction(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		id := r.FormValue("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		_, _ = a.DB.Exec(`DELETE FROM databases WHERE id = ? AND user_id = ?`, id, auth.FromContext(r).ID)
		http.Redirect(w, r, "/databases", http.StatusFound)
	}
}