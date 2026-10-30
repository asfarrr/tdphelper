package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"tdphelper/internal/service"
)

const (
	materialColumns = `id, name, machine_kop_per_hour, failure_bp, hidden`
	filamentColumns = `id, material_id, color_name, color_hex, brand, price_kop_per_kg, low_stock_grams, hidden`
	spoolColumns    = `id, filament_id, initial_grams, empty_spool_grams, remaining_grams, status`
	movementColumns = `id, spool_id, delta_grams, kind, order_id, note, created_at`
)

type scanner interface {
	Scan(dest ...any) error
}

func scanMaterial(row scanner) (service.Material, error) {
	var m service.Material
	err := row.Scan(&m.ID, &m.Name, &m.MachineKopPerHour, &m.FailureBP, &m.Hidden)
	return m, err
}

func scanFilament(row scanner) (service.Filament, error) {
	var f service.Filament
	err := row.Scan(&f.ID, &f.MaterialID, &f.ColorName, &f.ColorHex, &f.Brand,
		&f.PriceKopPerKg, &f.LowStockGrams, &f.Hidden)
	return f, err
}

func scanSpool(row scanner) (service.Spool, error) {
	var s service.Spool
	err := row.Scan(&s.ID, &s.FilamentID, &s.InitialGrams, &s.EmptySpoolGrams, &s.RemainingGrams, &s.Status)
	return s, err
}

func scanMovement(row scanner) (service.Movement, error) {
	var (
		m       service.Movement
		orderID sql.NullInt64
		created string
	)
	if err := row.Scan(&m.ID, &m.SpoolID, &m.DeltaGrams, &m.Kind, &orderID, &m.Note, &created); err != nil {
		return service.Movement{}, err
	}
	if orderID.Valid {
		m.OrderID = &orderID.Int64
	}
	t, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return service.Movement{}, fmt.Errorf("movement %d created_at: %w", m.ID, err)
	}
	m.CreatedAt = t
	return m, nil
}

func queryAll[T any](ctx context.Context, d *DB, scan func(scanner) (T, error), query string, args ...any) ([]T, error) {
	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []T
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrNotFound
	}
	return err
}

func requireUpdated(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return service.ErrNotFound
	}
	return nil
}

// CreateMaterial stores a material and returns it with its ID.
func (d *DB) CreateMaterial(ctx context.Context, m service.Material) (service.Material, error) {
	err := d.sql.QueryRowContext(ctx, `
		INSERT INTO materials (name, machine_kop_per_hour, failure_bp, hidden)
		VALUES (?, ?, ?, ?) RETURNING id`,
		m.Name, m.MachineKopPerHour, m.FailureBP, m.Hidden).Scan(&m.ID)
	return m, err
}

// UpdateMaterial saves a material.
func (d *DB) UpdateMaterial(ctx context.Context, m service.Material) error {
	res, err := d.sql.ExecContext(ctx, `
		UPDATE materials SET name = ?, machine_kop_per_hour = ?, failure_bp = ?, hidden = ?
		WHERE id = ?`,
		m.Name, m.MachineKopPerHour, m.FailureBP, m.Hidden, m.ID)
	return requireUpdated(res, err)
}

// Material returns a material or service.ErrNotFound.
func (d *DB) Material(ctx context.Context, id int64) (service.Material, error) {
	m, err := scanMaterial(d.sql.QueryRowContext(ctx,
		`SELECT `+materialColumns+` FROM materials WHERE id = ?`, id))
	return m, notFound(err)
}

// ListMaterials lists materials by name.
func (d *DB) ListMaterials(ctx context.Context, includeHidden bool) ([]service.Material, error) {
	return queryAll(ctx, d, scanMaterial,
		`SELECT `+materialColumns+` FROM materials WHERE hidden = 0 OR ? ORDER BY name`, includeHidden)
}

// CreateFilament stores a filament and returns it with its ID.
func (d *DB) CreateFilament(ctx context.Context, f service.Filament) (service.Filament, error) {
	err := d.sql.QueryRowContext(ctx, `
		INSERT INTO filaments (material_id, color_name, color_hex, brand, price_kop_per_kg, low_stock_grams, hidden)
		VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id`,
		f.MaterialID, f.ColorName, f.ColorHex, f.Brand, f.PriceKopPerKg, f.LowStockGrams, f.Hidden).Scan(&f.ID)
	return f, err
}

// UpdateFilament saves a filament.
func (d *DB) UpdateFilament(ctx context.Context, f service.Filament) error {
	res, err := d.sql.ExecContext(ctx, `
		UPDATE filaments SET material_id = ?, color_name = ?, color_hex = ?, brand = ?,
			price_kop_per_kg = ?, low_stock_grams = ?, hidden = ?
		WHERE id = ?`,
		f.MaterialID, f.ColorName, f.ColorHex, f.Brand, f.PriceKopPerKg, f.LowStockGrams, f.Hidden, f.ID)
	return requireUpdated(res, err)
}

// Filament returns a filament or service.ErrNotFound.
func (d *DB) Filament(ctx context.Context, id int64) (service.Filament, error) {
	f, err := scanFilament(d.sql.QueryRowContext(ctx,
		`SELECT `+filamentColumns+` FROM filaments WHERE id = ?`, id))
	return f, notFound(err)
}

