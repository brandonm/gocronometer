package gocronometer

import (
	"os"
	"testing"
	"time"
)

// TestParseDayInfoServings_LocalFixture iterates the diary deserializer against a
// real captured getDayInfo response. The fixture contains personal health data so
// it is gitignored (testdata/dayinfo-local.txt) — the test skips if it's absent.
// This is the harness used to verify/refine the Serving field mapping.
func TestParseDayInfoServings_LocalFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/dayinfo-local.txt")
	if err != nil {
		t.Skip("no local fixture (testdata/dayinfo-local.txt) — skipping")
	}

	day := time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC)
	const userID = "15983721"
	servings, err := parseDayInfoServings(string(body), day, userID)
	if err != nil {
		t.Fatalf("parseDayInfoServings: %v", err)
	}

	t.Logf("decoded %d servings", len(servings))
	for i, s := range servings {
		t.Logf("  serving[%d]: foodID=%d amount=%.4f date=%s", i, s.FoodID, s.Amount, s.Date.Format("2006-01-02"))
	}

	if len(servings) == 0 {
		t.Fatal("expected servings from the populated fixture")
	}
	// Verified live: foodID 69796849 == "Mocha (20oz)", amount 1.0, Breakfast (meal 1).
	var sawMocha, sawWater bool
	for _, s := range servings {
		if s.FoodID == 69796849 && s.Amount == 1.0 {
			sawMocha = true
			if s.MealGroup != 1 {
				t.Errorf("Mocha mealGroup = %d (%s), want 1 (Breakfast)", s.MealGroup, MealGroupName(s.MealGroup))
			}
		}
		if s.FoodID == 27536487 && s.MealGroup == 6 { // Water
			sawWater = true
		}
	}
	if !sawMocha {
		t.Errorf("expected the Mocha serving (foodID 69796849, amount 1.0) — field mapping off")
	}
	if !sawWater {
		t.Errorf("expected Water servings in meal group 6 — meal mapping off")
	}
}
