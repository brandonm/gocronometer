package gocronometer

import (
	"strings"
	"testing"
)

// Actual getFood(67861120) response — Basic Sandwich recipe (Custom, 5 ingredients)
const basicSandwichResponse = `//OK[15983721,1,29,75865366,28,26,27,26,25,24,23,1,2,0,22,21,0,20,18,19,18,1,14,-21,511,0.02736,16,511,15,-21,255,45.1248,16,255,15,-21,510,0.02688,16,510,15,-21,509,0.01248,16,509,15,-21,508,0.021119999999999996,16,508,15,-21,507,0.00624,16,507,15,-21,506,0.00624,16,506,15,-21,505,0.03216,16,505,15,-21,504,0.03024,16,504,15,-21,503,0.03216,16,503,15,-21,502,0.02256,16,502,15,-21,246,1.152,16,246,15,-21,629,0.0,16,629,15,-21,501,0.00336,16,501,15,-21,621,0.0,16,621,15,-21,606,7.510080000000001,16,606,15,-21,605,0.0,16,605,15,-21,221,0.0,16,221,15,-21,-221,0.0,16,-221,15,-21,601,35.0,16,601,15,-21,343,0.0096,16,343,15,-21,342,0.19679999999999997,16,342,15,-21,214,0.0,16,214,15,-21,853,0.0,16,853,15,-21,341,0.0,16,341,15,-21,213,0.0,16,213,15,-21,212,0.2064,16,212,15,-21,851,0.029759999999999998,16,851,15,-21,211,0.17279999999999998,16,211,15,-21,338,830.4,16,338,15,-21,210,0.0,16,210,15,-21,337,0.0,16,337,15,-21,209,0.0,16,209,15,-21,208,328.64,16,208,15,-21,207,0.3216,16,207,15,-21,334,0.0,16,334,15,-21,205,25.9536,16,205,15,-21,-205,85.66290539016394,16,-205,15,-21,204,21.076800000000002,16,204,15,-21,-204,193.20487688992978,16,-204,15,-21,203,12.5232,16,203,15,-21,-203,50.02606091990632,16,-203,15,-21,324,0.0,16,324,15,-21,323,0.1056,16,323,15,-21,322,0.0,16,322,15,-21,321,2132.64,16,321,15,-21,320,177.72,16,320,15,-21,319,0.0,16,319,15,-21,317,0.288,16,317,15,-21,315,0.07488,16,315,15,-21,312,0.0192,16,312,15,-21,309,0.1488,16,309,15,-21,-1205,21.318080000000002,16,-1205,15,-21,307,643.92,16,307,15,-21,306,262.96000000000004,16,306,15,-21,305,12.959999999999999,16,305,15,-21,304,6.24,16,304,15,-21,303,1.3536,16,303,15,-21,430,56.879999999999995,16,430,15,-21,301,259.20000000000005,16,301,15,-21,297,0.624,16,297,15,-21,295,0.0,16,295,15,-21,421,6.528,16,421,15,-21,675,0.0,16,675,15,-21,291,4.624,16,291,15,-21,418,0.0,16,418,15,-21,417,18.24,16,417,15,-21,415,0.03407999999999999,16,415,15,-21,287,0.0048,16,287,15,-21,10012,0.144,16,10012,15,-21,410,0.06432,16,410,15,-21,10009,2.0,16,10009,15,-21,10007,0.011519999999999999,16,10007,15,-21,406,0.18,16,406,15,-21,10005,0.576,16,10005,15,-21,405,0.0384,16,405,15,-21,404,0.03936,16,404,15,-21,10002,0.0,16,10002,15,-21,10001,0.029759999999999998,16,10001,15,-21,401,7.295999999999999,16,401,15,-21,269,2.3744,16,269,15,-21,646,6.041759999999999,16,646,15,-21,518,0.014879999999999999,16,518,15,-21,262,0.0,16,262,15,-21,645,2.50288,16,645,15,-21,517,0.01824,16,517,15,-21,516,0.02208,16,516,15,-21,515,0.07007999999999999,16,515,15,-21,514,0.05472,16,514,15,-21,513,0.021599999999999998,16,513,15,0,17,512,0.008639999999999998,16,512,15,91,14,0,13,12,1.0,-13,11,0,237264512,67861120,0,1.0,7,1.0,-13,10,0,237264510,67861120,0,1.0,7,151.0,3,9,8,0,237264511,67861120,0,1.0,7,3,2,237264510,6,"Z0tJTOQ",1,5,413734,0,1079813,"WVC3i",466098,48.0,4,0,0,21799474,"WVC3h",7643764,10.0,4,0,0,52080155,"WVC3g",18812812,13.0,4,3878862,0,9648045,"WVC3f",3739632,28.0,4,0,0,59630308,"WVC3e",21649514,52.0,4,5,2,67861120,0,0,3,15,0,2,0,0,1,["com.cronometer.shared.foods.models.Food/2097636843","java.util.ArrayList/4159755760","","com.cronometer.shared.foods.models.Ingredient/1280520736","com.cronometer.shared.foods.NutritionLabelType/1598919019","com.cronometer.shared.foods.models.FoodMeasures/2106205728","com.cronometer.shared.foods.models.Measure/824760657","g","com.cronometer.shared.foods.models.Measure\u0024Type/2365167904","full recipe","Serving","com.cronometer.shared.foods.models.NutrientMap/168231382","com.cronometer.shared.foods.models.NutrientMap\u0024NutrientFilter/1990310964","java.util.HashMap/1797211028","java.lang.Integer/3438268394","com.cronometer.shared.foods.models.Nutrient/331784102","com.cronometer.shared.foods.models.Nutrient\u0024Type/4187872513","java.lang.String/2004016611","advancedServingSize","false","Custom","java.util.HashSet/3273092938","com.cronometer.shared.foods.models.Translation/4034452093","com.cronometer.shared.user.models.Language/1257207975","en","English","https://cdn1.cronometer.com/media/flags/us.png","Basic Sandwich","com.cronometer.shared.foods.FoodType/2323555378"],0,7]`

