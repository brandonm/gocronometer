package gocronometer

import (
	"testing"
)

const findMyFoodsResponse = `//OK[0,-6,0,0,-3,0,0,9,0,0,0,67861262,0,0,2,0,-4,0,0,-3,0,0,8,0,0,0,67861120,0,0,2,0,-4,0,0,-3,0,0,7,0,0,0,67860387,0,0,2,0,2,5,0,0,-3,0,0,6,0,0,0,67859823,0,0,2,0,1,5,0,0,6,4,0,0,3,0,0,0,67859683,0,0,2,5,1,["[Lcom.cronometer.shared.foods.models.SearchHit;/3199648558","com.cronometer.shared.foods.models.SearchHit/1904627920","Egg whites + Spinach","com.cronometer.shared.foods.FoodSource/4236433762","com.cronometer.shared.foods.FoodType/2323555378","Egg whites + Spinach + Banana Breakfast","Mocha (20oz) - Monin Chocolate + %2 Milk","Basic Sandwich","Turkey/Ham Sandwich"],0,7]`

func TestParseFindMyFoodsResponse(t *testing.T) {
	foods, err := parseFindMyFoodsResponse(findMyFoodsResponse)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(foods) != 5 {
		t.Fatalf("expected 5 foods, got %d", len(foods))
	}

	// Build a map for easier assertions
	byName := make(map[string]int64)
	for _, f := range foods {
		byName[f.Name] = f.ID
	}

	expected := map[string]int64{
		"Turkey/Ham Sandwich":                     67861262,
		"Basic Sandwich":                          67861120,
		"Mocha (20oz) - Monin Chocolate + %2 Milk": 67860387,
		"Egg whites + Spinach + Banana Breakfast":  67859823,
		"Egg whites + Spinach":                     67859683,
	}

	for name, expectedID := range expected {
		gotID, ok := byName[name]
		if !ok {
			t.Errorf("missing food: %s", name)
			continue
		}
		if gotID != expectedID {
			t.Errorf("food %q: expected ID %d, got %d", name, expectedID, gotID)
		}
	}
}

