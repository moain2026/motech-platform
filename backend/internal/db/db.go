// Package db handles the PostgreSQL connection and migration runner.
package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Connect opens a sqlx DB using a standard PostgreSQL DATABASE_URL and verifies it.
func Connect(databaseURL string) (*sqlx.DB, error) {
	conn, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	conn.SetMaxOpenConns(20)
	conn.SetMaxIdleConns(5)
	return conn, nil
}

// MigrateDir runs every *.sql file in dir, sorted by filename. Each migration
// is written with IF NOT EXISTS, so running them repeatedly is safe.
func MigrateDir(conn *sqlx.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, n := range names {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", n, err)
		}
		if _, err := conn.Exec(string(b)); err != nil {
			return fmt.Errorf("apply migration %s: %w", n, err)
		}
	}
	return nil
}