// Actual getFood(466098) response — Lettuce, Green Leaf (NCCDB)
const lettuceResponse = "//OK[0,0,43,12417149,42,41,40,39,38,28,27,2374339,37,36,35,34,33,28,27,413734,32,30,31,30,29,28,27,3,2,0,26,5,26,6,26,3,25,24,0,0,20,-21,511,0.057,22,511,21,-21,255,94.01,22,255,21,-21,510,0.056,22,510,21,-21,509,0.026,22,509,21,-21,508,0.044,22,508,21,-21,507,0.013,22,507,21,-21,506,0.013,22,506,21,-21,505,0.067,22,505,21,-21,504,0.063,22,504,21,-21,503,0.067,22,503,21,-21,502,0.047,22,502,21,-21,501,0.007,22,501,21,-21,606,0.021,22,606,21,-21,605,0.0,22,605,21,-21,-221,0.0,22,-221,21,-21,601,0.0,22,601,21,-21,343,0.02,22,343,21,-21,342,0.41,22,342,21,-21,214,0.0,22,214,21,-21,853,0.0,22,853,21,-21,341,0.0,22,341,21,-21,213,0.0,22,213,21,-21,212,0.43,22,212,21,-21,851,0.062,22,851,21,-21,211,0.36,22,211,21,-21,338,1730.0,22,338,21,-21,210,0.0,22,210,21,-21,337,0.0,22,337,21,-21,209,0.0,22,209,21,-21,208,18.0,22,208,21,-21,207,0.67,22,207,21,-21,334,0.0,22,334,21,-21,205,4.07,22,205,21,-21,204,0.16,22,204,21,-21,-205,14.529959999999999,22,-205,21,-21,-204,1.3392000000000002,22,-204,21,-21,-203,2.6596800000000003,22,-203,21,-21,326,0.0,22,326,21,-21,325,0.0,22,325,21,-21,324,0.0,22,324,21,-21,323,0.22,22,323,21,-21,322,0.0,22,322,21,-21,321,4443.0,22,321,21,-21,320,370.25,22,320,21,-21,319,0.0,22,319,21,-21,318,7405.0,22,318,21,-21,317,0.6,22,317,21,-21,315,0.156,22,315,21,-21,312,0.04,22,312,21,-21,309,0.31,22,309,21,-21,-1205,2.7460000000000004,22,-1205,21,-21,307,29.0,22,307,21,-21,306,277.0,22,306,21,-21,305,27.0,22,305,21,-21,304,13.0,22,304,21,-21,303,0.32,22,303,21,-21,430,118.5,22,430,21,-21,301,40.0,22,301,21,-21,297,1.3,22,297,21,-21,295,0.0,22,295,21,-21,421,13.6,22,421,21,-21,675,0.0,22,675,21,-21,291,1.3,22,291,21,-21,418,0.0,22,418,21,-21,417,38.0,22,417,21,-21,415,0.071,22,415,21,-21,287,0.01,22,287,21,-21,10012,0.3,22,10012,21,-21,410,0.134,22,410,21,-21,10009,0.0,22,10009,21,-21,10007,0.024,22,10007,21,-21,406,0.375,22,406,21,-21,10005,1.2,22,10005,21,-21,405,0.08,22,405,21,-21,404,0.082,22,404,21,-21,10002,0.0,22,10002,21,-21,10001,0.062,22,10001,21,-21,401,15.2,22,401,21,-21,269,0.78,22,269,21,-21,646,0.087,22,646,21,-21,518,0.031,22,518,21,-21,262,0.0,22,262,21,-21,645,0.006,22,645,21,-21,517,0.038,22,517,21,-21,516,0.046,22,516,21,-21,515,0.146,22,515,21,-21,514,0.114,22,514,21,-21,513,0.045,22,513,21,0,23,512,0.018,22,512,21,94,20,0,19,18,0.7501478140000001,-8,17,0,1079815,466098,0,1.0,8,1.0,-8,16,0,1079819,466098,0,1.0,8,2.250445936,-8,15,0,1079816,466098,0,1.0,8,4.8,-8,14,0,1079812,466098,0,1.0,8,24.0,-8,13,0,1079813,466098,0,1.0,8,28.3495231,-8,12,0,1080738,466098,0,1.0,8,36.007125,-8,11,0,1079817,466098,0,1.0,8,360.0,0,10,9,0,1079814,466098,0,1.0,8,8,2,1079813,7,\"Zjjf7wA\",0,6,0,466098,0,0,5,26,4,3,1,2,0,0,1,[\"com.cronometer.shared.foods.models.Food/2097636843\",\"java.util.ArrayList/4159755760\",\"java.lang.String/2004016611\",\"4076\",\"vegetables, lettuce, green leaf\",\"com.cronometer.shared.foods.NutritionLabelType/1598919019\",\"com.cronometer.shared.foods.models.FoodMeasures/2106205728\",\"com.cronometer.shared.foods.models.Measure/824760657\",\"head\",\"com.cronometer.shared.foods.models.Measure\\u0024Type/2365167904\",\"cup, chopped\",\"oz\",\"outer leaf - large\",\"inner leaf - small\",\"tbsp, chopped\",\"g\",\"tsp, chopped\",\"com.cronometer.shared.foods.models.NutrientMap/168231382\",\"com.cronometer.shared.foods.models.NutrientMap\\u0024NutrientFilter/1990310964\",\"java.util.HashMap/1797211028\",\"java.lang.Integer/3438268394\",\"com.cronometer.shared.foods.models.Nutrient/331784102\",\"com.cronometer.shared.foods.models.Nutrient\\u0024Type/4187872513\",\"NCCDB:13930\",\"java.util.HashSet/3273092938\",\"com.cronometer.shared.foods.FoodTag/3220417118\",\"com.cronometer.shared.foods.models.Translation/4034452093\",\"com.cronometer.shared.user.models.Language/1257207975\",\"en\",\"English\",\"https://cdn1.cronometer.com/media/flags/us.png\",\"Lettuce, Green Leaf\",\"fr\",\"French\",\"https://cdn1.cronometer.com/media/flags/fr.png\",\"Fran\\u00e7ais\",\"Laitue Fris\\u00e9e\",\"de\",\"German\",\"https://cdn1.cronometer.com/media/flags/de.png\",\"Deutsch\",\"Kopfsalat, Gr\\u00fcnes Blatt\",\"com.cronometer.shared.foods.FoodType/2323555378\"],0,7]"

