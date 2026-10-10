// Package emails handles email account management: list accounts, create/delete,
// alias management, deliverability checks (SPF/DKIM/DMARC), and webmail links.
package emails

import (
	"net/http"
	"strings"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /emails", auth.Require(list(a)))
	mux.HandleFunc("GET /emails/new", auth.Require(createPage(a)))
	mux.HandleFunc("POST /emails/new", auth.Require(createAction(a)))
	mux.HandleFunc("POST /emails/delete", auth.Require(deleteAction(a)))
	mux.HandleFunc("GET /emails/detail", auth.Require(detailPage(a)))
	mux.HandleFunc("POST /emails/password", auth.Require(changePassword(a)))
	mux.HandleFunc("GET /emails/aliases", auth.Require(aliasesPage(a)))
	mux.HandleFunc("POST /emails/aliases/new", auth.Require(createAlias(a)))
	mux.HandleFunc("POST /emails/aliases/delete", auth.Require(deleteAlias(a)))
	mux.HandleFunc("GET /emails/deliverability", auth.Require(deliverabilityPage(a)))
	mux.HandleFunc("GET /emails/webmail", auth.Require(webmailRedirect(a)))
}

type emailAccount struct {
	ID       int
	Email    string
	Domain   string
	Quota    string
	Usage    string
	Created  string
}

type emailAlias struct {
	ID       int
	Source   string
	Dest     string
	Domain   string
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		rows, err := a.DB.Query(
			`SELECT id, email, domain, COALESCE(quota,'unlimited'), COALESCE(usage,'0'),
			 COALESCE(created_at,'') FROM email_accounts WHERE user_id = ? ORDER BY email`,
			user.ID)
		if err != nil {
			web.Render(w, r, a, "user/emails.html", web.PageData{
				Title: "Email Accounts", ActiveNav: "emails", User: user, Error: err.Error(),
			})
			return
		}
		defer rows.Close()
		var accounts []emailAccount
		for rows.Next() {
			var acct emailAccount
			_ = rows.Scan(&acct.ID, &acct.Email, &acct.Domain, &acct.Quota, &acct.Usage, &acct.Created)
			accounts = append(accounts, acct)
		}
		web.Render(w, r, a, "user/emails.html", web.PageData{
			Title: "Email Accounts", ActiveNav: "emails", User: user,
			Extra: map[string]any{"Accounts": accounts},
		})
	}
}

func createPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		// List user's domains for the domain picker
		rows, _ := a.DB.Query(`SELECT domain_url FROM domains WHERE user_id = ? ORDER BY domain_url`, user.ID)
		defer rows.Close()
		var domains []string
		for rows.Next() {
			var d string
			_ = rows.Scan(&d)
			domains = append(domains, d)
		}
		web.Render(w, r, a, "user/emails_create.html", web.PageData{
			Title: "Create Email Account", ActiveNav: "emails", User: user,
			Extra: map[string]any{"Domains": domains},
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
		localPart := strings.TrimSpace(r.FormValue("local_part"))
		domain := strings.TrimSpace(r.FormValue("domain"))
		password := strings.TrimSpace(r.FormValue("password"))
		quota := strings.TrimSpace(r.FormValue("quota"))

		if localPart == "" || domain == "" || password == "" {
			web.Render(w, r, a, "user/emails.html", web.PageData{
				Title: "Email Accounts", ActiveNav: "emails", User: user,
				Error: "All fields are required.",
			})
			return
		}
		if quota == "" {
			quota = "unlimited"
		}

		email := localPart + "@" + domain
		_, err := a.DB.Exec(
			`INSERT INTO email_accounts (email, domain, user_id, password, quota) VALUES (?, ?, ?, ?, ?)`,
			email, domain, user.ID, password, quota)
		if err != nil {
			web.Render(w, r, a, "user/emails.html", web.PageData{
				Title: "Email Accounts", ActiveNav: "emails", User: user,
				Error: "Failed to create account: " + err.Error(),
			})
			return
		}
		http.Redirect(w, r, "/emails", http.StatusFound)
	}
}

