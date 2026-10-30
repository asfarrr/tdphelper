package storage

import (
	"context"
	"database/sql"

	"tdphelper/internal/service"
)

// LoadSettings returns all saved settings.
func (d *DB) LoadSettings(ctx context.Context) (map[string]string, error) {
	rows, err := d.sql.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	values := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		values[key] = value
	}
	return values, rows.Err()
}

// SaveSettings stores the given settings atomically.
func (d *DB) SaveSettings(ctx context.Context, values map[string]string) error {
	return d.inTx(ctx, func(tx *sql.Tx) error {
		for key, value := range values {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO settings (key, value) VALUES (?, ?)
				ON CONFLICT (key) DO UPDATE SET value = excluded.value`, key, value); err != nil {
				return err
			}
		}
		return nil
	})
}

var _ service.SettingsStore = (*DB)(nil)
