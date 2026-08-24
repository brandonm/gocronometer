package gocronometer

// Regression tests for the 2026-08-24 GWT Measure serialization change
// (Measure/1979099908 + DerivedMeasure/4214796590), which inserted a
// measure-name translation Map between the name string and the Measure$Type
// enum. See deserializeMeasureFields for the recovered layout.
//
// The fixtures are live getAllFood / getFood responses captured on 2026-08-24
// with wire_capture_probe_test.go. The account's user ID has been replaced
// with 10000001.

import (
	"strings"
	"testing"
)

// gwtFoodsFromBatch walks a getAllFood response and returns the raw GWTFood
// values, which — unlike FoodDetail — retain each measure's millilitre volume.
func gwtFoodsFromBatch(t *testing.T, body string) []*GWTFood {
	t.Helper()
	r, err := NewGWTReader(body)
	if err != nil {
		t.Fatalf("GWT reader: %v", err)
	}
	outer := r.ReadObject()
	if !strings.Contains(outer, "ArrayList") {
		t.Fatalf("expected an ArrayList wrapper, got %q", outer)
	}
	count := r.ReadInt()
	foods := make([]*GWTFood, 0, count)
	for i := 0; i < count; i++ {
		typ := r.ReadObject()
		if !strings.Contains(typ, "models.Food/") {
			t.Fatalf("element %d: expected a Food, got %q", i, typ)
		}
		f, err := DeserializeFood(r)
		if err != nil {
			t.Fatalf("element %d: DeserializeFood: %v", i, err)
		}
		foods = append(foods, f)
	}
	return foods
}

// TestParseGetAllFoods_MeasureV3 is the core regression: every food in a live
// V3 batch must resolve with a real name and a full nutrient profile. Before
// the V3 dispatch existed this failed outright with "unknown Measure vintage".
func TestParseGetAllFoods_MeasureV3(t *testing.T) {
	body := loadFixture(t, "testdata/getallfood-2026-08-measure-v3.txt")
	ids := []int64{69796849, 67859823, 22688784, 74791270, 462822, 16248130, 75451580, 20118617, 20398297, 27536487}

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
	}

	// NCCDB food: a stable, fully-populated nutrient profile.
	chicken := foods[462822]
	if chicken.Name != "Chicken Leg, Thigh and Drumstick, Skin Eaten" {
		t.Errorf("chicken name: got %q", chicken.Name)
	}
	if chicken.Source != "NCCDB:6263" {
		t.Errorf("chicken source: got %q", chicken.Source)
	}
	if got := chicken.NutrientsPer100g[203]; !approx(got, 27.27) {
		t.Errorf("chicken protein (203): got %v, want 27.27", got)
	}

	// A database food's measures are physical units, so they are the sharpest
	// alignment check available: if the V3 translation map were mis-walked,
	// these exact constants would shift into neighbouring fields.
	apple := foods[22688784]
	wantMeasures := map[string]float64{"g": 1, "oz": 28.3495231, "Medium apple": 182}
	for _, m := range apple.Measures {
		if want, ok := wantMeasures[m.Name]; ok {
			if !approx(m.Grams, want) {
				t.Errorf("apple measure %q: got %v grams, want %v", m.Name, m.Grams, want)
			}
			delete(wantMeasures, m.Name)
		}
	}
	for name := range wantMeasures {
		t.Errorf("apple measure %q missing", name)
	}

	// Custom recipe: proves the ingredient walk still aligns past the measures.
	mocha := foods[69796849]
	if mocha.Source != "Custom" {
		t.Errorf("mocha source: got %q", mocha.Source)
	}
	if len(mocha.Ingredients) == 0 {
		t.Error("mocha: expected recipe ingredients")
	}
}

// TestParseGetFood_MeasureV3 covers the single-object response shape, which
// wraps the Food differently from the batch call.
func TestParseGetFood_MeasureV3(t *testing.T) {
	body := loadFixture(t, "testdata/getfood-2026-08-measure-v3.txt")

	detail, err := parseGetFoodResponse(body, 69796849)
	if err != nil {
		t.Fatalf("parseGetFoodResponse: %v", err)
	}
	if detail.Name == "" {
		t.Fatal("empty name — the husk failure mode")
	}
	if len(detail.NutrientsPer100g) == 0 {
		t.Fatal("no nutrients decoded")
	}
	// This fixture is a custom recipe, where Cronometer hangs the recipe's
	// total weight on the "g" measure and gives "full recipe" a weight of 1.
	// (Same shape as the 2026-07 V2 recipes, so it is not a V3 artifact.)
	got := map[string]float64{}
	for _, m := range detail.Measures {
		got[m.Name] = m.Grams
	}
	if g, ok := got["g"]; !ok || !approx(g, 40) {
		t.Errorf(`"g" measure: got %v (present=%v), want 40`, g, ok)
	}
	if g, ok := got["full recipe"]; !ok || !approx(g, 1) {
		t.Errorf(`"full recipe" measure: got %v (present=%v), want 1`, g, ok)
	}
}

