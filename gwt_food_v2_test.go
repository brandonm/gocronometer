package gocronometer

// Regression tests for the 2026-10-28 GWT Food serialization change
// (Food/3482863798), which dropped one of the two always-zero ints preceding
// the food ID, shifting every later field up by one. See deserializeFoodFields
// for the recovered layout.
//
// The fixtures are live getAllFood / getFood responses captured on 2026-10-28
// with wire_capture_probe_test.go. The account's user ID has been replaced
// with 10000001.

import (
	"strings"
	"testing"
)

// TestParseGetAllFoods_FoodV2 is the core regression: every food in a live V2
// batch must resolve with a real name, a full nutrient profile, and a source
// that is a database tag rather than a stray type signature.
func TestParseGetAllFoods_FoodV2(t *testing.T) {
	body := loadFixture(t, "testdata/getallfood-2026-10-food-v2.txt")
	ids := []int64{81323197, 68943617, 450856, 80069621, 22688784, 14570120, 20398297, 20118617, 28932880, 27536487}

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
			t.Errorf("food %d: empty name — the husk failure mode", id)
		}
		if len(f.NutrientsPer100g) == 0 {
			t.Errorf("food %d (%q): no nutrients", id, f.Name)
		}
		if strings.Contains(f.Source, "/") {
			t.Errorf("food %d (%q): source %q is a type signature — stream misaligned", id, f.Name, f.Source)
		}
	}

	banana := foods[450856]
	if banana.Name != "Bananas, Raw" || banana.Source != "NCCDB:4974" {
		t.Errorf("banana: got name %q source %q", banana.Name, banana.Source)
	}
}

// TestDeserializeFood_V2Fields checks the fields on both sides of the dropped
// int, plus the trailing user ID, on custom and database foods. Walking V2
// with the V1 layout reads the ingredient list as the food ID and lands every
// later field one token late, so these values cannot survive a misaligned walk.
func TestDeserializeFood_V2Fields(t *testing.T) {
	foods := gwtFoodsFromBatch(t, loadFixture(t, "testdata/getallfood-2026-10-food-v2.txt"))
	byID := make(map[int]*GWTFood, len(foods))
	for _, f := range foods {
		byID[f.ID] = f
	}

	mocha, ok := byID[81323197]
	if !ok {
		t.Fatal("custom food 81323197 missing — food ID field misaligned")
	}
	if mocha.UserID != 10000001 {
		t.Errorf("mocha user ID: got %d, want 10000001", mocha.UserID)
	}
	want := []GWTIngredient{
		{FoodID: 450230, MeasureID: 995402, Amount: 40},
		{FoodID: 15431433, MeasureID: 42194944, Amount: 100},
		{FoodID: 26273465, MeasureID: 90510270, Amount: 197.15686374999999},
	}
	if len(mocha.Ingredients) != len(want) {
		t.Fatalf("mocha ingredients: got %d, want %d", len(mocha.Ingredients), len(want))
	}
	for i, w := range want {
		if mocha.Ingredients[i] != w {
			t.Errorf("mocha ingredient %d: got %+v, want %+v", i, mocha.Ingredients[i], w)
		}
	}

	// Database foods carry no user ID, and their measures are physical
	// constants — the sharpest alignment check available.
	banana := byID[450856]
	if banana == nil {
		t.Fatal("database food 450856 missing")
	}
	if banana.UserID != 0 {
		t.Errorf("banana user ID: got %d, want 0", banana.UserID)
	}
	grams := map[string]float64{}
	for _, m := range banana.Measures {
		grams[m.Name] = m.Grams
	}
	if grams["g"] != 1 || grams["oz"] != 28.3495231 {
		t.Errorf("banana measures: g=%v oz=%v, want 1 and 28.3495231", grams["g"], grams["oz"])
	}
}

// TestParseGetAllFoods_BackReferencedPropertyKey covers a second food whose
// properties map reuses a key String already serialized by an earlier food.
// GWT writes that key as a lone back-reference token; reading a string after
// it misaligned the walk by one token and decoded the omelette's source as
// "java.util.HashSet/3273092938".
func TestParseGetAllFoods_BackReferencedPropertyKey(t *testing.T) {
	foods := gwtFoodsFromBatch(t, loadFixture(t, "testdata/getallfood-2026-10-food-v2.txt"))
	var omelette *GWTFood
	for _, f := range foods {
		if f.ID == 68943617 {
			omelette = f
		}
	}
	if omelette == nil {
		t.Fatal("omelette 68943617 missing")
	}
	if omelette.Source != "Custom" {
		t.Errorf("omelette source: got %q, want \"Custom\"", omelette.Source)
	}
	if omelette.UserID != 10000001 {
		t.Errorf("omelette user ID: got %d, want 10000001", omelette.UserID)
	}
}

func TestParseGetFood_FoodV2(t *testing.T) {
	body := loadFixture(t, "testdata/getfood-2026-10-food-v2.txt")
	f, err := parseGetFoodResponse(body, 81323197)
	if err != nil {
		t.Fatalf("parseGetFoodResponse: %v", err)
	}
	if f.Name != "Mocha (20oz) - Monin Chocolate (Sugar Free) + Nonfat Milk" {
		t.Errorf("name: got %q", f.Name)
	}
	if f.Source != "Custom" {
		t.Errorf("source: got %q", f.Source)
	}
	if len(f.NutrientsPer100g) == 0 {
		t.Error("no nutrients")
	}
}

// TestDeserializeFood_UnknownVintage is the canary itself: a Food hash the
// library has not decoded must stop the walk rather than guess at a layout.
func TestDeserializeFood_UnknownVintage(t *testing.T) {
	body := loadFixture(t, "testdata/getfood-2026-10-food-v2.txt")
	r, err := NewGWTReader(body)
	if err != nil {
		t.Fatalf("GWT reader: %v", err)
	}
	r.ReadObject()
	_, err = DeserializeFood(r, "com.cronometer.shared.foods.models.Food/1")
	if err == nil || !strings.Contains(err.Error(), "unknown Food vintage") {
		t.Fatalf("expected unknown Food vintage error, got %v", err)
	}
}

func TestKnownGWTClassSignatures_LiveFoodV2Policy(t *testing.T) {
	// The full policy discovered live on 2026-10-28. compareGWTClassSignatures
	// reports a watched class as changed when it is absent, so this has to be
	// the complete watched set rather than just the one that moved.
	changes := compareGWTClassSignatures(map[string]string{
		"com.cronometer.shared.entries.models.Day":         "782579793",
		"com.cronometer.shared.entries.models.DayInfo":     "416556043",
		"com.cronometer.shared.entries.models.Serving":     "2553599101",
		"com.cronometer.shared.foods.models.Food":          "3482863798",
		"com.cronometer.shared.foods.models.FoodMeasures":  "2106205728",
		"com.cronometer.shared.foods.models.Ingredient":    "1280520736",
		"com.cronometer.shared.foods.models.Measure":       "1979099908",
		"com.cronometer.shared.foods.models.Nutrient":      "331784102",
		"com.cronometer.shared.foods.models.NutrientMap":   "168231382",
		"com.cronometer.shared.foods.models.Translation":   "4034452093",
		"com.cronometer.shared.measurement.DerivedMeasure": "4214796590",
	})
	if len(changes) != 0 {
		t.Errorf("live Food V2 signatures reported as incompatible: %+v", changes)
	}
}