func TestDeserializeFood_Lettuce(t *testing.T) {
	r, err := NewGWTReader(lettuceResponse)
	if err != nil {
		t.Fatalf("NewGWTReader: %v", err)
	}

	// Read the Food type signature
	typeSig := r.ReadObject()
	if !strings.Contains(typeSig, "Food") {
		t.Fatalf("expected Food type, got %q", typeSig)
	}

	food, err := DeserializeFood(r)
	if err != nil {
		t.Fatalf("DeserializeFood: %v", err)
	}

	t.Logf("Food: ID=%d Name=%q Source=%q FoodType=%d", food.ID, food.Name, food.Source, food.FoodType)
	t.Logf("Nutrients: %d, Ingredients: %d, Measures: %d, Translations: %d",
		len(food.Nutrients), len(food.Ingredients), len(food.Measures), len(food.Translations))
	t.Logf("Remaining tokens: %d", r.Remaining())

	// Verify Food ID
	if food.ID != 466098 {
		t.Errorf("ID: got %d, want 466098", food.ID)
	}

	// Verify name (from English translation)
	if food.Name != "Lettuce, Green Leaf" {
		t.Errorf("Name: got %q, want %q", food.Name, "Lettuce, Green Leaf")
	}

	// Verify source
	if food.Source != "NCCDB:13930" {
		t.Errorf("Source: got %q, want %q", food.Source, "NCCDB:13930")
	}

	// Should have no ingredients (simple food, not a recipe)
	if len(food.Ingredients) != 0 {
		t.Errorf("expected 0 ingredients for simple food, got %d", len(food.Ingredients))
	}

	// Should have nutrients
	if len(food.Nutrients) == 0 {
		t.Error("expected nutrients, got none")
	} else if cal, ok := food.Nutrients[208]; !ok || cal != 18.0 {
		t.Errorf("Energy (208): got %v (exists=%v), want 18.0", cal, ok)
	}

	// Should have 3 translations (en, fr, de)
	if len(food.Translations) != 3 {
		t.Errorf("expected 3 translations, got %d: %v", len(food.Translations), food.Translations)
	}

	// Should have measures (head, cup chopped, oz, etc.)
	if len(food.Measures) < 5 {
		t.Errorf("expected >= 5 measures, got %d", len(food.Measures))
	}

	// Verify remaining tokens consumed
	if r.Remaining() != 0 {
		t.Errorf("expected 0 remaining tokens, got %d", r.Remaining())
	}
}

