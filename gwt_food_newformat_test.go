package gocronometer

// Regression tests for the post-2026-07-16 GWT Food serialization format
// (Measure/1410168823 + DerivedMeasure + back-referenced Language instances).
// The fixtures are getAllFood responses captured from the live Cronometer web
// app on 2026-07-17 with Playwright devtools. See deserializeMeasureNew for
// the reverse-engineered layout.

import (
	"math"
	"os"
	"testing"
)

func loadFixture(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return string(body)
}

func approx(got, want float64) bool {
	return math.Abs(got-want) < 0.01
}

// TestParseGetAllFoods_NewFormat_Burrito parses the 10-food capture that
// includes custom recipes (Basic Burrito, Basic Sandwich, an omelette recipe),
// NCCDB foods, and CRDB barcode foods.
func TestParseGetAllFoods_NewFormat_Burrito(t *testing.T) {
	body := loadFixture(t, "testdata/getallfood-2026-07-new-format-burrito.txt")
	ids := []int64{450230, 7345420, 15431433, 67859683, 450856, 67861120, 13605711, 67915870, 462802, 31290980}

	foods, err := parseGetAllFoodsResponse(body, ids)
	if err != nil {
		t.Fatalf("parseGetAllFoodsResponse: %v", err)
	}
	if len(foods) != len(ids) {
		t.Fatalf("expected %d foods, got %d", len(ids), len(foods))
	}
	for _, id := range ids {
		f, ok := foods[id]
		if !ok {
			t.Errorf("food %d missing from result", id)
			continue
		}
		if f.Name == "" {
			t.Errorf("food %d: empty name", id)
		}
	}

	// NCCDB database food with full nutrient profile.
	thigh := foods[462802]
	if thigh.Name != "Chicken Thigh, Skin Removed" {
		t.Errorf("thigh name: got %q", thigh.Name)
	}
	if thigh.Source != "NCCDB:6272" {
		t.Errorf("thigh source: got %q", thigh.Source)
	}
	if got := thigh.NutrientsPer100g[203]; !approx(got, 27.71) {
		t.Errorf("thigh protein (203): got %v, want 27.71", got)
	}
	if got := thigh.NutrientsPer100g[208]; !approx(got, 173) {
		t.Errorf("thigh energy (208): got %v, want 173", got)
	}

	// CRDB barcode food.
	tortilla := foods[31290980]
	if tortilla.Name != "Mission, Flour Tortillas, Burrito" {
		t.Errorf("tortilla name: got %q", tortilla.Name)
	}
	if got := tortilla.NutrientsPer100g[208]; !approx(got, 295.7746478873239) {
		t.Errorf("tortilla energy (208): got %v, want 295.77", got)
	}
	if got := tortilla.NutrientsPer100g[307]; !approx(got, 830.9859154929577) {
		t.Errorf("tortilla sodium (307): got %v, want 830.99", got)
	}

	// Custom recipe: the nested sub-recipe of "Chicken Burrito" that the
	// 2026-07-16 husk incident stored with zero nutrients. Its ingredient
	// list is the entire point of resolving it.
	burrito := foods[67915870]
	if burrito.Name != "Basic Burrito" {
		t.Errorf("burrito name: got %q", burrito.Name)
	}
	if burrito.Source != "Custom" {
		t.Errorf("burrito source: got %q", burrito.Source)
	}
	if len(burrito.Ingredients) != 5 {
		t.Errorf("burrito ingredients: got %d, want 5", len(burrito.Ingredients))
	}
	if len(burrito.NutrientsPer100g) == 0 {
		t.Error("burrito: expected per-100g nutrients")
	}

	// Custom recipe already pinned by the old-format fixture — the name must
	// agree across vintages.
	sandwich := foods[67861120]
	if sandwich.Name != "Basic Sandwich" {
		t.Errorf("sandwich name: got %q", sandwich.Name)
	}
	if len(sandwich.Ingredients) != 5 {
		t.Errorf("sandwich ingredients: got %d, want 5", len(sandwich.Ingredients))
	}

	// New-format Measure walk: the milk carries 9 measures including the
	// DerivedMeasure volume set ("Gallon", "Quart", "Pint", "Cup").
	milk := foods[7345420]
	if len(milk.Measures) < 5 {
		t.Errorf("milk measures: got %d, want >= 5", len(milk.Measures))
	}
	var sawGallon bool
	for _, m := range milk.Measures {
		if m.Name == "Gallon" {
			sawGallon = true
			if m.Grams <= 0 {
				t.Errorf("Gallon measure grams: got %v, want > 0", m.Grams)
			}
		}
	}
	if !sawGallon {
		t.Error("milk: expected a Gallon DerivedMeasure")
	}
}

// TestParseGetAllFoods_NewFormat_Mixed parses the 14-food capture. Two of the
// foods (Lettuce 466098, Bread 21649514) also appear in the old-format
// fixtures, pinning name/source consistency across serialization vintages.
func TestParseGetAllFoods_NewFormat_Mixed(t *testing.T) {
	body := loadFixture(t, "testdata/getallfood-2026-07-new-format-mixed.txt")
	ids := []int64{61604351, 4174072, 20041618, 3491290, 21649514, 3739632, 18812812, 7643764, 466098, 465141, 455434, 465854, 451081, 461926}

	foods, err := parseGetAllFoodsResponse(body, ids)
	if err != nil {
		t.Fatalf("parseGetAllFoodsResponse: %v", err)
	}
	if len(foods) != len(ids) {
		t.Fatalf("expected %d foods, got %d", len(ids), len(foods))
	}
	for _, id := range ids {
		f, ok := foods[id]
		if !ok {
			t.Errorf("food %d missing from result", id)
			continue
		}
		if f.Name == "" {
			t.Errorf("food %d: empty name", id)
		}
		if len(f.NutrientsPer100g) == 0 {
			t.Errorf("food %d (%s): no nutrients", id, f.Name)
		}
	}

	// Cross-vintage ground truth: same foods exist in the old-format fixtures.
	if got := foods[466098].Name; got != "Lettuce, Green Leaf" {
		t.Errorf("lettuce name: got %q", got)
	}
	if got := foods[466098].Source; got != "NCCDB:13930" {
		t.Errorf("lettuce source: got %q", got)
	}
	if got := foods[21649514].Name; got != "Nature's Harvest, Bread, 100% Whole Wheat Bread" {
		t.Errorf("bread name: got %q", got)
	}
	if got := foods[466098].NutrientsPer100g[208]; !approx(got, 18.0) {
		t.Errorf("lettuce energy (208): got %v, want 18.0 (old-fixture ground truth)", got)
	}
}

// TestParseGetFood_NewFormat_RejectsWrongType pins the fail-loud behavior:
// a response whose payload is not a Food object must error, never return a
// husk (the 2026-07-16 incident inserted empty foods for months of diary
// history because this used to pass).
func TestParseGetFood_NewFormat_RejectsWrongType(t *testing.T) {
	// A batch response is an ArrayList, not a Food — must be rejected.
	body := loadFixture(t, "testdata/getallfood-2026-07-new-format-burrito.txt")
	if _, err := parseGetFoodResponse(body, 450230); err == nil {
		t.Fatal("expected error for non-Food payload, got nil")
	}
}
