package gocronometer

import (
	"testing"
)

// Actual getFood(466098) response — Lettuce, Green Leaf (NCCDB)
const lettuceResponse = "//OK[0,0,43,12417149,42,41,40,39,38,28,27,2374339,37,36,35,34,33,28,27,413734,32,30,31,30,29,28,27,3,2,0,26,5,26,6,26,3,25,24,0,0,20,-21,511,0.057,22,511,21,-21,255,94.01,22,255,21,-21,510,0.056,22,510,21,-21,509,0.026,22,509,21,-21,508,0.044,22,508,21,-21,507,0.013,22,507,21,-21,506,0.013,22,506,21,-21,505,0.067,22,505,21,-21,504,0.063,22,504,21,-21,503,0.067,22,503,21,-21,502,0.047,22,502,21,-21,501,0.007,22,501,21,-21,606,0.021,22,606,21,-21,605,0.0,22,605,21,-21,-221,0.0,22,-221,21,-21,601,0.0,22,601,21,-21,343,0.02,22,343,21,-21,342,0.41,22,342,21,-21,214,0.0,22,214,21,-21,853,0.0,22,853,21,-21,341,0.0,22,341,21,-21,213,0.0,22,213,21,-21,212,0.43,22,212,21,-21,851,0.062,22,851,21,-21,211,0.36,22,211,21,-21,338,1730.0,22,338,21,-21,210,0.0,22,210,21,-21,337,0.0,22,337,21,-21,209,0.0,22,209,21,-21,208,18.0,22,208,21,-21,207,0.67,22,207,21,-21,334,0.0,22,334,21,-21,205,4.07,22,205,21,-21,204,0.16,22,204,21,-21,-205,14.529959999999999,22,-205,21,-21,-204,1.3392000000000002,22,-204,21,-21,-203,2.6596800000000003,22,-203,21,-21,326,0.0,22,326,21,-21,325,0.0,22,325,21,-21,324,0.0,22,324,21,-21,323,0.22,22,323,21,-21,322,0.0,22,322,21,-21,321,4443.0,22,321,21,-21,320,370.25,22,320,21,-21,319,0.0,22,319,21,-21,318,7405.0,22,318,21,-21,317,0.6,22,317,21,-21,315,0.156,22,315,21,-21,312,0.04,22,312,21,-21,309,0.31,22,309,21,-21,-1205,2.7460000000000004,22,-1205,21,-21,307,29.0,22,307,21,-21,306,277.0,22,306,21,-21,305,27.0,22,305,21,-21,304,13.0,22,304,21,-21,303,0.32,22,303,21,-21,430,118.5,22,430,21,-21,301,40.0,22,301,21,-21,297,1.3,22,297,21,-21,295,0.0,22,295,21,-21,421,13.6,22,421,21,-21,675,0.0,22,675,21,-21,291,1.3,22,291,21,-21,418,0.0,22,418,21,-21,417,38.0,22,417,21,-21,415,0.071,22,415,21,-21,287,0.01,22,287,21,-21,10012,0.3,22,10012,21,-21,410,0.134,22,410,21,-21,10009,0.0,22,10009,21,-21,10007,0.024,22,10007,21,-21,406,0.375,22,406,21,-21,10005,1.2,22,10005,21,-21,405,0.08,22,405,21,-21,404,0.082,22,404,21,-21,10002,0.0,22,10002,21,-21,10001,0.062,22,10001,21,-21,401,15.2,22,401,21,-21,269,0.78,22,269,21,-21,646,0.087,22,646,21,-21,518,0.031,22,518,21,-21,262,0.0,22,262,21,-21,645,0.006,22,645,21,-21,517,0.038,22,517,21,-21,516,0.046,22,516,21,-21,515,0.146,22,515,21,-21,514,0.114,22,514,21,-21,513,0.045,22,513,21,0,23,512,0.018,22,512,21,94,20,0,19,18,0.7501478140000001,-8,17,0,1079815,466098,0,1.0,8,1.0,-8,16,0,1079819,466098,0,1.0,8,2.250445936,-8,15,0,1079816,466098,0,1.0,8,4.8,-8,14,0,1079812,466098,0,1.0,8,24.0,-8,13,0,1079813,466098,0,1.0,8,28.3495231,-8,12,0,1080738,466098,0,1.0,8,36.007125,-8,11,0,1079817,466098,0,1.0,8,360.0,0,10,9,0,1079814,466098,0,1.0,8,8,2,1079813,7,\"Zjjf7wA\",0,6,0,466098,0,0,5,26,4,3,1,2,0,0,1,[\"com.cronometer.shared.foods.models.Food/2097636843\",\"java.util.ArrayList/4159755760\",\"java.lang.String/2004016611\",\"4076\",\"vegetables, lettuce, green leaf\",\"com.cronometer.shared.foods.NutritionLabelType/1598919019\",\"com.cronometer.shared.foods.models.FoodMeasures/2106205728\",\"com.cronometer.shared.foods.models.Measure/824760657\",\"head\",\"com.cronometer.shared.foods.models.Measure\\u0024Type/2365167904\",\"cup, chopped\",\"oz\",\"outer leaf - large\",\"inner leaf - small\",\"tbsp, chopped\",\"g\",\"tsp, chopped\",\"com.cronometer.shared.foods.models.NutrientMap/168231382\",\"com.cronometer.shared.foods.models.NutrientMap\\u0024NutrientFilter/1990310964\",\"java.util.HashMap/1797211028\",\"java.lang.Integer/3438268394\",\"com.cronometer.shared.foods.models.Nutrient/331784102\",\"com.cronometer.shared.foods.models.Nutrient\\u0024Type/4187872513\",\"NCCDB:13930\",\"java.util.HashSet/3273092938\",\"com.cronometer.shared.foods.FoodTag/3220417118\",\"com.cronometer.shared.foods.models.Translation/4034452093\",\"com.cronometer.shared.user.models.Language/1257207975\",\"en\",\"English\",\"https://cdn1.cronometer.com/media/flags/us.png\",\"Lettuce, Green Leaf\",\"fr\",\"French\",\"https://cdn1.cronometer.com/media/flags/fr.png\",\"Fran\\u00e7ais\",\"Laitue Fris\\u00e9e\",\"de\",\"German\",\"https://cdn1.cronometer.com/media/flags/de.png\",\"Deutsch\",\"Kopfsalat, Gr\\u00fcnes Blatt\",\"com.cronometer.shared.foods.FoodType/2323555378\"],0,7]"