func TestDeserializeFood_BasicSandwich(t *testing.T) {
	r, err := NewGWTReader(basicSandwichResponse)
	if err != nil {
		t.Fatalf("NewGWTReader: %v", err)
	}

	typeSig := r.ReadObject()
	if !strings.Contains(typeSig, "Food") {
		t.Fatalf("expected Food type, got %q", typeSig)
	}

	food, err := DeserializeFood(r)
	if err != nil {
		t.Fatalf("DeserializeFood: %v", err)
	}

	t.Logf("Food: ID=%d Name=%q Source=%q FoodType=%d UserID=%d",
		food.ID, food.Name, food.Source, food.FoodType, food.UserID)
	t.Logf("Nutrients: %d, Ingredients: %d, Measures: %d, Translations: %d",
		len(food.Nutrients), len(food.Ingredients), len(food.Measures), len(food.Translations))
	for i, ing := range food.Ingredients {
		t.Logf("  Ingredient %d: foodID=%d, amount=%.1f, measureID=%d", i, ing.FoodID, ing.Amount, ing.MeasureID)
	}
	for i, m := range food.Measures {
		t.Logf("  Measure %d: name=%q, grams=%.1f, id=%d", i, m.Name, m.Grams, m.ID)
	}
	t.Logf("Remaining tokens: %d", r.Remaining())

	// Verify Food ID
	if food.ID != 67861120 {
		t.Errorf("ID: got %d, want 67861120", food.ID)
	}

	// Verify name (from English translation)
	if food.Name != "Basic Sandwich" {
		t.Errorf("Name: got %q, want %q", food.Name, "Basic Sandwich")
	}

	// Verify source
	if food.Source != "Custom" {
		t.Errorf("Source: got %q, want %q", food.Source, "Custom")
	}

	// Verify 5 ingredients with known food IDs and amounts
	if len(food.Ingredients) != 5 {
		t.Errorf("expected 5 ingredients, got %d", len(food.Ingredients))
	} else {
		expected := []struct {
			foodID int
			amount float64
		}{
			{21649514, 52.0},  // Nature's Harvest Bread
			{3739632, 28.0},   // Arla Havarti Cheese
			{18812812, 13.0},  // Best Foods Mayo
			{7643764, 10.0},   // French's Mustard
			{466098, 48.0},    // Lettuce Green Leaf
		}
		for i, exp := range expected {
			if food.Ingredients[i].FoodID != exp.foodID {
				t.Errorf("Ingredient %d FoodID: got %d, want %d", i, food.Ingredients[i].FoodID, exp.foodID)
			}
			if food.Ingredients[i].Amount != exp.amount {
				t.Errorf("Ingredient %d Amount: got %.1f, want %.1f", i, food.Ingredients[i].Amount, exp.amount)
			}
		}
	}

	// Verify energy nutrient (208 = 328.64 kcal per full recipe, but NutrientMap is per 100g)
	// Per 100g of 151g recipe: 328.64 / 151 * 100 ≈ 217.6... actually the map stores
	// the values as-is from the response. Let's just check it exists.
	if _, ok := food.Nutrients[208]; !ok {
		t.Error("missing energy nutrient (code 208)")
	} else {
		t.Logf("Energy (208) = %.2f", food.Nutrients[208])
	}

	// Verify measures: "g" (151g), "full recipe" (1.0), "Serving" (1.0)
	if len(food.Measures) != 3 {
		t.Errorf("expected 3 measures, got %d", len(food.Measures))
	}

	// Verify all tokens consumed
	if r.Remaining() != 0 {
		t.Errorf("expected 0 remaining tokens, got %d", r.Remaining())
	}
}