func TestParseFoodExport(t *testing.T) {
	csv := `Food ID,Food Name,Comments,Amount,Energy (kcal),Alcohol (g),Protein (g),Fat (g),Carbs (g),Fiber (g),Sodium (mg),Calcium (mg)
67861120,"Basic Sandwich","","full recipe  — 151.0g",328.64,0.00,12.52,21.08,25.95,4.62,643.92,259.20`

	export, err := ParseFoodExport(csv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if export.FoodID != 67861120 {
		t.Errorf("expected food ID 67861120, got %d", export.FoodID)
	}
	if export.FoodName != "Basic Sandwich" {
		t.Errorf("expected food name 'Basic Sandwich', got %q", export.FoodName)
	}
	if export.Amount != `full recipe  — 151.0g` {
		t.Errorf("unexpected amount: %q", export.Amount)
	}

	assertNutrient := func(name string, expected float64) {
		t.Helper()
		got, ok := export.Nutrients[name]
		if !ok {
			t.Errorf("missing nutrient: %s", name)
			return
		}
		if got != expected {
			t.Errorf("nutrient %s: expected %f, got %f", name, expected, got)
		}
	}

	assertNutrient("Energy (kcal)", 328.64)
	assertNutrient("Protein (g)", 12.52)
	assertNutrient("Fat (g)", 21.08)
	assertNutrient("Carbs (g)", 25.95)
	assertNutrient("Fiber (g)", 4.62)
	assertNutrient("Sodium (mg)", 643.92)
	assertNutrient("Calcium (mg)", 259.20)

	// Zero values should be excluded
	if _, ok := export.Nutrients["Alcohol (g)"]; ok {
		t.Error("zero-value nutrient 'Alcohol (g)' should not be included")
	}
}

func TestParseFoodExportFullCSV(t *testing.T) {
	// Real CSV from Cronometer export for Turkey/Ham Sandwich
	csv := `Food ID,Food Name,Comments,Amount,Energy (kcal),Alcohol (g),Ash (g),Beta-Hydroxybutyrate (g),Caffeine (mg),Oxalate (mg),Phytate (mg),Water (g),B1 (Thiamine) (mg),B2 (Riboflavin) (mg),B3 (Niacin) (mg),B5 (Pantothenic Acid) (mg),B6 (Pyridoxine) (mg),B12 (Cobalamin) (µg),Alpha-carotene (µg),Beta Tocopherol (mg),Beta-carotene (µg),Beta-cryptoxanthin (µg),Biotin (µg),Choline (mg),Delta Tocopherol (mg),Folate (µg),Gamma Tocopherol (mg),Lutein+Zeaxanthin (µg),Lycopene (µg),Retinol (µg),Vitamin A (µg),Vitamin C (mg),Vitamin D (IU),Vitamin E (mg),Vitamin K (µg),Calcium (mg),Chromium (µg),Copper (mg),Fluoride (µg),Iodine (µg),Iron (mg),Magnesium (mg),Manganese (mg),Molybdenum (µg),Phosphorus (mg),Potassium (mg),Selenium (µg),Sodium (mg),Zinc (mg),Net Carbs (g),Allulose (g),Carbs (g),Fiber (g),Insoluble Fiber (g),Soluble Fiber (g),Fructose (g),Galactose (g),Glucose (g),Lactose (g),Maltose (g),Starch (g),Sucrose (g),Sugars (g),Added Sugars (g),Sugar Alcohol (g),Fat (g),Cholesterol (mg),Monounsaturated (g),Polyunsaturated (g),Saturated (g),Trans-Fats (g),Omega-3 (g),ALA (g),DHA (g),EPA (g),Omega-6 (g),AA (g),LA (g),Phytosterol (mg),Alanine (g),Arginine (g),Aspartic acid (g),Cystine (g),Glutamic acid (g),Glycine (g),Histidine (g),Hydroxyproline (g),Isoleucine (g),Leucine (g),Lysine (g),Methionine (g),Phenylalanine (g),Proline (g),Protein (g),Serine (g),Threonine (g),Tryptophan (g),Tyrosine (g),Valine (g)
67861262,"Turkey/Ham Sandwich","","full recipe  — 235.0g",413.95,0.00,0.32,,0.00,0.14,1.15,45.12,0.04,0.04,0.18,0.06,0.03,0.00,0.00,0.00,2132.64,0.00,,6.53,0.01,18.24,0.20,830.40,0.00,0.00,177.72,7.30,0.00,0.11,56.88,272.33,,0.02,,0.58,1.75,6.24,0.07,,12.96,689.52,0.29,1385.48,0.15,21.97,,26.61,4.62,0.62,0.00,0.21,0.00,0.17,0.00,0.00,0.00,0.00,2.70,2.33,0.01,22.72,71.09,2.50,6.04,7.51,0.00,0.03,0.03,0.00,0.00,0.00,0.00,0.00,,0.02,0.03,0.05,0.01,0.07,0.02,0.01,,0.03,0.03,0.03,0.01,0.02,0.02,30.24,0.01,0.02,0.00,0.01,0.03`

	export, err := ParseFoodExport(csv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if export.FoodID != 67861262 {
		t.Errorf("expected food ID 67861262, got %d", export.FoodID)
	}
	if export.FoodName != "Turkey/Ham Sandwich" {
		t.Errorf("expected 'Turkey/Ham Sandwich', got %q", export.FoodName)
	}

	// Verify key nutrients are parsed
	if v, ok := export.Nutrients["Energy (kcal)"]; !ok || v != 413.95 {
		t.Errorf("Energy: got %v, %v", v, ok)
	}
	if v, ok := export.Nutrients["Protein (g)"]; !ok || v != 30.24 {
		t.Errorf("Protein: got %v, %v", v, ok)
	}
	if v, ok := export.Nutrients["Sodium (mg)"]; !ok || v != 1385.48 {
		t.Errorf("Sodium: got %v, %v", v, ok)
	}
}

func TestExtractGWTStringTable(t *testing.T) {
	body := `//OK[1,2,3,["hello","world"],0,7]`
	strings, err := extractGWTStringTable(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(strings) != 2 {
		t.Fatalf("expected 2 strings, got %d", len(strings))
	}
	if strings[0] != "hello" || strings[1] != "world" {
		t.Errorf("got %v", strings)
	}
}

func TestExtractGWTNumericData(t *testing.T) {
	body := `//OK[1,2,67861262,["strings"],0,7]`
	data, err := extractGWTNumericData(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 3 {
		t.Fatalf("expected 3 values, got %d: %v", len(data), data)
	}
	if data[2] != "67861262" {
		t.Errorf("expected 67861262, got %s", data[2])
	}
}

// TestParseGetFoodIngredients tests ingredient extraction using the real Basic Sandwich response.
func TestParseGetFoodIngredients(t *testing.T) {
	ingredients, err := parseGetFoodIngredients(basicSandwichResponse)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ingredients) != 5 {
		t.Fatalf("expected 5 ingredients, got %d", len(ingredients))
	}

	expectedFoodIDs := map[int64]bool{
		466098: true, 7643764: true, 18812812: true, 3739632: true, 21649514: true,
	}
	for _, ing := range ingredients {
		if !expectedFoodIDs[ing.FoodID] {
			t.Errorf("unexpected food ID: %d", ing.FoodID)
		}
		delete(expectedFoodIDs, ing.FoodID)
	}
	for id := range expectedFoodIDs {
		t.Errorf("missing food ID: %d", id)
	}

	amountByFood := make(map[int64]float64)
	for _, ing := range ingredients {
		amountByFood[ing.FoodID] = ing.Amount
	}
	if amountByFood[466098] != 48.0 {
		t.Errorf("food 466098: expected amount 48.0, got %f", amountByFood[466098])
	}
	if amountByFood[21649514] != 52.0 {
		t.Errorf("food 21649514: expected amount 52.0, got %f", amountByFood[21649514])
	}
}

// TestParseGetFoodNoIngredients tests that a simple food (Lettuce) has no ingredients.
func TestParseGetFoodNoIngredients(t *testing.T) {
	ingredients, err := parseGetFoodIngredients(lettuceResponse)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ingredients) != 0 {
		t.Errorf("expected 0 ingredients for simple food, got %d", len(ingredients))
	}
}

// TestParseGetAllFoodsResponse tests the batch parsing wrapper that the collector calls.
func TestParseGetAllFoodsResponse(t *testing.T) {
	requestedIDs := []int64{21649514, 3739632, 18812812, 7643764, 466098}

	results, err := parseGetAllFoodsResponse(getAllFoodResponse, requestedIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 5 {
		t.Fatalf("expected 5 foods, got %d", len(results))
	}

	// Verify Nature's Harvest Bread (CRDB)
	bread := results[21649514]
	if bread == nil {
		t.Fatal("missing food 21649514 (Bread)")
	}
	if bread.Name != "Nature's Harvest, Bread, 100% Whole Wheat Bread" {
		t.Errorf("Bread name: got %q", bread.Name)
	}
	if bread.Source != "CRDB" {
		t.Errorf("Bread source: got %q, want CRDB", bread.Source)
	}
	if len(bread.NutrientsPer100g) == 0 {
		t.Error("Bread: expected nutrients")
	}

	// Verify Lettuce (NCCDB)
	lettuce := results[466098]
	if lettuce == nil {
		t.Fatal("missing food 466098 (Lettuce)")
	}
	if lettuce.Source != "NCCDB:13930" {
		t.Errorf("Lettuce source: got %q, want NCCDB:13930", lettuce.Source)
	}
	if lettuce.Name != "Lettuce, Green Leaf" {
		t.Errorf("Lettuce name: got %q", lettuce.Name)
	}

	// Verify all foods have nutrients
	for id, food := range results {
		if len(food.NutrientsPer100g) == 0 {
			t.Errorf("food %d (%s): no nutrients", id, food.Name)
		}
	}
}

// TestParseGetFoodResponse tests the single-food parsing wrapper.
func TestParseGetFoodResponse(t *testing.T) {
	detail, err := parseGetFoodResponse(basicSandwichResponse, 67861120)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detail.FoodID != 67861120 {
		t.Errorf("FoodID: got %d, want 67861120", detail.FoodID)
	}
	if detail.Name != "Basic Sandwich" {
		t.Errorf("Name: got %q, want Basic Sandwich", detail.Name)
	}
	if detail.Source != "Custom" {
		t.Errorf("Source: got %q, want Custom", detail.Source)
	}
	if len(detail.Ingredients) != 5 {
		t.Errorf("Ingredients: got %d, want 5", len(detail.Ingredients))
	}
	if len(detail.NutrientsPer100g) == 0 {
		t.Error("expected nutrients")
	}
	if detail.NutrientsPer100g[208] != 328.64 {
		t.Errorf("Energy: got %f, want 328.64", detail.NutrientsPer100g[208])
	}
}
