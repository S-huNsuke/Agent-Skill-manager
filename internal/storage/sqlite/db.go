package sqlite

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", withForeignKeysPragma(dsn))
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	return db, nil
}

func Migrate(db *sql.DB) error {
	if err := ensureMigrationTable(db); err != nil {
		return fmt.Errorf("ensure migration table: %w", err)
	}

	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		migration, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		applied, err := migrationApplied(db, entry.Name())
		if err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if applied {
			continue
		}

		if err := applyMigration(db, entry.Name(), migration); err != nil {
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func ensureMigrationTable(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
    name TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL
);
`)
	if err != nil {
		return err
	}

	return nil
}

func migrationApplied(db *sql.DB, name string) (bool, error) {
	var appliedName string
	err := db.QueryRow(`SELECT name FROM schema_migrations WHERE name = ?`, name).Scan(&appliedName)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return appliedName == name, nil
}

func applyMigration(db *sql.DB, name string, migration []byte) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(string(migration)); err != nil {
		return err
	}

	if _, err = tx.Exec(
		`INSERT INTO schema_migrations (name, applied_at) VALUES (?, ?)`,
		name,
		time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func withForeignKeysPragma(dsn string) string {
	if strings.Contains(dsn, "_pragma=foreign_keys(1)") || strings.Contains(dsn, "_pragma=foreign_keys%281%29") {
		return dsn
	}

	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}

	return dsn + separator + "_pragma=foreign_keys(1)"
}