// getAllFood batch response — 5 ingredient foods (4 CRDB + 1 NCCDB)
// Captured via Playwright from Basic Sandwich recipe view
const getAllFoodResponse = `//OK[0,-70,12417149,88,87,86,85,84,25,24,2374339,83,82,81,80,79,25,24,413734,78,27,28,27,26,25,24,3,1,-66,0,23,-64,3,22,77,0,0,17,-19,511,0.057,19,511,18,-19,255,94.01,19,255,18,-19,510,0.056,19,510,18,-19,509,0.026,19,509,18,-19,508,0.044,19,508,18,-19,507,0.013,19,507,18,-19,506,0.013,19,506,18,-19,505,0.067,19,505,18,-19,504,0.063,19,504,18,-19,503,0.067,19,503,18,-19,502,0.047,19,502,18,-19,246,2.4,19,246,18,-19,629,0.0,19,629,18,-19,501,0.007,19,501,18,-19,621,0.0,19,621,18,-19,606,0.021,19,606,18,-19,605,0.0,19,605,18,-19,221,0.0,19,221,18,-19,-221,0.0,19,-221,18,-19,601,0.0,19,601,18,-19,343,0.02,19,343,18,-19,342,0.41,19,342,18,-19,214,0.0,19,214,18,-19,853,0.0,19,853,18,-19,341,0.0,19,341,18,-19,213,0.0,19,213,18,-19,212,0.43,19,212,18,-19,851,0.062,19,851,18,-19,211,0.36,19,211,18,-19,338,1730.0,19,338,18,-19,210,0.0,19,210,18,-19,337,0.0,19,337,18,-19,209,0.0,19,209,18,-19,208,18.0,19,208,18,-19,207,0.67,19,207,18,-19,334,0.0,19,334,18,-19,205,4.07,19,205,18,-19,204,0.16,19,204,18,-19,-205,14.529959999999999,19,-205,18,-19,203,1.09,19,203,18,-19,-204,1.3392000000000002,19,-204,18,-19,-203,2.6596800000000003,19,-203,18,-19,326,0.0,19,326,18,-19,325,0.0,19,325,18,-19,324,0.0,19,324,18,-19,323,0.22,19,323,18,-19,322,0.0,19,322,18,-19,321,4443.0,19,321,18,-19,320,370.25,19,320,18,-19,319,0.0,19,319,18,-19,318,7405.0,19,318,18,-19,317,0.6,19,317,18,-19,315,0.156,19,315,18,-19,312,0.04,19,312,18,-19,309,0.31,19,309,18,-19,-1205,2.7460000000000004,19,-1205,18,-19,307,29.0,19,307,18,-19,306,277.0,19,306,18,-19,305,27.0,19,305,18,-19,304,13.0,19,304,18,-19,303,0.32,19,303,18,-19,430,118.5,19,430,18,-19,301,40.0,19,301,18,-19,297,1.3,19,297,18,-19,295,0.0,19,295,18,-19,421,13.6,19,421,18,-19,675,0.0,19,675,18,-19,291,1.3,19,291,18,-19,418,0.0,19,418,18,-19,417,38.0,19,417,18,-19,415,0.071,19,415,18,-19,287,0.01,19,287,18,-19,10012,0.3,19,10012,18,-19,410,0.134,19,410,18,-19,10009,0.0,19,10009,18,-19,10007,0.024,19,10007,18,-19,406,0.375,19,406,18,-19,10005,1.2,19,10005,18,-19,405,0.08,19,405,18,-19,404,0.082,19,404,18,-19,10002,0.0,19,10002,18,-19,10001,0.062,19,10001,18,-19,401,15.2,19,401,18,-19,269,0.78,19,269,18,-19,646,0.087,19,646,18,-19,518,0.031,19,518,18,-19,262,0.0,19,262,18,-19,645,0.006,19,645,18,-19,517,0.038,19,517,18,-19,516,0.046,19,516,18,-19,515,0.146,19,515,18,-19,514,0.114,19,514,18,-19,513,0.045,19,513,18,-19,512,0.018,19,512,18,94,17,-15,15,0.7501478140000001,-10,76,0,1079815,466098,0,1.0,9,1.0,-10,14,0,1079819,466098,0,1.0,9,2.250445936,-10,75,0,1079816,466098,0,1.0,9,4.8,-10,74,0,1079812,466098,0,1.0,9,24.0,-10,73,0,1079813,466098,0,1.0,9,28.3495231,-10,12,0,1080738,466098,0,1.0,9,36.007125,-10,72,0,1079817,466098,0,1.0,9,360.0,-10,71,0,1079814,466098,0,1.0,9,8,1,1079819,8,"Zjjf7wA",0,7,0,466098,0,0,70,26,69,3,1,1,0,0,2,0,-70,7999084,68,27,28,27,26,25,24,1,1,-66,-65,-64,3,22,21,0,0,17,-19,606,0.0,19,606,18,-19,605,0.0,19,605,18,-19,-221,0.0,19,-221,18,-19,10009,0.0,19,10009,18,-19,601,0.0,19,601,18,-19,-1205,0.0,19,-1205,18,-19,307,1100.0,19,307,18,-19,306,0.0,19,306,18,-19,208,0.0,19,208,18,-19,303,0.0,19,303,18,-19,301,0.0,19,301,18,-19,269,0.0,19,269,18,-19,205,0.0,19,205,18,-19,204,0.0,19,204,18,-19,-205,0.0,19,-205,18,-19,203,0.0,19,203,18,-19,-204,0.0,19,-204,18,-19,-203,0.0,19,-203,18,-19,324,0.0,19,324,18,-19,291,0.0,19,291,18,20,17,-15,15,1.0,-10,14,0,19475679,7643764,0,1.0,9,5.0,-10,67,0,21799474,7643764,0,1.0,9,28.3495231,-10,12,0,21799523,7643764,0,1.0,9,3,1,21799474,8,"ZwOKoq4",-6,0,7643764,0,0,6,23,66,3,65,3,64,3,63,3,62,3,61,3,60,3,59,3,58,3,57,3,56,3,55,3,54,3,53,3,52,3,51,3,50,3,49,3,48,3,47,3,46,3,45,3,44,3,23,1,0,0,2,0,-70,20626175,43,27,28,27,26,25,24,1,1,-65,-64,2,22,21,0,0,17,-19,606,11.538461538461537,19,606,18,-19,605,0.0,19,605,18,-19,-221,0.0,19,-221,18,-19,10009,0.0,19,10009,18,-19,601,38.46153846153846,19,601,18,-19,-1205,0.0,19,-1205,18,-19,307,692.3076923076923,19,307,18,-19,306,0.0,19,306,18,-19,208,692.3076923076923,19,208,18,-19,303,0.0,19,303,18,-19,301,0.0,19,301,18,-19,269,0.0,19,269,18,-19,205,0.0,19,205,18,-19,-205,0.0,19,-205,18,-19,204,76.92307692307692,19,204,18,-19,-204,692.3076923076923,19,-204,18,-19,203,0.0,19,203,18,-19,-203,0.0,19,-203,18,-19,646,46.153846153846146,19,646,18,-19,645,19.23076923076923,19,645,18,-19,324,0.0,19,324,18,-19,291,0.0,19,291,18,22,17,-15,15,1.0,-10,14,0,52080153,18812812,0,1.0,9,13.0,-10,42,0,52080155,18812812,0,1.0,9,28.3495231,-10,12,0,52080154,18812812,0,1.0,9,3,1,52080155,8,"ZfUfKUg",-6,0,18812812,0,0,6,10,41,3,40,3,39,3,38,3,37,3,36,3,35,3,34,3,33,3,9,1,0,0,2,0,-70,3878862,32,27,28,27,26,25,24,1,1,-65,7,23,-64,3,22,21,0,0,17,-19,606,21.42857142857143,19,606,18,-19,605,0.0,19,605,18,-19,-221,0.0,19,-221,18,-19,10009,0.0,19,10009,18,-19,601,107.14285714285714,19,601,18,-19,-1205,0.0,19,-1205,18,-19,307,678.5714285714286,19,307,18,-19,306,35.71428571428571,19,306,18,-19,208,392.8571428571429,19,208,18,-19,303,0.0,19,303,18,-19,301,642.8571428571429,19,301,18,-19,269,0.0,19,269,18,-19,205,0.0,19,205,18,-19,-205,0.0,19,-205,18,-19,204,32.14285714285715,19,204,18,-19,-204,303.06122448979596,19,-204,18,-19,203,21.42857142857143,19,203,18,-19,-203,89.79591836734693,19,-203,18,-19,324,0.0,19,324,18,-19,291,0.0,19,291,18,20,17,-15,15,1.0,-10,14,0,9648046,3739632,0,1.0,9,28.0,-10,13,0,9648045,3739632,0,1.0,9,28.3495231,-10,12,0,9648658,3739632,0,1.0,9,3,1,9648045,8,"WG14WTY",-6,0,3739632,0,0,6,7,31,3,1,1,0,0,2,0,0,30,23644934,29,27,28,27,26,25,24,1,1,6,23,1,23,5,23,3,22,21,0,0,17,-19,606,0.0,19,606,18,-19,605,0.0,19,605,18,-19,-221,0.0,19,-221,18,-19,10009,3.846153846153846,19,10009,18,-19,601,0.0,19,601,18,-19,-1205,38.46153846153846,19,-1205,18,-19,307,461.53846153846155,19,307,18,-19,306,230.76923076923077,19,306,18,-19,208,230.76923076923077,19,208,18,-19,303,2.3076923076923075,19,303,18,-19,301,115.38461538461539,19,301,18,-19,269,3.846153846153846,19,269,18,-19,205,46.15384615384615,19,205,18,-19,204,3.846153846153846,19,204,18,-19,-205,151.32408575031525,19,-205,18,-19,203,11.538461538461538,19,203,18,-19,-204,34.04791929382093,19,-204,18,-19,-203,45.39722572509458,19,-203,18,-19,646,0.0,19,646,18,-19,645,0.0,19,645,18,-19,324,0.0,19,324,18,0,20,291,7.692307692307692,19,291,18,22,17,0,16,15,1.0,-10,14,0,59630310,21649514,0,1.0,9,26.0,-10,13,0,59630308,21649514,0,1.0,9,28.3495231,-10,12,0,59630309,21649514,0,1.0,9,56.333,0,11,10,0,87141750,21649514,0,2.0,9,4,1,59630308,8,"Zdb5jUY",1,7,0,21649514,0,0,6,2,5,3,4,3,2,1,0,0,2,5,1,["java.util.ArrayList/4159755760","com.cronometer.shared.foods.models.Food/2097636843","java.lang.String/2004016611","78700801623","78700801685","","com.cronometer.shared.foods.NutritionLabelType/1598919019","com.cronometer.shared.foods.models.FoodMeasures/2106205728","com.cronometer.shared.foods.models.Measure/824760657","slices","com.cronometer.shared.foods.models.Measure\u0024Type/2365167904","oz","slice","g","com.cronometer.shared.foods.models.NutrientMap/168231382","com.cronometer.shared.foods.models.NutrientMap\u0024NutrientFilter/1990310964","java.util.HashMap/1797211028","java.lang.Integer/3438268394","com.cronometer.shared.foods.models.Nutrient/331784102","com.cronometer.shared.foods.models.Nutrient\u0024Type/4187872513","CRDB","java.util.HashSet/3273092938","com.cronometer.shared.foods.FoodTag/3220417118","com.cronometer.shared.foods.models.Translation/4034452093","com.cronometer.shared.user.models.Language/1257207975","en","English","https://cdn1.cronometer.com/media/flags/us.png","Nature\u0027s Harvest, Bread, 100% Whole Wheat Bread","com.cronometer.shared.foods.FoodType/2323555378","654858702816","Arla, Havarti Cheese Slices","42001813364","48001016972","48001213517","64366422","68619349","68619351","68619352","69793831","69982985","tbsp","Best Foods, Real Mayonnaise, 90 Cal","2047708000312","41500000251","41500000312","41500003009","41500007007","41500010397","41500756776","41500763682","41500766034","41500810713","41500819389","41500853987","41500853994","41500951614","41500954035","41500965550","4151037","56200761142","56200762163","56200762170","56200824861","6038300000257","9300631990840","tsp","French's, Classic Yellow Mustard","4076","vegetables, lettuce, green leaf","head","cup, chopped","outer leaf - large","inner leaf - small","tbsp, chopped","tsp, chopped","NCCDB:13930","Lettuce, Green Leaf","fr","French","https://cdn1.cronometer.com/media/flags/fr.png","Fran\u00e7ais","Laitue Fris\u00e9e","de","German","https://cdn1.cronometer.com/media/flags/de.png","Deutsch","Kopfsalat, Gr\u00fcnes Blatt"],0,7]`

