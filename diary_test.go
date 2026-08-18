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

func TestMealGroupFromServingSignature(t *testing.T) {
	tests := []struct {
		name      string
		tokens    []string
		userIndex int
		want      int
	}{
		{
			name:      "older short serving layout",
			tokens:    []string{"user", "0", "327688", "0", "1", "1", "2026", "6", "28", "2", "30"},
			userIndex: 0,
			want:      5,
		},
		{
			name:      "current serving layout",
			tokens:    []string{"user", "0", "48", "13", "15", "327710", "12", "1", "1", "2026", "8", "17", "2", "26"},
			userIndex: 0,
			want:      5,
		},
		{
			name:      "current layout with optional token before type marker",
			tokens:    []string{"user", "0", "48", "13", "15", "327711", "-420", "12", "1", "1", "2026", "8", "17", "2", "26"},
			userIndex: 0,
			want:      5,
		},
		{
			name:      "unknown group",
			tokens:    []string{"user", "0", "48", "13", "15", "458753"},
			userIndex: 0,
			want:      0,
		},
		{
			name:      "short serving",
			tokens:    []string{"user", "0"},
			userIndex: 0,
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mealGroupFromServingSignature(tt.tokens, tt.userIndex); got != tt.want {
				t.Fatalf("mealGroupFromServingSignature() = %d, want %d", got, tt.want)
			}
		})
	}
}
