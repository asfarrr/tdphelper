// Package storage implements persistence on SQLite.
package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"slices"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

//go:embed migrations/*.sql
var migrations embed.FS

// DB is the application database.
type DB struct {
	sql *sql.DB
}

// Open opens the database at path, creating it and its directory if needed,
// and applies pending migrations.
func Open(ctx context.Context, path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	dsn := "file:" + (&url.URL{Path: path}).EscapedPath() +
		"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// One connection serializes access, so a transaction never hits SQLITE_BUSY
	// when it upgrades from read to write. Load is far below what this limits.
	conn.SetMaxOpenConns(1)

	db := &DB{sql: conn}
	if err := db.migrate(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return db, nil
}

// Close releases the database.
func (d *DB) Close() error {
	return d.sql.Close()
}

func (d *DB) migrate(ctx context.Context) error {
	if _, err := d.sql.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	slices.Sort(names)

	for _, name := range names {
		if err := d.applyMigration(ctx, name); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
	}
	return nil
}

func (d *DB) applyMigration(ctx context.Context, name string) error {
	return d.inTx(ctx, func(tx *sql.Tx) error {
		var applied int
		err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, name).Scan(&applied)
		if err != nil || applied > 0 {
			return err
		}
		body, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, name)
		return err
	})
}

func (d *DB) inTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
