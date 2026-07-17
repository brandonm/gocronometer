package gocronometer

// Probe: understand how a water serving's Amount + MeasureID relate to the
// food's measures, so the collector can show "N bottles / X fl oz". Manual:
//
//	GOCRONOMETER_WIRE_PROBE=1 infisical run --env dev -- sh -c \
//	  'GOCRONOMETER_TEST_USERNAME=$CRONOMETER_EMAIL GOCRONOMETER_TEST_PASSWORD=$CRONOMETER_PASSWORD \
//	   go test -run TestMeasureProbe -v .'

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestMeasureProbe(t *testing.T) {
	if os.Getenv("GOCRONOMETER_WIRE_PROBE") != "1" {
		t.Skip("wire probe disabled")
	}
	u, p := os.Getenv("GOCRONOMETER_TEST_USERNAME"), os.Getenv("GOCRONOMETER_TEST_PASSWORD")
	if u == "" || p == "" {
		t.Skip("no creds")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := NewClient(nil)
	if err := c.Login(ctx, u, p); err != nil {
		t.Fatalf("login: %v", err)
	}
	defer c.Logout(ctx)

	day := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	servings, err := c.GetDayServings(ctx, day)
	if err != nil {
		t.Fatalf("servings: %v", err)
	}

	waterIDs := map[int64]bool{}
	for _, s := range servings {
		if s.MealGroup == 6 { // Water
			t.Logf("WATER serving: foodID=%d amount=%.4f measureID=%d", s.FoodID, s.Amount, s.MeasureID)
			waterIDs[s.FoodID] = true
		}
	}
	if len(waterIDs) == 0 {
		t.Log("no water servings on this day; dumping all measure info for the first few foods instead")
		for i, s := range servings {
			if i >= 3 {
				break
			}
			waterIDs[s.FoodID] = true
		}
	}

	for id := range waterIDs {
		f, err := c.GetFood(ctx, id)
		if err != nil || f == nil {
			t.Logf("food %d: resolve error: %v", id, err)
			continue
		}
		t.Logf("food %d = %q; measures:", id, f.Name)
		for _, m := range f.Measures {
			t.Logf("    measure id=%d name=%q grams=%.4f", m.ID, m.Name, m.Grams)
		}
	}
}