func deleteAction(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		id := r.FormValue("id")
		_, _ = a.DB.Exec(`DELETE FROM email_accounts WHERE id = ? AND user_id = ?`, id, auth.FromContext(r).ID)
		http.Redirect(w, r, "/emails", http.StatusFound)
	}
}

func detailPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		id := r.URL.Query().Get("id")
		var acct emailAccount
		err := a.DB.QueryRow(
			`SELECT id, email, domain, COALESCE(quota,'unlimited'), COALESCE(usage,'0'),
			 COALESCE(created_at,'') FROM email_accounts WHERE id = ? AND user_id = ?`,
			id, user.ID).Scan(&acct.ID, &acct.Email, &acct.Domain, &acct.Quota, &acct.Usage, &acct.Created)
		if err != nil {
			http.Redirect(w, r, "/emails", http.StatusFound)
			return
		}
		web.Render(w, r, a, "user/emails_detail.html", web.PageData{
			Title: "Email: " + acct.Email, ActiveNav: "emails", User: user,
			Extra: map[string]any{"Account": acct},
		})
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
		_, _ = a.DB.Exec(`UPDATE email_accounts SET password = ? WHERE id = ? AND user_id = ?`,
			password, id, auth.FromContext(r).ID)
		http.Redirect(w, r, "/emails", http.StatusFound)
	}
}

func aliasesPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		rows, _ := a.DB.Query(
			`SELECT id, source_email, dest_email, domain FROM email_aliases
			 WHERE user_id = ? ORDER BY source_email`, user.ID)
		defer rows.Close()
		var aliases []emailAlias
		for rows.Next() {
			var al emailAlias
			_ = rows.Scan(&al.ID, &al.Source, &al.Dest, &al.Domain)
			aliases = append(aliases, al)
		}
		web.Render(w, r, a, "user/emails_aliases.html", web.PageData{
			Title: "Email Aliases", ActiveNav: "emails", User: user,
			Extra: map[string]any{"Aliases": aliases},
		})
	}
}

func createAlias(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		user := auth.FromContext(r)
		source := r.FormValue("source")
		dest := r.FormValue("destination")
		if source == "" || dest == "" {
			http.Redirect(w, r, "/emails/aliases", http.StatusFound)
			return
		}
		domain := strings.SplitN(source, "@", 2)
		d := ""
		if len(domain) > 1 {
			d = domain[1]
		}
		_, _ = a.DB.Exec(`INSERT INTO email_aliases (source_email, dest_email, domain, user_id) VALUES (?, ?, ?, ?)`,
			source, dest, d, user.ID)
		http.Redirect(w, r, "/emails/aliases", http.StatusFound)
	}
}

func deleteAlias(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		id := r.FormValue("id")
		_, _ = a.DB.Exec(`DELETE FROM email_aliases WHERE id = ? AND user_id = ?`, id, auth.FromContext(r).ID)
		http.Redirect(w, r, "/emails/aliases", http.StatusFound)
	}
}

func deliverabilityPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		// List domains with mock deliverability status
		rows, _ := a.DB.Query(`SELECT domain_url FROM domains WHERE user_id = ? ORDER BY domain_url`, user.ID)
		defer rows.Close()
		type delivCheck struct {
			Domain string
			SPF    string
			DKIM   string
			DMARC  string
		}
		var checks []delivCheck
		for rows.Next() {
			var d string
			_ = rows.Scan(&d)
			checks = append(checks, delivCheck{
				Domain: d,
				SPF:    "Not Configured",
				DKIM:   "Not Configured",
				DMARC:  "Not Configured",
			})
		}
		web.Render(w, r, a, "user/emails_deliverability.html", web.PageData{
			Title: "Email Deliverability", ActiveNav: "emails", User: user,
			Extra: map[string]any{"Checks": checks},
		})
	}
}

func webmailRedirect(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("email")
		if email == "" {
			http.Redirect(w, r, "/emails", http.StatusFound)
			return
		}
		// In production, generate a signed URL to Roundcube/SnappyMail
		http.Redirect(w, r, "https://webmail."+strings.SplitN(email, "@", 2)[1], http.StatusFound)
	}
}