func TestDeserializeFood_BatchCRDB(t *testing.T) {
	r, err := NewGWTReader(getAllFoodResponse)
	if err != nil {
		t.Fatalf("NewGWTReader: %v", err)
	}

	// Outer wrapper: ArrayList of Food objects
	outerType := r.ReadObject()
	if !strings.Contains(outerType, "ArrayList") {
		t.Fatalf("expected ArrayList, got %q", outerType)
	}
	count := r.ReadInt()
	t.Logf("Batch: %d foods (remaining: %d)", count, r.Remaining())

	var foods []*GWTFood
	for i := 0; i < count; i++ {
		foodType := r.ReadObject()
		if foodType == "" || !strings.Contains(foodType, "Food") {
			t.Fatalf("food %d: expected Food type, got %q", i, foodType)
		}
		food, err := DeserializeFood(r)
		if err != nil {
			t.Fatalf("food %d: %v", i, err)
		}
		foods = append(foods, food)
		t.Logf("  [%d] ID=%d Name=%q Source=%q Nutrients=%d Measures=%d Barcodes=%d Tags=%d Translations=%d",
			i, food.ID, food.Name, food.Source,
			len(food.Nutrients), len(food.Measures),
			0, // barcodes not stored yet
			0, // tags not stored yet
			len(food.Translations))
	}

	t.Logf("Remaining tokens: %d", r.Remaining())

	// Verify we got all 5 foods
	if len(foods) != 5 {
		t.Fatalf("expected 5 foods, got %d", len(foods))
	}

	// Food 0: Nature's Harvest Bread (CRDB)
	if foods[0].ID != 21649514 {
		t.Errorf("food 0 ID: got %d, want 21649514", foods[0].ID)
	}
	if foods[0].Source != "CRDB" {
		t.Errorf("food 0 Source: got %q, want CRDB", foods[0].Source)
	}
	if foods[0].Name != "Nature's Harvest, Bread, 100% Whole Wheat Bread" {
		t.Errorf("food 0 Name: got %q", foods[0].Name)
	}

	// Food 4: Lettuce (NCCDB) — already tested standalone
	if foods[4].ID != 466098 {
		t.Errorf("food 4 ID: got %d, want 466098", foods[4].ID)
	}
	if foods[4].Source != "NCCDB:13930" {
		t.Errorf("food 4 Source: got %q, want NCCDB:13930", foods[4].Source)
	}

	// All tokens consumed
	if r.Remaining() != 0 {
		t.Errorf("expected 0 remaining tokens, got %d", r.Remaining())
	}
}
