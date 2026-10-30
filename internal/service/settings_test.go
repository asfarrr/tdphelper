package service_test

import (
	"errors"
	"testing"

	"tdphelper/internal/service"
)

func TestSettingsDefaultsAndSave(t *testing.T) {
	mgr := service.NewSettingsManager(newDB(t))
	ctx := t.Context()

	got, err := mgr.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != service.DefaultSettings() {
		t.Errorf("first load = %+v, want defaults", got)
	}

	want := service.Settings{LaborKopPerHour: 1, MarkupBP: 2, MinOrderKop: 3, RoundToKop: 0}
	if err := mgr.Save(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, err = mgr.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("after save = %+v, want %+v", got, want)
	}

	bad := want
	bad.MarkupBP = -1
	if err := mgr.Save(ctx, bad); !errors.Is(err, service.ErrInvalid) {
		t.Errorf("negative value: err = %v, want ErrInvalid", err)
	}
}

func TestBuildQuoteInputTakesHighestRates(t *testing.T) {
	in := service.BuildQuoteInput(service.DefaultSettings(), []service.QuoteLine{
		{Material: service.Material{MachineKopPerHour: 3000, FailureBP: 2000}, Filament: service.Filament{PriceKopPerKg: 1}, Grams: 10},
		{Material: service.Material{MachineKopPerHour: 9000, FailureBP: 500}, Filament: service.Filament{PriceKopPerKg: 2}, Grams: 20},
	}, 60, 30, 100)

	if in.MachineKopPerH != 9000 || in.FailureBP != 2000 {
		t.Errorf("rates = %d, %d", in.MachineKopPerH, in.FailureBP)
	}
	if len(in.Lines) != 2 || in.Lines[1].Grams != 20 || in.Lines[1].PriceKopPerKg != 2 {
		t.Errorf("lines = %+v", in.Lines)
	}
	if in.PrintMinutes != 60 || in.LaborMinutes != 30 || in.DeliveryKop != 100 {
		t.Errorf("input = %+v", in)
	}
}

// The design document's worked example, end to end through the database.
func TestDesignExampleEndToEnd(t *testing.T) {
	db := newDB(t)
	ctx := t.Context()
	inv := service.NewInventory(db)
	settings, err := service.NewSettingsManager(db).Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	f := newFilament(t, inv)
	materials, err := inv.Materials(ctx, false)
	if err != nil {
		t.Fatal(err)
	}

	in := service.BuildQuoteInput(settings, []service.QuoteLine{
		{Material: materials[0], Filament: f, Grams: 120},
	}, 360, 20, 0)
	q, err := service.CalculateQuote(in)
	if err != nil {
		t.Fatal(err)
	}
	if q.CostKop != 66667 || q.PriceKop != 90000 {
		t.Errorf("quote = %+v, want cost 66667 and price 90000", q)
	}
}
