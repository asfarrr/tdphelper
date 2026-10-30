package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// MovementKind says why a spool's remaining weight changed.
type MovementKind string

// Kinds of stock movements.
const (
	KindPurchase MovementKind = "purchase"
	KindConsume  MovementKind = "consume"
	KindWaste    MovementKind = "waste"
	KindPersonal MovementKind = "personal"
	KindAdjust   MovementKind = "adjust"
)

// Spool statuses.
const (
	SpoolActive = "active"
	SpoolEmpty  = "empty"
)

// Material is a plastic type with its own printing costs.
type Material struct {
	ID                int64
	Name              string
	MachineKopPerHour int64
	FailureBP         int
	Hidden            bool
}

// Filament is a product a client can choose: material, color and brand.
type Filament struct {
	ID            int64
	MaterialID    int64
	ColorName     string
	ColorHex      string
	Brand         string
	PriceKopPerKg int64
	LowStockGrams int64
	Hidden        bool
}

// Spool is a physical spool of a filament. Weights are net of the empty spool.
type Spool struct {
	ID              int64
	FilamentID      int64
	InitialGrams    int64
	EmptySpoolGrams int64
	RemainingGrams  int64
	Status          string
}

// Movement is a journal entry for a change in a spool's remaining weight.
type Movement struct {
	ID         int64
	SpoolID    int64
	DeltaGrams int64
	Kind       MovementKind
	OrderID    *int64
	Note       string
	CreatedAt  time.Time
}

// InventoryStore persists materials, filaments, spools and their journal.
// ApplyMovement and SetRemaining change the spool and write the journal atomically.
type InventoryStore interface {
	CreateMaterial(ctx context.Context, m Material) (Material, error)
	UpdateMaterial(ctx context.Context, m Material) error
	Material(ctx context.Context, id int64) (Material, error)
	ListMaterials(ctx context.Context, includeHidden bool) ([]Material, error)

	CreateFilament(ctx context.Context, f Filament) (Filament, error)
	UpdateFilament(ctx context.Context, f Filament) error
	Filament(ctx context.Context, id int64) (Filament, error)
	ListFilaments(ctx context.Context, includeHidden bool) ([]Filament, error)
	ListOrderableFilaments(ctx context.Context) ([]Filament, error)

	CreateSpool(ctx context.Context, s Spool) (Spool, error)
	Spool(ctx context.Context, id int64) (Spool, error)
	ApplyMovement(ctx context.Context, m Movement) (Movement, error)
	SetRemaining(ctx context.Context, spoolID, remainingGrams int64, note string) (Movement, error)
	Movements(ctx context.Context, spoolID int64) ([]Movement, error)
	Available(ctx context.Context, filamentID int64) (int64, error)
}

// Inventory holds the rules for filament stock.
type Inventory struct {
	store InventoryStore
}

// NewInventory returns an Inventory backed by store.
func NewInventory(store InventoryStore) *Inventory {
	return &Inventory{store: store}
}

var colorHexRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// CreateMaterial validates and stores a new material.
func (i *Inventory) CreateMaterial(ctx context.Context, m Material) (Material, error) {
	if err := m.validate(); err != nil {
		return Material{}, err
	}
	return i.store.CreateMaterial(ctx, m)
}

// UpdateMaterial validates and saves a material, including its hidden flag.
func (i *Inventory) UpdateMaterial(ctx context.Context, m Material) error {
	if err := m.validate(); err != nil {
		return err
	}
	return i.store.UpdateMaterial(ctx, m)
}

// Materials lists materials.
func (i *Inventory) Materials(ctx context.Context, includeHidden bool) ([]Material, error) {
	return i.store.ListMaterials(ctx, includeHidden)
}

// CreateFilament validates and stores a new filament of an existing material.
func (i *Inventory) CreateFilament(ctx context.Context, f Filament) (Filament, error) {
	if err := f.validate(); err != nil {
		return Filament{}, err
	}
	if _, err := i.store.Material(ctx, f.MaterialID); err != nil {
		return Filament{}, fmt.Errorf("material %d: %w", f.MaterialID, err)
	}
	return i.store.CreateFilament(ctx, f)
}

