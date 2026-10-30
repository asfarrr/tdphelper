package storage

import (
	"path/filepath"
	"testing"
)

func openAt(t *testing.T, path string) *DB {
	t.Helper()
	db, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestOpenCreatesDirectoryAndHandlesSpacesInPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "with space", "nested", "bot.db")

	db := openAt(t, path)

	var count int
	err := db.sql.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM materials`).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOpenTwiceKeepsDataAndAppliesMigrationsOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot.db")

	first := openAt(t, path)
	if _, err := first.sql.ExecContext(t.Context(),
		`INSERT INTO settings (key, value) VALUES ('k', 'v')`); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second := openAt(t, path)
	var value string
	if err := second.sql.QueryRowContext(t.Context(),
		`SELECT value FROM settings WHERE key = 'k'`).Scan(&value); err != nil {
		t.Fatal(err)
	}
	var migrated int
	if err := second.sql.QueryRowContext(t.Context(),
		`SELECT COUNT(*) FROM schema_migrations WHERE version = '001_initial.sql'`).Scan(&migrated); err != nil {
		t.Fatal(err)
	}
	if value != "v" || migrated != 1 {
		t.Errorf("value = %q, migrated = %d", value, migrated)
	}
}

func TestForeignKeysAreEnforced(t *testing.T) {
	db := openAt(t, filepath.Join(t.TempDir(), "bot.db"))

	_, err := db.sql.ExecContext(t.Context(), `
		INSERT INTO filaments (material_id, color_name, color_hex, brand, price_kop_per_kg, low_stock_grams)
		VALUES (999, 'Black', '#000000', 'X', 1, 0)`)
	if err == nil {
		t.Fatal("expected a foreign key error")
	}
}