func TestDeserializeFood_Lettuce(t *testing.T) {
	r, err := NewGWTReader(lettuceResponse)
	if err != nil {
		t.Fatalf("NewGWTReader: %v", err)
	}

	// Dump last 20 tokens (where reading starts)
	tokens := r.Tokens()
	start := len(tokens) - 20
	if start < 0 {
		start = 0
	}
	t.Logf("Last 20 tokens: %v", tokens[start:])
	t.Logf("Total tokens: %d, index after version/flags: %d", len(tokens), r.Remaining())

	// Read the Food type signature
	typeSig := r.ReadObject()
	t.Logf("Type: %s (remaining: %d)", typeSig, r.Remaining())

	// First, manually read the first few fields to understand the pattern
	t.Log("=== Manual field reads ===")
	for i := 0; i < 40 && r.Remaining() > 0; i++ {
		raw := r.Peek()
		// Try as int
		intVal := r.ReadInt()
		strVal := r.GetString(intVal)
		if strVal != "" && len(strVal) < 80 {
			t.Logf("  [%d] raw=%q → int=%d → str=%q", i, raw, intVal, strVal)
		} else {
			t.Logf("  [%d] raw=%q → int=%d", i, raw, intVal)
		}
	}
	t.FailNow() // stop here for debugging

	if typeSig == "" || typeSig == "com.cronometer.shared.foods.models.Food/2097636843" {
		if typeSig == "" {
			t.Skip("Food type token was null — response format may differ")
		}

		food, err := DeserializeFood(r)
		if err != nil {
			t.Logf("DeserializeFood error (may be partial): %v", err)
		}

		t.Logf("Food: ID=%d Name=%q Description=%q Source=%q", food.ID, food.Name, food.Description, food.Source)
		t.Logf("Nutrients: %d entries", len(food.Nutrients))
		t.Logf("Ingredients: %d", len(food.Ingredients))
		t.Logf("Measures: %d", len(food.Measures))
		t.Logf("Translations: %v", food.Translations)

		// Verify key fields
		if food.Name != "Lettuce, Green Leaf" {
			t.Errorf("Name: got %q, want %q", food.Name, "Lettuce, Green Leaf")
		}

		if food.Source != "NCCDB:13930" {
			t.Errorf("Source: got %q, want %q", food.Source, "NCCDB:13930")
		}

		// Should have nutrients
		if len(food.Nutrients) == 0 {
			t.Error("expected nutrients, got none")
		} else {
			// Check energy (USDA code 208)
			if cal, ok := food.Nutrients[208]; ok {
				if cal != 18.0 {
					t.Errorf("Energy (208): got %f, want 18.0", cal)
				}
			} else {
				t.Error("missing energy nutrient (code 208)")
			}
		}

		// Should have no ingredients (simple food, not a recipe)
		if len(food.Ingredients) != 0 {
			t.Errorf("expected 0 ingredients for simple food, got %d", len(food.Ingredients))
		}
	}
}
