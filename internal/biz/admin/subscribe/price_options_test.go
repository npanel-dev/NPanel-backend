package subscribe

import (
	"testing"

	v1 "github.com/npanel-dev/NPanel-backend/api/admin/subscribe/v1"
)

func TestConvertPriceOptionsToModelRejectsDuplicateVisibleSellableDurations(t *testing.T) {
	_, err := convertPriceOptionsToModel([]*v1.SubscribePriceOption{
		{
			Code:          "monthly_a",
			Type:          "duration",
			DurationUnit:  "Month",
			DurationValue: 1,
			Price:         500,
			Show:          true,
			Sell:          true,
			IsDefault:     true,
		},
		{
			Code:          "monthly_b",
			Type:          "duration",
			DurationUnit:  "Month",
			DurationValue: 1,
			Price:         600,
			Show:          true,
			Sell:          true,
		},
	})
	if err == nil {
		t.Fatal("expected duplicate visible sellable duration to be rejected")
	}
}

func TestConvertPriceOptionsToModelAllowsArchivedDuplicateDuration(t *testing.T) {
	options, err := convertPriceOptionsToModel([]*v1.SubscribePriceOption{
		{
			Id:            10,
			Code:          "monthly_archived",
			Type:          "duration",
			DurationUnit:  "Month",
			DurationValue: 1,
			Price:         500,
			Show:          false,
			Sell:          false,
			Version:       1,
		},
		{
			Code:          "monthly",
			Type:          "duration",
			DurationUnit:  "Month",
			DurationValue: 1,
			Price:         600,
			Show:          true,
			Sell:          true,
		},
	})
	if err != nil {
		t.Fatalf("expected archived duplicate duration to be allowed: %v", err)
	}
	if len(options) != 2 {
		t.Fatalf("len(options) = %d, want 2", len(options))
	}
	if options[0].IsDefault {
		t.Fatal("archived option must not remain default")
	}
	if !options[1].IsDefault {
		t.Fatal("first sellable visible option should become default")
	}
}
