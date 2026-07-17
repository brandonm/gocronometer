package gocronometer

// Probe: does editing a custom recipe "going forward" fork the Cronometer food
// ID, or reuse the same ID with date-dependent contents? Fetches several past
// days' diaries and resolves each serving's food ID to a name, so the same
// recipe (e.g. "Turkey Sandwich") can be tracked across a brand change. Manual:
//
//	GOCRONOMETER_WIRE_PROBE=1 infisical run --env dev -- sh -c \
//	  'GOCRONOMETER_TEST_USERNAME=$CRONOMETER_EMAIL GOCRONOMETER_TEST_PASSWORD=$CRONOMETER_PASSWORD \
//	   go test -run TestDiaryVersioningProbe -v .'

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDiaryVersioningProbe(t *testing.T) {
	if os.Getenv("GOCRONOMETER_WIRE_PROBE") != "1" {
		t.Skip("wire probe disabled; set GOCRONOMETER_WIRE_PROBE=1")
	}
	username := os.Getenv("GOCRONOMETER_TEST_USERNAME")
	password := os.Getenv("GOCRONOMETER_TEST_PASSWORD")
	if username == "" || password == "" {
		t.Skip("credentials not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	c := NewClient(nil)
	if err := c.Login(ctx, username, password); err != nil {
		t.Fatalf("login: %v", err)
	}
	defer c.Logout(ctx)

	dates := []string{"2026-05-15", "2026-06-15", "2026-07-01", "2026-07-10", "2026-07-14", "2026-07-16"}
	nameCache := map[int64]string{}
	resolve := func(id int64) string {
		if n, ok := nameCache[id]; ok {
			return n
		}
		f, err := c.GetFood(ctx, id)
		n := "?"
		if err == nil && f != nil {
			n = f.Name
		} else if err != nil {
			n = "ERR:" + err.Error()
		}
		nameCache[id] = n
		return n
	}

	for _, ds := range dates {
		d, _ := time.Parse("2006-01-02", ds)
		servings, err := c.GetDayServings(ctx, d)
		if err != nil {
			t.Logf("%s: getDayInfo error: %v", ds, err)
			continue
		}
		t.Logf("=== %s: %d servings ===", ds, len(servings))
		for _, s := range servings {
			name := resolve(s.FoodID)
			// Highlight anything that looks like the turkey sandwich the user edits.
			t.Logf("   meal=%d foodID=%d amount=%.2f  %s", s.MealGroup, s.FoodID, s.Amount, name)
		}
	}
}
