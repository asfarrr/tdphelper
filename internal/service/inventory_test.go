package service_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"tdphelper/internal/service"
	"tdphelper/internal/storage"
)

func newDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(t.Context(), filepath.Join(t.TempDir(), "bot.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func newFilament(t *testing.T, inv *service.Inventory) service.Filament {
	t.Helper()
	m, err := inv.CreateMaterial(t.Context(), service.Material{Name: "PLA", MachineKopPerHour: 3000, FailureBP: 1000})
	if err != nil {
		t.Fatal(err)
	}
	f, err := inv.CreateFilament(t.Context(), service.Filament{
		MaterialID:    m.ID,
		ColorName:     "Черный",
		ColorHex:      "#000000",
		Brand:         "Bambu",
		PriceKopPerKg: 200000,
	})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func newSpool(t *testing.T, inv *service.Inventory, f service.Filament, grams int64) service.Spool {
	t.Helper()
	s, err := inv.AddSpool(t.Context(), f.ID, grams, 200)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func assertJournalMatches(t *testing.T, inv *service.Inventory, spoolID int64) {
	t.Helper()
	spool, err := inv.Spool(t.Context(), spoolID)
	if err != nil {
		t.Fatal(err)
	}
	moves, err := inv.Movements(t.Context(), spoolID)
	if err != nil {
		t.Fatal(err)
	}
	var sum int64
	for _, m := range moves {
		sum += m.DeltaGrams
	}
	if sum != spool.RemainingGrams {
		t.Errorf("journal sum = %d, remaining = %d", sum, spool.RemainingGrams)
	}
}

func TestMaterialAndFilamentValidation(t *testing.T) {
	inv := service.NewInventory(newDB(t))
	ctx := t.Context()
	f := newFilament(t, inv)

	materials := map[string]service.Material{
		"blank name":    {Name: "  "},
		"negative rate": {Name: "A", MachineKopPerHour: -1},
		"failure 100%":  {Name: "A", FailureBP: 10000},
	}
	for name, m := range materials {
		t.Run(name, func(t *testing.T) {
			if _, err := inv.CreateMaterial(ctx, m); !errors.Is(err, service.ErrInvalid) {
				t.Errorf("err = %v, want ErrInvalid", err)
			}
		})
	}

	valid := service.Filament{MaterialID: f.MaterialID, ColorName: "Красный", ColorHex: "#ff0000"}
	filaments := map[string]func(*service.Filament){
		"blank color":    func(f *service.Filament) { f.ColorName = "" },
		"bad hex":        func(f *service.Filament) { f.ColorHex = "red" },
		"negative price": func(f *service.Filament) { f.PriceKopPerKg = -1 },
	}
	for name, change := range filaments {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			change(&candidate)
			if _, err := inv.CreateFilament(ctx, candidate); !errors.Is(err, service.ErrInvalid) {
				t.Errorf("err = %v, want ErrInvalid", err)
			}
		})
	}

	valid.MaterialID = 999
	if _, err := inv.CreateFilament(ctx, valid); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("unknown material: err = %v, want ErrNotFound", err)
	}
}

func TestOrderableFilaments(t *testing.T) {
	inv := service.NewInventory(newDB(t))
	ctx := t.Context()
	f := newFilament(t, inv)

	orderable := func() int {
		t.Helper()
		list, err := inv.OrderableFilaments(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return len(list)
	}

	if got := orderable(); got != 0 {
		t.Errorf("without stock: %d, want 0", got)
	}

	spool := newSpool(t, inv, f, 100)
	if got := orderable(); got != 1 {
		t.Errorf("with stock: %d, want 1", got)
	}

	f.Hidden = true
	if err := inv.UpdateFilament(ctx, f); err != nil {
		t.Fatal(err)
	}
	if got := orderable(); got != 0 {
		t.Errorf("hidden filament: %d, want 0", got)
	}
	f.Hidden = false
	if err := inv.UpdateFilament(ctx, f); err != nil {
		t.Fatal(err)
	}

	materials, err := inv.Materials(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	m := materials[0]
	m.Hidden = true
	if err := inv.UpdateMaterial(ctx, m); err != nil {
		t.Fatal(err)
	}
	if got := orderable(); got != 0 {
		t.Errorf("hidden material: %d, want 0", got)
	}
	m.Hidden = false
	if err := inv.UpdateMaterial(ctx, m); err != nil {
		t.Fatal(err)
	}

	if err := inv.Consume(ctx, spool.ID, 100, service.KindConsume, ""); err != nil {
		t.Fatal(err)
	}
	if got := orderable(); got != 0 {
		t.Errorf("empty spool: %d, want 0", got)
	}
}

func TestUpdateMissingRecords(t *testing.T) {
	inv := service.NewInventory(newDB(t))
	f := newFilament(t, inv)

	if err := inv.UpdateMaterial(t.Context(), service.Material{ID: 999, Name: "X"}); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("material: err = %v, want ErrNotFound", err)
	}
	f.ID = 999
	if err := inv.UpdateFilament(t.Context(), f); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("filament: err = %v, want ErrNotFound", err)
	}
}

