// Package db opens the panel metadata store (MySQL in production, SQLite for
// local development) and provides the small set of queries the auth layer needs.
package db

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"

	"vietpanel/panel/internal/app"
)

func Open(cfg *app.Config) (*sql.DB, error) {
	var (
		d   *sql.DB
		err error
	)
	if cfg.DBDSN != "" {
		d, err = sql.Open("mysql", cfg.DBDSN)
		if err != nil {
			return nil, fmt.Errorf("open mysql: %w", err)
		}
		log.Print("db: using MySQL")
	} else {
		path := filepath.Join(cfg.DataDir, "panel.db")
		d, err = sql.Open("sqlite", path)
		if err != nil {
			return nil, fmt.Errorf("open sqlite: %w", err)
		}
		log.Printf("db: using SQLite at %s", path)
	}
	d.SetMaxOpenConns(10)
	if err := d.Ping(); err != nil {
		return nil, err
	}
	if err := migrate(d, cfg.DBDSN != ""); err != nil {
		return nil, err
	}
	if err := seedDemoUser(d); err != nil {
		return nil, err
	}
	return d, nil
}

func migrate(d *sql.DB, mysql bool) error {
	autoInc := "AUTOINCREMENT"
	if mysql {
		autoInc = "AUTO_INCREMENT"
	}
	schema := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			username VARCHAR(64) NOT NULL UNIQUE,
			password VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_login_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS activity_log (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			username VARCHAR(64) NOT NULL,
			action VARCHAR(255) NOT NULL,
			ip VARCHAR(64) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, q := range schema {
		if _, err := d.Exec(q); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// seedDemoUser creates a dev login when the table is empty so the panel is
// usable out of the box; production installs get users from the admin panel.
func seedDemoUser(d *sql.DB) error {
	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("demo123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if _, err := d.Exec(`INSERT INTO users (username, password, email) VALUES (?, ?, ?)`,
		"demo", string(hash), "demo@vietpanel.local"); err != nil {
		return err
	}
	log.Print("db: seeded demo user 'demo' / 'demo123'")
	return nil
}

// VerifyUser checks credentials and returns the user identity on success.
func VerifyUser(d *sql.DB, username, password string) (int64, string, string, error) {
	var (
		id    int64
		email string
		hash  string
	)
	err := d.QueryRow(`SELECT id, email, password FROM users WHERE username = ?`, username).
		Scan(&id, &email, &hash)
	if err != nil {
		return 0, "", "", fmt.Errorf("invalid credentials")
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return 0, "", "", fmt.Errorf("invalid credentials")
	}
	_, _ = d.Exec(`UPDATE users SET last_login_at = ? WHERE id = ?`, time.Now(), id)
	return id, username, email, nil
}

func LogActivity(d *sql.DB, username, action, ip string) {
	_, _ = d.Exec(`INSERT INTO activity_log (username, action, ip) VALUES (?, ?, ?)`, username, action, ip)
}