// TestMeasureV3_VolumeMeasure pins the field that moved. A DerivedMeasure for
// a US gallon carries both a millilitre volume and a gram weight; if the new
// translation Map were mis-walked, these two doubles would shift into each
// other's slots.
func TestMeasureV3_VolumeMeasure(t *testing.T) {
	foods := gwtFoodsFromBatch(t, loadFixture(t, "testdata/getallfood-2026-08-measure-v3.txt"))

	var gallon *GWTMeasure
	var gramCount int
	for _, f := range foods {
		for i := range f.Measures {
			m := &f.Measures[i]
			if m.Name == "Gallon" {
				gallon = m
			}
			if m.Name == "g" {
				gramCount++
			}
		}
	}
	if gramCount == 0 {
		t.Fatal(`no "g" measures decoded at all`)
	}
	if gallon == nil {
		t.Fatal(`expected a "Gallon" DerivedMeasure in the batch`)
	}
	if !approx(gallon.MilliL, 3785.411784) {
		t.Errorf("Gallon millilitres: got %v, want 3785.411784", gallon.MilliL)
	}
	if !approx(gallon.Grams, 1113.3564070588236) {
		t.Errorf("Gallon grams: got %v, want 1113.3564070588236", gallon.Grams)
	}
}

// TestDeserializeMeasure_UnknownVintageFailsLoud is the canary that turned the
// 2026-08-24 change into an alert instead of silent corruption. It must keep
// erroring rather than walking an unknown layout.
func TestDeserializeMeasure_UnknownVintageFailsLoud(t *testing.T) {
	resp := `//OK[1.0,1,["com.cronometer.shared.foods.models.Measure/999999999"],0,7]`
	r, err := NewGWTReader(resp)
	if err != nil {
		t.Fatalf("GWT reader: %v", err)
	}
	if _, err := deserializeMeasure(r); err == nil {
		t.Fatal("expected an error for an unknown Measure vintage, got nil")
	} else if !strings.Contains(err.Error(), "unknown Measure vintage") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestSkipMeasureTranslations covers both branches of the V3 field: an empty
// map (every measure observed so far) is consumed silently, while a populated
// one fails loudly rather than guessing at an unobserved entry layout.
func TestSkipMeasureTranslations(t *testing.T) {
	tests := []struct {
		name    string
		resp    string
		wantErr string
	}{
		{
			name: "empty map is consumed",
			resp: `//OK[0,1,["java.util.HashMap/1797211028"],0,7]`,
		},
		{
			name: "null map is consumed",
			resp: `//OK[0,["java.util.HashMap/1797211028"],0,7]`,
		},
		{
			name:    "populated map fails loudly",
			resp:    `//OK[2,1,["java.util.HashMap/1797211028"],0,7]`,
			wantErr: "measure translations map has 2 entries",
		},
		{
			name:    "wrong type fails loudly",
			resp:    `//OK[1,["com.cronometer.shared.foods.models.Nutrient/331784102"],0,7]`,
			wantErr: "expected a Map at the translations field",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := NewGWTReader(tc.resp)
			if err != nil {
				t.Fatalf("GWT reader: %v", err)
			}
			err = skipMeasureTranslations(r)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected success, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

// TestKnownSignaturesCoverLiveV3 keeps the schema canary and the deserializer
// in lockstep: a vintage the dispatch understands must not trip the alert, and
// a vintage the canary allows must not reach an unknown-layout error.
func TestKnownSignaturesCoverLiveV3(t *testing.T) {
	for class, want := range map[string]string{
		"com.cronometer.shared.foods.models.Measure":       "1979099908",
		"com.cronometer.shared.measurement.DerivedMeasure": "4214796590",
	} {
		known := knownGWTClassSignatures[class]
		found := false
		for _, h := range known {
			if h == want {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: live hash %s missing from knownGWTClassSignatures %v", class, want, known)
		}
	}

	// The full policy discovered live on 2026-08-24. compareGWTClassSignatures
	// reports a watched class as changed when it is absent, so this has to be
	// the complete watched set rather than just the two that moved.
	changes := compareGWTClassSignatures(map[string]string{
		"com.cronometer.shared.entries.models.Day":         "782579793",
		"com.cronometer.shared.entries.models.DayInfo":     "416556043",
		"com.cronometer.shared.entries.models.Serving":     "2553599101",
		"com.cronometer.shared.foods.models.Food":          "2097636843",
		"com.cronometer.shared.foods.models.FoodMeasures":  "2106205728",
		"com.cronometer.shared.foods.models.Ingredient":    "1280520736",
		"com.cronometer.shared.foods.models.Measure":       "1979099908",
		"com.cronometer.shared.foods.models.Nutrient":      "331784102",
		"com.cronometer.shared.foods.models.NutrientMap":   "168231382",
		"com.cronometer.shared.foods.models.Translation":   "4034452093",
		"com.cronometer.shared.measurement.DerivedMeasure": "4214796590",
	})
	if len(changes) != 0 {
		t.Errorf("live V3 signatures reported as incompatible: %+v", changes)
	}
}