// ListFilaments lists filaments by ID.
func (d *DB) ListFilaments(ctx context.Context, includeHidden bool) ([]service.Filament, error) {
	return queryAll(ctx, d, scanFilament,
		`SELECT `+filamentColumns+` FROM filaments WHERE hidden = 0 OR ? ORDER BY id`, includeHidden)
}

// ListOrderableFilaments lists visible filaments of visible materials with stock.
func (d *DB) ListOrderableFilaments(ctx context.Context) ([]service.Filament, error) {
	return queryAll(ctx, d, scanFilament, `
		SELECT `+filamentColumns+` FROM filaments
		WHERE hidden = 0
		  AND material_id IN (SELECT id FROM materials WHERE hidden = 0)
		  AND id IN (SELECT filament_id FROM spools WHERE remaining_grams > 0)
		ORDER BY id`)
}

// CreateSpool stores a spool and its purchase movement.
func (d *DB) CreateSpool(ctx context.Context, s service.Spool) (service.Spool, error) {
	err := d.inTx(ctx, func(tx *sql.Tx) error {
		err := tx.QueryRowContext(ctx, `
			INSERT INTO spools (filament_id, initial_grams, empty_spool_grams, remaining_grams, status)
			VALUES (?, ?, ?, ?, ?) RETURNING id`,
			s.FilamentID, s.InitialGrams, s.EmptySpoolGrams, s.RemainingGrams, s.Status).Scan(&s.ID)
		if err != nil {
			return err
		}
		_, err = insertMovement(ctx, tx, service.Movement{
			SpoolID:    s.ID,
			DeltaGrams: s.RemainingGrams,
			Kind:       service.KindPurchase,
			Note:       "new spool",
		})
		return err
	})
	return s, err
}

// Spool returns a spool or service.ErrNotFound.
func (d *DB) Spool(ctx context.Context, id int64) (service.Spool, error) {
	s, err := scanSpool(d.sql.QueryRowContext(ctx,
		`SELECT `+spoolColumns+` FROM spools WHERE id = ?`, id))
	return s, notFound(err)
}

// ApplyMovement changes a spool's remaining weight and writes the journal. It
// returns service.ErrInsufficientStock if the spool would go below zero.
func (d *DB) ApplyMovement(ctx context.Context, m service.Movement) (service.Movement, error) {
	var saved service.Movement
	err := d.inTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, updateRemainingSQL, m.DeltaGrams, m.DeltaGrams, m.SpoolID, m.DeltaGrams)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			var exists int
			err := tx.QueryRowContext(ctx, `SELECT 1 FROM spools WHERE id = ?`, m.SpoolID).Scan(&exists)
			if err != nil {
				return notFound(err)
			}
			return service.ErrInsufficientStock
		}
		saved, err = insertMovement(ctx, tx, m)
		return err
	})
	return saved, err
}

// SetRemaining sets a spool's remaining weight and journals the difference.
func (d *DB) SetRemaining(ctx context.Context, spoolID, remainingGrams int64, note string) (service.Movement, error) {
	var saved service.Movement
	err := d.inTx(ctx, func(tx *sql.Tx) error {
		var current int64
		err := tx.QueryRowContext(ctx, `SELECT remaining_grams FROM spools WHERE id = ?`, spoolID).Scan(&current)
		if err != nil {
			return notFound(err)
		}
		delta := remainingGrams - current
		if _, err := tx.ExecContext(ctx, updateRemainingSQL, delta, delta, spoolID, delta); err != nil {
			return err
		}
		saved, err = insertMovement(ctx, tx, service.Movement{
			SpoolID:    spoolID,
			DeltaGrams: delta,
			Kind:       service.KindAdjust,
			Note:       note,
		})
		return err
	})
	return saved, err
}

const updateRemainingSQL = `
	UPDATE spools
	SET remaining_grams = remaining_grams + ?1,
	    status = CASE WHEN remaining_grams + ?2 = 0 THEN 'empty' ELSE 'active' END
	WHERE id = ?3 AND remaining_grams + ?4 >= 0`

func insertMovement(ctx context.Context, tx *sql.Tx, m service.Movement) (service.Movement, error) {
	m.CreatedAt = time.Now().UTC()
	err := tx.QueryRowContext(ctx, `
		INSERT INTO stock_movements (spool_id, delta_grams, kind, order_id, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?) RETURNING id`,
		m.SpoolID, m.DeltaGrams, m.Kind, m.OrderID, m.Note, m.CreatedAt.Format(time.RFC3339Nano)).Scan(&m.ID)
	return m, err
}

// Movements returns a spool's journal, oldest first.
func (d *DB) Movements(ctx context.Context, spoolID int64) ([]service.Movement, error) {
	return queryAll(ctx, d, scanMovement,
		`SELECT `+movementColumns+` FROM stock_movements WHERE spool_id = ? ORDER BY id`, spoolID)
}

// Available sums the remaining grams of a filament across its spools.
func (d *DB) Available(ctx context.Context, filamentID int64) (int64, error) {
	var total int64
	err := d.sql.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(remaining_grams), 0) FROM spools WHERE filament_id = ?`, filamentID).Scan(&total)
	return total, err
}

var _ service.InventoryStore = (*DB)(nil)
