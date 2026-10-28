package gocronometer

import (
	"testing"
)

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
