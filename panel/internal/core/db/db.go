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
			owner VARCHAR(255) NOT NULL DEFAULT '',
			plan VARCHAR(255) NOT NULL DEFAULT '',
			plan_id INTEGER NOT NULL DEFAULT 0,
			server VARCHAR(64) NOT NULL DEFAULT 'active',
			registered_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_login_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS plans (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			domains_limit INTEGER NOT NULL DEFAULT 0,
			websites_limit INTEGER NOT NULL DEFAULT 0,
			email_limit INTEGER NOT NULL DEFAULT 0,
			ftp_limit INTEGER NOT NULL DEFAULT 0,
			disk_limit TEXT NOT NULL DEFAULT '0',
			inodes_limit BIGINT NOT NULL DEFAULT 0,
			db_limit INTEGER NOT NULL DEFAULT 0,
			cpu VARCHAR(50) DEFAULT '1',
			ram VARCHAR(50) DEFAULT '1G',
			bandwidth INTEGER DEFAULT 0,
			feature_set VARCHAR(255) DEFAULT 'default',
			max_email_quota TEXT NOT NULL DEFAULT '0',
			max_hourly_email INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS domains (
			domain_id INTEGER PRIMARY KEY ` + autoInc + `,
			docroot VARCHAR(255) NOT NULL DEFAULT '/var/www/html',
			domain_url VARCHAR(255) NOT NULL,
			user_id INTEGER DEFAULT NULL,
			php_version VARCHAR(255) DEFAULT '8.2'
		)`,
		`CREATE TABLE IF NOT EXISTS sites (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			site_name VARCHAR(255) NOT NULL UNIQUE,
			domain_id INTEGER NOT NULL,
			admin_email VARCHAR(255),
			version VARCHAR(32),
			created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
			type VARCHAR(50) DEFAULT '',
			container VARCHAR(255) DEFAULT '',
			ports VARCHAR(255) DEFAULT '',
			path VARCHAR(255) DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS databases (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			db_name VARCHAR(64) NOT NULL,
			db_user VARCHAR(64) NOT NULL,
			user_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS activity_log (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			username VARCHAR(64) NOT NULL,
			action VARCHAR(255) NOT NULL,
			ip VARCHAR(64) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS email_accounts (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			email VARCHAR(255) NOT NULL,
			domain VARCHAR(255) NOT NULL,
			user_id INTEGER NOT NULL,
			password VARCHAR(255) NOT NULL DEFAULT '',
			quota VARCHAR(32) NOT NULL DEFAULT 'unlimited',
			usage VARCHAR(32) NOT NULL DEFAULT '0',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS email_aliases (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			source_email VARCHAR(255) NOT NULL,
			dest_email VARCHAR(255) NOT NULL,
			domain VARCHAR(255) NOT NULL DEFAULT '',
			user_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS backup_destinations (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			user_id INTEGER NOT NULL,
			name VARCHAR(255) NOT NULL,
			dest_type VARCHAR(32) NOT NULL DEFAULT 's3',
			host VARCHAR(255) NOT NULL DEFAULT '',
			path VARCHAR(255) NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS backup_jobs (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			user_id INTEGER NOT NULL,
			dest_id INTEGER NOT NULL DEFAULT 0,
			dest_name VARCHAR(255) NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			size VARCHAR(32) NOT NULL DEFAULT '',
			started_at DATETIME,
			completed_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS dns_records (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			domain_id INTEGER NOT NULL,
			record_type VARCHAR(10) NOT NULL,
			name VARCHAR(255) NOT NULL DEFAULT '',
			value VARCHAR(512) NOT NULL,
			ttl INTEGER NOT NULL DEFAULT 3600,
			priority INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS waf_settings (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			domain_id INTEGER NOT NULL UNIQUE,
			enabled INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS ip_blocks (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			user_id INTEGER NOT NULL,
			ip_address VARCHAR(64) NOT NULL,
			reason VARCHAR(255) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS php_extensions (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			user_id INTEGER NOT NULL,
			ext_name VARCHAR(64) NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS ftp_accounts (
			id INTEGER PRIMARY KEY ` + autoInc + `,
			user_id INTEGER NOT NULL,
			username VARCHAR(64) NOT NULL,
			password VARCHAR(255) NOT NULL DEFAULT '',
			home_path VARCHAR(255) NOT NULL DEFAULT '/',
			quota VARCHAR(32) NOT NULL DEFAULT 'unlimited',
			status VARCHAR(16) NOT NULL DEFAULT 'active',
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

// seedDemoUser creates a dev login + default plan when tables are empty.
func seedDemoUser(d *sql.DB) error {
	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM plans`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		if _, err := d.Exec(`INSERT INTO plans (id, name, domains_limit, websites_limit, disk_limit, db_limit, cpu, ram)
			VALUES (1, 'Standard plan', 5, 10, '5 GB', 10, '2', '2G')`); err != nil {
			return err
		}
		log.Print("db: seeded default plan 'Standard plan'")
	}

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
	if _, err := d.Exec(`INSERT INTO users (username, password, email, plan, plan_id) VALUES (?, ?, ?, ?, ?)`,
		"demo", string(hash), "demo@vietpanel.local", "Standard plan", 1); err != nil {
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
