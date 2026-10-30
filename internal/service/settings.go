package service

import (
	"context"
	"fmt"
	"strconv"
)

const (
	keyLaborKopPerHour = "labor_kop_per_hour"
	keyMarkupBP        = "markup_bp"
	keyMinOrderKop     = "min_order_kop"
	keyRoundToKop      = "round_to_kop"
)

// Settings are the master's global pricing parameters.
type Settings struct {
	LaborKopPerHour int64
	MarkupBP        int64
	MinOrderKop     int64
	RoundToKop      int64
}

// DefaultSettings returns starting values until the master sets real ones.
func DefaultSettings() Settings {
	return Settings{
		LaborKopPerHour: 60000,
		MarkupBP:        3000,
		MinOrderKop:     30000,
		RoundToKop:      5000,
	}
}

// SettingsStore persists settings as key-value pairs.
type SettingsStore interface {
	LoadSettings(ctx context.Context) (map[string]string, error)
	SaveSettings(ctx context.Context, values map[string]string) error
}

// SettingsManager reads and writes typed settings.
type SettingsManager struct {
	store SettingsStore
}

// NewSettingsManager returns a SettingsManager backed by store.
func NewSettingsManager(store SettingsStore) *SettingsManager {
	return &SettingsManager{store: store}
}

// Load returns the saved settings; anything not saved yet has its default value.
func (m *SettingsManager) Load(ctx context.Context) (Settings, error) {
	values, err := m.store.LoadSettings(ctx)
	if err != nil {
		return Settings{}, err
	}
	s := DefaultSettings()
	for key, field := range s.fields() {
		raw, ok := values[key]
		if !ok {
			continue
		}
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return Settings{}, fmt.Errorf("setting %s: %w", key, err)
		}
		*field = v
	}
	return s, nil
}

// Save validates and stores settings.
func (m *SettingsManager) Save(ctx context.Context, s Settings) error {
	values := make(map[string]string, 4)
	for key, field := range s.fields() {
		if *field < 0 {
			return fmt.Errorf("%w: %s is negative", ErrInvalid, key)
		}
		values[key] = strconv.FormatInt(*field, 10)
	}
	return m.store.SaveSettings(ctx, values)
}

func (s *Settings) fields() map[string]*int64 {
	return map[string]*int64{
		keyLaborKopPerHour: &s.LaborKopPerHour,
		keyMarkupBP:        &s.MarkupBP,
		keyMinOrderKop:     &s.MinOrderKop,
		keyRoundToKop:      &s.RoundToKop,
	}
}

// QuoteLine is the use of one filament in an order.
type QuoteLine struct {
	Material Material
	Filament Filament
	Grams    int64
}

// BuildQuoteInput prepares calculator input. With several materials it takes
// the highest machine rate and failure rate among them.
func BuildQuoteInput(s Settings, lines []QuoteLine, printMinutes, laborMinutes int, deliveryKop int64) QuoteInput {
	in := QuoteInput{
		Lines:        make([]MaterialLine, 0, len(lines)),
		PrintMinutes: printMinutes,
		LaborMinutes: laborMinutes,
		LaborKopPerH: s.LaborKopPerHour,
		MarkupBP:     int(s.MarkupBP),
		MinKop:       s.MinOrderKop,
		RoundToKop:   s.RoundToKop,
		DeliveryKop:  deliveryKop,
	}
	for _, line := range lines {
		in.Lines = append(in.Lines, MaterialLine{Grams: line.Grams, PriceKopPerKg: line.Filament.PriceKopPerKg})
		in.MachineKopPerH = max(in.MachineKopPerH, line.Material.MachineKopPerHour)
		in.FailureBP = max(in.FailureBP, line.Material.FailureBP)
	}
	return in
}
