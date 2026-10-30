package service

import (
	"errors"
	"testing"
)

func designExample() QuoteInput {
	return QuoteInput{
		Lines:          []MaterialLine{{Grams: 120, PriceKopPerKg: 200000}},
		PrintMinutes:   360,
		LaborMinutes:   20,
		MachineKopPerH: 3000,
		LaborKopPerH:   60000,
		FailureBP:      1000,
		MarkupBP:       3000,
		MinKop:         30000,
		RoundToKop:     5000,
	}
}

func TestCalculateQuoteDesignExample(t *testing.T) {
	got, err := CalculateQuote(designExample())
	if err != nil {
		t.Fatal(err)
	}
	want := Quote{
		MaterialKop: 24000,
		MachineKop:  18000,
		LaborKop:    20000,
		BufferKop:   4667,
		CostKop:     66667,
		PriceKop:    90000,
		TotalKop:    90000,
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCalculateQuoteVariants(t *testing.T) {
	tests := []struct {
		name         string
		change       func(*QuoteInput)
		wantMaterial int64
		wantPrice    int64
		wantTotal    int64
	}{
		{
			name: "several materials are summed",
			change: func(in *QuoteInput) {
				in.Lines = append(in.Lines, MaterialLine{Grams: 30, PriceKopPerKg: 400000})
			},
			wantMaterial: 36000,
			wantPrice:    105000,
			wantTotal:    105000,
		},
		{
			name: "minimum order applies",
			change: func(in *QuoteInput) {
				in.MinKop = 200000
			},
			wantMaterial: 24000,
			wantPrice:    200000,
			wantTotal:    200000,
		},
		{
			name: "price already on the rounding step stays",
			change: func(in *QuoteInput) {
				in.Lines = []MaterialLine{{Grams: 100, PriceKopPerKg: 100000}}
				in.PrintMinutes, in.LaborMinutes, in.FailureBP, in.MarkupBP = 0, 0, 0, 0
				in.MinKop = 0
			},
			wantMaterial: 10000,
			wantPrice:    10000,
			wantTotal:    10000,
		},
		{
			name: "no rounding when step is zero",
			change: func(in *QuoteInput) {
				in.RoundToKop = 0
			},
			wantMaterial: 24000,
			wantPrice:    86667,
			wantTotal:    86667,
		},
		{
			name: "delivery is added after markup and rounding",
			change: func(in *QuoteInput) {
				in.DeliveryKop = 33300
			},
			wantMaterial: 24000,
			wantPrice:    90000,
			wantTotal:    123300,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := designExample()
			tt.change(&in)
			got, err := CalculateQuote(in)
			if err != nil {
				t.Fatal(err)
			}
			if got.MaterialKop != tt.wantMaterial || got.PriceKop != tt.wantPrice || got.TotalKop != tt.wantTotal {
				t.Errorf("got %+v, want material=%d price=%d total=%d",
					got, tt.wantMaterial, tt.wantPrice, tt.wantTotal)
			}
		})
	}
}

func TestCalculateQuoteRoundsIntermediateHalfUp(t *testing.T) {
	// 1 g at 5 ₽/kg is exactly half a kopeck, 1 minute at 1.50 ₽/h is 2.5 kopecks.
	got, err := CalculateQuote(QuoteInput{
		Lines:          []MaterialLine{{Grams: 1, PriceKopPerKg: 500}},
		PrintMinutes:   1,
		MachineKopPerH: 150,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.MaterialKop != 1 || got.MachineKop != 3 {
		t.Errorf("got material=%d machine=%d, want 1 and 3", got.MaterialKop, got.MachineKop)
	}
}

func TestCalculateQuoteRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name   string
		change func(*QuoteInput)
	}{
		{"no lines", func(in *QuoteInput) { in.Lines = nil }},
		{"zero grams", func(in *QuoteInput) { in.Lines[0].Grams = 0 }},
		{"negative price per kg", func(in *QuoteInput) { in.Lines[0].PriceKopPerKg = -1 }},
		{"negative print minutes", func(in *QuoteInput) { in.PrintMinutes = -1 }},
		{"negative labor minutes", func(in *QuoteInput) { in.LaborMinutes = -1 }},
		{"negative machine rate", func(in *QuoteInput) { in.MachineKopPerH = -1 }},
		{"negative labor rate", func(in *QuoteInput) { in.LaborKopPerH = -1 }},
		{"negative failure", func(in *QuoteInput) { in.FailureBP = -1 }},
		{"failure of 100 percent", func(in *QuoteInput) { in.FailureBP = 10000 }},
		{"negative markup", func(in *QuoteInput) { in.MarkupBP = -1 }},
		{"negative minimum", func(in *QuoteInput) { in.MinKop = -1 }},
		{"negative rounding", func(in *QuoteInput) { in.RoundToKop = -1 }},
		{"negative delivery", func(in *QuoteInput) { in.DeliveryKop = -1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := designExample()
			tt.change(&in)
			if _, err := CalculateQuote(in); !errors.Is(err, ErrInvalid) {
				t.Errorf("err = %v, want ErrInvalid", err)
			}
		})
	}
}