// UpdateFilament validates and saves a filament, including its hidden flag.
func (i *Inventory) UpdateFilament(ctx context.Context, f Filament) error {
	if err := f.validate(); err != nil {
		return err
	}
	if _, err := i.store.Material(ctx, f.MaterialID); err != nil {
		return fmt.Errorf("material %d: %w", f.MaterialID, err)
	}
	return i.store.UpdateFilament(ctx, f)
}

// Filaments lists filaments for the admin.
func (i *Inventory) Filaments(ctx context.Context, includeHidden bool) ([]Filament, error) {
	return i.store.ListFilaments(ctx, includeHidden)
}

// OrderableFilaments lists what clients may choose: neither the filament nor
// its material is hidden and some filament is in stock.
func (i *Inventory) OrderableFilaments(ctx context.Context) ([]Filament, error) {
	return i.store.ListOrderableFilaments(ctx)
}

// AddSpool records a purchased spool. initialGrams is net filament weight.
func (i *Inventory) AddSpool(ctx context.Context, filamentID, initialGrams, emptySpoolGrams int64) (Spool, error) {
	if initialGrams <= 0 || emptySpoolGrams < 0 {
		return Spool{}, fmt.Errorf("%w: spool weights", ErrInvalid)
	}
	if _, err := i.store.Filament(ctx, filamentID); err != nil {
		return Spool{}, fmt.Errorf("filament %d: %w", filamentID, err)
	}
	return i.store.CreateSpool(ctx, Spool{
		FilamentID:      filamentID,
		InitialGrams:    initialGrams,
		EmptySpoolGrams: emptySpoolGrams,
		RemainingGrams:  initialGrams,
		Status:          SpoolActive,
	})
}

// Consume takes grams from a spool for an order, a failed print or personal use.
func (i *Inventory) Consume(ctx context.Context, spoolID, grams int64, kind MovementKind, note string) error {
	if grams <= 0 {
		return fmt.Errorf("%w: grams must be positive", ErrInvalid)
	}
	switch kind {
	case KindConsume, KindWaste, KindPersonal:
	default:
		return fmt.Errorf("%w: kind %q", ErrInvalid, kind)
	}
	_, err := i.store.ApplyMovement(ctx, Movement{SpoolID: spoolID, DeltaGrams: -grams, Kind: kind, Note: note})
	return err
}

// AdjustByWeight sets the remaining weight from a scale reading of the whole
// spool and records the difference in the journal.
func (i *Inventory) AdjustByWeight(ctx context.Context, spoolID, weightGrams int64) (Movement, error) {
	spool, err := i.store.Spool(ctx, spoolID)
	if err != nil {
		return Movement{}, err
	}
	if weightGrams < spool.EmptySpoolGrams {
		return Movement{}, fmt.Errorf("%w: weight %d g is below the empty spool %d g",
			ErrInvalid, weightGrams, spool.EmptySpoolGrams)
	}
	return i.store.SetRemaining(ctx, spoolID, weightGrams-spool.EmptySpoolGrams, "scale reading")
}

// Available returns the grams of a filament in stock across all spools.
func (i *Inventory) Available(ctx context.Context, filamentID int64) (int64, error) {
	return i.store.Available(ctx, filamentID)
}

// Movements returns a spool's journal, oldest first.
func (i *Inventory) Movements(ctx context.Context, spoolID int64) ([]Movement, error) {
	return i.store.Movements(ctx, spoolID)
}

// Spool returns a spool.
func (i *Inventory) Spool(ctx context.Context, id int64) (Spool, error) {
	return i.store.Spool(ctx, id)
}

func (m Material) validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%w: material name is empty", ErrInvalid)
	}
	if m.MachineKopPerHour < 0 {
		return fmt.Errorf("%w: negative machine rate", ErrInvalid)
	}
	if m.FailureBP < 0 || m.FailureBP >= basisPoints {
		return fmt.Errorf("%w: failure rate must be in [0, 100%%)", ErrInvalid)
	}
	return nil
}

func (f Filament) validate() error {
	if strings.TrimSpace(f.ColorName) == "" {
		return fmt.Errorf("%w: color name is empty", ErrInvalid)
	}
	if !colorHexRe.MatchString(f.ColorHex) {
		return fmt.Errorf("%w: color must look like #RRGGBB", ErrInvalid)
	}
	if f.PriceKopPerKg < 0 || f.LowStockGrams < 0 {
		return fmt.Errorf("%w: negative price or threshold", ErrInvalid)
	}
	return nil
}
