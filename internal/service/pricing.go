package service

import "fmt"

const (
	basisPoints   = 10000
	minutesInHour = 60
	gramsInKg     = 1000
)

// MaterialLine is the amount of one filament used by an order.
type MaterialLine struct {
	Grams         int64
	PriceKopPerKg int64
}

// QuoteInput holds everything needed to price an order. Money is in kopecks,
// rates FailureBP and MarkupBP are in basis points (1000 = 10%).
type QuoteInput struct {
	Lines          []MaterialLine
	PrintMinutes   int
	LaborMinutes   int
	MachineKopPerH int64
	LaborKopPerH   int64
	FailureBP      int
	MarkupBP       int
	MinKop         int64
	RoundToKop     int64
	DeliveryKop    int64
}

// Quote is a price breakdown in kopecks. PriceKop is the recommended price and
// TotalKop adds delivery on top of it.
type Quote struct {
	MaterialKop int64
	MachineKop  int64
	LaborKop    int64
	BufferKop   int64
	CostKop     int64
	PriceKop    int64
	TotalKop    int64
}

// CalculateQuote computes the recommended price. Every intermediate amount is
// rounded half up to a whole kopeck; the final price is rounded up to RoundToKop.
func CalculateQuote(in QuoteInput) (Quote, error) {
	if err := in.validate(); err != nil {
		return Quote{}, err
	}

	var q Quote
	for _, line := range in.Lines {
		q.MaterialKop += divRound(line.Grams*line.PriceKopPerKg, gramsInKg)
	}
	q.MachineKop = divRound(int64(in.PrintMinutes)*in.MachineKopPerH, minutesInHour)
	q.LaborKop = divRound(int64(in.LaborMinutes)*in.LaborKopPerH, minutesInHour)
	// Labor stays out of the buffer: a failed print rarely repeats all of the manual work.
	q.BufferKop = divRound((q.MaterialKop+q.MachineKop)*int64(in.FailureBP), basisPoints-int64(in.FailureBP))
	q.CostKop = q.MaterialKop + q.MachineKop + q.LaborKop + q.BufferKop

	price := divRound(q.CostKop*(basisPoints+int64(in.MarkupBP)), basisPoints)
	q.PriceKop = ceilTo(max(price, in.MinKop), in.RoundToKop)
	q.TotalKop = q.PriceKop + in.DeliveryKop
	return q, nil
}

func (in QuoteInput) validate() error {
	if len(in.Lines) == 0 {
		return fmt.Errorf("%w: no material lines", ErrInvalid)
	}
	for i, line := range in.Lines {
		if line.Grams <= 0 || line.PriceKopPerKg < 0 {
			return fmt.Errorf("%w: material line %d", ErrInvalid, i)
		}
	}
	if in.PrintMinutes < 0 || in.LaborMinutes < 0 {
		return fmt.Errorf("%w: negative minutes", ErrInvalid)
	}
	if in.MachineKopPerH < 0 || in.LaborKopPerH < 0 {
		return fmt.Errorf("%w: negative rate", ErrInvalid)
	}
	if in.FailureBP < 0 || in.FailureBP >= basisPoints {
		return fmt.Errorf("%w: failure rate must be in [0, 100%%)", ErrInvalid)
	}
	if in.MarkupBP < 0 {
		return fmt.Errorf("%w: negative markup", ErrInvalid)
	}
	if in.MinKop < 0 || in.RoundToKop < 0 || in.DeliveryKop < 0 {
		return fmt.Errorf("%w: negative amount", ErrInvalid)
	}
	return nil
}

// divRound divides non-negative a by positive b, rounding half up.
func divRound(a, b int64) int64 {
	return (a + b/2) / b
}

// ceilTo rounds v up to a multiple of step; a zero step leaves v unchanged.
func ceilTo(v, step int64) int64 {
	if step == 0 {
		return v
	}
	return (v + step - 1) / step * step
}