func TestSpoolConsumeAndJournal(t *testing.T) {
	inv := service.NewInventory(newDB(t))
	ctx := t.Context()
	f := newFilament(t, inv)
	spool := newSpool(t, inv, f, 1000)

	for _, step := range []struct {
		grams int64
		kind  service.MovementKind
	}{
		{120, service.KindConsume},
		{30, service.KindWaste},
		{50, service.KindPersonal},
	} {
		if err := inv.Consume(ctx, spool.ID, step.grams, step.kind, "test"); err != nil {
			t.Fatal(err)
		}
	}

	got, err := inv.Spool(ctx, spool.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RemainingGrams != 800 || got.Status != service.SpoolActive {
		t.Errorf("spool = %+v", got)
	}
	moves, err := inv.Movements(ctx, spool.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(moves) != 4 || moves[0].Kind != service.KindPurchase || moves[2].Kind != service.KindWaste {
		t.Errorf("movements = %+v", moves)
	}
	if moves[1].CreatedAt.IsZero() {
		t.Error("created_at not stored")
	}
	assertJournalMatches(t, inv, spool.ID)
}

func TestConsumeRejections(t *testing.T) {
	inv := service.NewInventory(newDB(t))
	ctx := t.Context()
	f := newFilament(t, inv)
	spool := newSpool(t, inv, f, 100)

	tests := []struct {
		name    string
		spoolID int64
		grams   int64
		kind    service.MovementKind
		want    error
	}{
		{"more than remains", spool.ID, 101, service.KindConsume, service.ErrInsufficientStock},
		{"zero grams", spool.ID, 0, service.KindConsume, service.ErrInvalid},
		{"negative grams", spool.ID, -5, service.KindConsume, service.ErrInvalid},
		{"purchase kind", spool.ID, 1, service.KindPurchase, service.ErrInvalid},
		{"adjust kind", spool.ID, 1, service.KindAdjust, service.ErrInvalid},
		{"missing spool", 999, 1, service.KindConsume, service.ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := inv.Consume(ctx, tt.spoolID, tt.grams, tt.kind, "")
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}

	got, err := inv.Spool(ctx, spool.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RemainingGrams != 100 {
		t.Errorf("remaining = %d, rejected movements must not change it", got.RemainingGrams)
	}
	assertJournalMatches(t, inv, spool.ID)
}

func TestAdjustByWeight(t *testing.T) {
	inv := service.NewInventory(newDB(t))
	ctx := t.Context()
	f := newFilament(t, inv)
	spool := newSpool(t, inv, f, 1000) // empty spool weighs 200 g

	move, err := inv.AdjustByWeight(ctx, spool.ID, 900)
	if err != nil {
		t.Fatal(err)
	}
	if move.DeltaGrams != -300 || move.Kind != service.KindAdjust {
		t.Errorf("movement = %+v", move)
	}

	if _, err := inv.AdjustByWeight(ctx, spool.ID, 199); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("below empty spool: err = %v, want ErrInvalid", err)
	}

	if _, err := inv.AdjustByWeight(ctx, spool.ID, 200); err != nil {
		t.Fatal(err)
	}
	got, err := inv.Spool(ctx, spool.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RemainingGrams != 0 || got.Status != service.SpoolEmpty {
		t.Errorf("after empty reading: %+v", got)
	}

	// A wrong earlier reading can be corrected upward, reactivating the spool.
	if _, err := inv.AdjustByWeight(ctx, spool.ID, 300); err != nil {
		t.Fatal(err)
	}
	got, err = inv.Spool(ctx, spool.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RemainingGrams != 100 || got.Status != service.SpoolActive {
		t.Errorf("after correction: %+v", got)
	}
	assertJournalMatches(t, inv, spool.ID)

	if _, err := inv.AdjustByWeight(ctx, 999, 500); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("missing spool: err = %v, want ErrNotFound", err)
	}
}

func TestAvailableSumsSpools(t *testing.T) {
	inv := service.NewInventory(newDB(t))
	f := newFilament(t, inv)
	newSpool(t, inv, f, 1000)
	s2 := newSpool(t, inv, f, 500)
	if err := inv.Consume(t.Context(), s2.ID, 100, service.KindConsume, ""); err != nil {
		t.Fatal(err)
	}

	got, err := inv.Available(t.Context(), f.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1400 {
		t.Errorf("available = %d, want 1400", got)
	}
}

func TestAddSpoolValidation(t *testing.T) {
	inv := service.NewInventory(newDB(t))
	f := newFilament(t, inv)
	ctx := t.Context()

	if _, err := inv.AddSpool(ctx, f.ID, 0, 200); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("zero weight: err = %v", err)
	}
	if _, err := inv.AddSpool(ctx, f.ID, 100, -1); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("negative empty spool: err = %v", err)
	}
	if _, err := inv.AddSpool(ctx, 999, 100, 200); !errors.Is(err, service.ErrNotFound) {
		t.Errorf("missing filament: err = %v", err)
	}
}

func TestConcurrentConsumeNeverOverdraws(t *testing.T) {
	inv := service.NewInventory(newDB(t))
	f := newFilament(t, inv)
	spool := newSpool(t, inv, f, 100)
	ctx := context.Background()

	var succeeded atomic.Int64
	var wg sync.WaitGroup
	for range 30 {
		wg.Go(func() {
			err := inv.Consume(ctx, spool.ID, 10, service.KindConsume, "")
			switch {
			case err == nil:
				succeeded.Add(1)
			case !errors.Is(err, service.ErrInsufficientStock):
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
	wg.Wait()

	got, err := inv.Spool(t.Context(), spool.ID)
	if err != nil {
		t.Fatal(err)
	}
	if succeeded.Load() != 10 || got.RemainingGrams != 0 {
		t.Errorf("succeeded = %d, remaining = %d", succeeded.Load(), got.RemainingGrams)
	}
	assertJournalMatches(t, inv, spool.ID)
}
