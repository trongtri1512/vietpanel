// Package db manages the admin panel's SQLite store: admin accounts and,
// when configured, a read handle on the user panel's MySQL metadata.
package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Email        string
	Role         string
	IsActive     bool
	TOTPEnabled  bool
	TOTPSecret   sql.NullString
}

func Open(path string) (*sql.DB, error) {
	d, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	d.SetMaxOpenConns(10)
	if err := d.Ping(); err != nil {
		return nil, err
	}
	if err := migrate(d); err != nil {
		return nil, err
	}
	if err := seedDefaultAdmin(d); err != nil {
		return nil, err
	}
	return d, nil
}

func migrate(d *sql.DB) error {
	schema := `CREATE TABLE IF NOT EXISTS admin_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		email TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL DEFAULT 'admin',
		is_active INTEGER NOT NULL DEFAULT 1,
		totp_enabled INTEGER NOT NULL DEFAULT 0,
		totp_secret TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`
	_, err := d.Exec(schema)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// seedDefaultAdmin creates the bootstrap admin account when the table is
// empty; the installer will replace this with a proper setup flow in P7.
func seedDefaultAdmin(d *sql.DB) error {
	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM admin_users`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if _, err := d.Exec(`INSERT INTO admin_users (username, password_hash, email, role) VALUES (?, ?, ?, ?)`,
		"admin", string(hash), "admin@vietpanel.local", "admin"); err != nil {
		return err
	}
	log.Print("db: seeded default admin 'admin' / 'admin123' — change this password")
	return nil
}

func UserByUsername(d *sql.DB, username string) (*User, error) {
	u := &User{}
	err := d.QueryRow(`SELECT id, username, password_hash, email, role, is_active, totp_enabled, totp_secret FROM admin_users WHERE username = ?`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Email, &u.Role, &u.IsActive, &u.TOTPEnabled, &u.TOTPSecret)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func CheckPasswordHash(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// OpenPanelDB opens an optional read-only-ish handle on the user panel's
// MySQL metadata store for dashboard counts.
func OpenPanelDB(dsn string) (*sql.DB, error) {
	d, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	d.SetMaxOpenConns(5)
	if err := d.Ping(); err != nil {
		return nil, err
	}
	return d, nil
}
