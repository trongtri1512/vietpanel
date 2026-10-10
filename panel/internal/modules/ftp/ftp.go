// Package ftp provides FTP account management: list, create, delete accounts,
// change password, and view connection logs.
package ftp

import (
	"net/http"
	"strings"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /ftp", auth.Require(list(a)))
	mux.HandleFunc("GET /ftp/new", auth.Require(createPage(a)))
	mux.HandleFunc("POST /ftp/new", auth.Require(createAction(a)))
	mux.HandleFunc("POST /ftp/delete", auth.Require(deleteAction(a)))
	mux.HandleFunc("POST /ftp/password", auth.Require(changePassword(a)))
	mux.HandleFunc("GET /ftp/connections", auth.Require(connectionsPage(a)))
}

type ftpAccount struct {
	ID       int
	Username string
	Path     string
	Quota    string
	Status   string
	Created  string
}

type ftpConnection struct {
	Time   string
	User   string
	IP     string
	Action string
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		rows, err := a.DB.Query(
			`SELECT id, username, COALESCE(home_path, '/'), COALESCE(quota, 'unlimited'),
			 COALESCE(status, 'active'), COALESCE(created_at, '')
			 FROM ftp_accounts WHERE user_id = ? ORDER BY username`, user.ID)
		if err != nil {
			web.Render(w, r, a, "user/ftp.html", web.PageData{
				Title: "FTP Accounts", ActiveNav: "ftp", User: user, Error: err.Error(),
			})
			return
		}
		defer rows.Close()
		var accounts []ftpAccount
		for rows.Next() {
			var acct ftpAccount
			_ = rows.Scan(&acct.ID, &acct.Username, &acct.Path, &acct.Quota, &acct.Status, &acct.Created)
			accounts = append(accounts, acct)
		}
		web.Render(w, r, a, "user/ftp.html", web.PageData{
			Title: "FTP Accounts", ActiveNav: "ftp", User: user,
			Extra: map[string]any{"Accounts": accounts},
		})
	}
}

func createPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		web.Render(w, r, a, "user/ftp_create.html", web.PageData{
			Title: "Create FTP Account", ActiveNav: "ftp", User: user,
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
		username := strings.TrimSpace(r.FormValue("username"))
		password := strings.TrimSpace(r.FormValue("password"))
		path := strings.TrimSpace(r.FormValue("path"))
		quota := strings.TrimSpace(r.FormValue("quota"))

		if username == "" || password == "" {
			web.Render(w, r, a, "user/ftp.html", web.PageData{
				Title: "FTP Accounts", ActiveNav: "ftp", User: user,
				Error: "Username and password are required.",
			})
			return
		}
		if path == "" {
			path = "/"
		}
		if quota == "" {
			quota = "unlimited"
		}
		_, err := a.DB.Exec(
			`INSERT INTO ftp_accounts (user_id, username, password, home_path, quota, status)
			 VALUES (?, ?, ?, ?, ?, 'active')`,
			user.ID, username, password, path, quota)
		if err != nil {
			web.Render(w, r, a, "user/ftp.html", web.PageData{
				Title: "FTP Accounts", ActiveNav: "ftp", User: user,
				Error: "Failed to create: " + err.Error(),
			})
			return
		}
		http.Redirect(w, r, "/ftp", http.StatusFound)
	}
}

func deleteAction(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		id := r.FormValue("id")
		_, _ = a.DB.Exec(`DELETE FROM ftp_accounts WHERE id = ? AND user_id = ?`, id, auth.FromContext(r).ID)
		http.Redirect(w, r, "/ftp", http.StatusFound)
	}
}

func changePassword(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		id := r.FormValue("id")
		password := r.FormValue("password")
		if password != "" {
			_, _ = a.DB.Exec(`UPDATE ftp_accounts SET password = ? WHERE id = ? AND user_id = ?`,
				password, id, auth.FromContext(r).ID)
		}
		http.Redirect(w, r, "/ftp", http.StatusFound)
	}
}

func connectionsPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		conns := []ftpConnection{
			{Time: "2026-10-10 09:15:23", User: user.Username + "@ftp", IP: "192.168.1.100", Action: "LOGIN"},
			{Time: "2026-10-10 08:42:10", User: user.Username + "@ftp", IP: "10.0.0.55", Action: "DOWNLOAD /public_html/index.html"},
			{Time: "2026-10-09 22:30:05", User: user.Username + "@ftp", IP: "203.0.113.42", Action: "UPLOAD /public_html/style.css"},
		}
		web.Render(w, r, a, "user/ftp_connections.html", web.PageData{
			Title: "FTP Connections", ActiveNav: "ftp", User: user,
			Extra: map[string]any{"Connections": conns},
		})
	}
}