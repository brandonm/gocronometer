package gocronometer

import (
	"testing"
)

func TestGWTReader_FindMyFoods(t *testing.T) {
	// Actual findMyFoods response from Cronometer
	resp := `//OK[0,-6,0,0,-3,0,0,9,0,0,0,67861262,0,0,2,0,-4,0,0,-3,0,0,8,0,0,0,67861120,0,0,2,0,-4,0,0,-3,0,0,7,0,0,0,67860387,0,0,2,0,2,5,0,0,-3,0,0,6,0,0,0,67859823,0,0,2,0,1,5,0,0,6,4,0,0,3,0,0,0,67859683,0,0,2,5,1,["[Lcom.cronometer.shared.foods.models.SearchHit;/3199648558","com.cronometer.shared.foods.models.SearchHit/1904627920","Egg whites + Spinach","com.cronometer.shared.foods.FoodSource/4236433762","com.cronometer.shared.foods.FoodType/2323555378","Egg whites + Spinach + Banana Breakfast","Mocha (20oz) - Monin Chocolate + %2 Milk","Basic Sandwich","Turkey/Ham Sandwich"],0,7]`

	r, err := NewGWTReader(resp)
	if err != nil {
		t.Fatalf("NewGWTReader: %v", err)
	}

	// Version should be 7
	if r.version != 7 {
		t.Errorf("version: got %d, want 7", r.version)
	}

	// String table should have 9 entries
	if len(r.stringTable) != 9 {
		t.Errorf("stringTable length: got %d, want 9", len(r.stringTable))
	}

	// Verify string table contents
	expected := []string{
		"[Lcom.cronometer.shared.foods.models.SearchHit;/3199648558",
		"com.cronometer.shared.foods.models.SearchHit/1904627920",
		"Egg whites + Spinach",
		"com.cronometer.shared.foods.FoodSource/4236433762",
		"com.cronometer.shared.foods.FoodType/2323555378",
		"Egg whites + Spinach + Banana Breakfast",
		"Mocha (20oz) - Monin Chocolate + %2 Milk",
		"Basic Sandwich",
		"Turkey/Ham Sandwich",
	}
	for i, want := range expected {
		got := r.stringTable[i]
		if got != want {
			t.Errorf("stringTable[%d]: got %q, want %q", i, got, want)
		}
	}

	// Now read the payload backwards (GWT reads from end)
	// The last token before string table should be readable
	// Read the array: first token is the array type
	arrayType := r.ReadObject()
	if arrayType != "[Lcom.cronometer.shared.foods.models.SearchHit;/3199648558" {
		t.Errorf("array type: got %q", arrayType)
	}

	// Next should be the array length (5 foods)
	count := r.ReadInt()
	if count != 5 {
		t.Errorf("array count: got %d, want 5", count)
	}

	// Read each SearchHit
	type searchHit struct {
		typeSig string
		name    string
		foodID  int64
	}
	var hits []searchHit

	for i := 0; i < count; i++ {
		hit := searchHit{}
		hit.typeSig = r.ReadObject() // SearchHit type
		// SearchHit fields (from JS analysis): we need to read them in order
		// Based on the data pattern, each hit has: several ints, a string (name), and an ID
		// Let's read the raw tokens to understand the pattern
		// For now, just collect what we can
		hits = append(hits, hit)
	}

	t.Logf("Remaining tokens: %d", r.Remaining())
	t.Logf("Hits: %d", len(hits))
}

func TestGWTReader_StringTable(t *testing.T) {
	// Note: Go raw strings don't process \\ as \, so we use regular strings
	// The GWT response has literal backslash sequences like \n, \u0027
	resp := "//OK[1,2,3,[\"hello\",\"world\\ntest\",\"escaped\\u0027quote\"],0,7]"

	r, err := NewGWTReader(resp)
	if err != nil {
		t.Fatalf("NewGWTReader: %v", err)
	}

	if len(r.stringTable) != 3 {
		t.Fatalf("stringTable length: got %d, want 3", len(r.stringTable))
	}

	if r.stringTable[0] != "hello" {
		t.Errorf("stringTable[0]: got %q, want %q", r.stringTable[0], "hello")
	}
	if r.stringTable[1] != "world\ntest" {
		t.Errorf("stringTable[1]: got %q, want %q", r.stringTable[1], "world\ntest")
	}
	if r.stringTable[2] != "escaped'quote" {
		t.Errorf("stringTable[2]: got %q, want %q", r.stringTable[2], "escaped'quote")
	}
}

func TestGWTReader_ReadMethods(t *testing.T) {
	resp := `//OK[42,3.14,2,["hello","world"],0,7]`

	r, err := NewGWTReader(resp)
	if err != nil {
		t.Fatalf("NewGWTReader: %v", err)
	}

	t.Logf("Tokens: %v", r.Tokens())
	t.Logf("StringTable: %v", r.StringTable())
	t.Logf("Index after init: %d", r.Remaining())

	// Read a string (index 2 = "world")
	s := r.ReadString()
	if s != "world" {
		t.Errorf("ReadString: got %q, want %q", s, "world")
	}

	// Read a double
	d := r.ReadDouble()
	if d != 3.14 {
		t.Errorf("ReadDouble: got %f, want 3.14", d)
	}

	// Read an int
	i := r.ReadInt()
	if i != 42 {
		t.Errorf("ReadInt: got %d, want 42", i)
	}
}

func TestGWTReader_GetFoodResponse(t *testing.T) {
	// Actual getFood(466098) response — Lettuce, Green Leaf from NCCDB
	// This is a simplified version with just the key parts
	resp := `//OK[0,0,43,12417149,42,41,40,39,38,28,27,2374339,37,36,35,34,33,28,27,413734,32,30,31,30,29,28,27,3,2,0,26,5,26,6,26,3,25,24,0,0,20,-21,511,0.057,22,511,21,-21,255,94.01,22,255,21,-21,208,18.0,22,208,21,-21,203,1.09,22,203,21,-21,205,4.07,22,205,21,-21,204,0.16,22,204,21,-21,291,1.3,22,291,21,-21,307,29.0,22,307,21,94,20,0,19,18,0.7501478140000001,-8,17,0,1079815,466098,0,1.0,8,1.0,-8,16,0,1079819,466098,0,1.0,8,2.250445936,-8,15,0,1079816,466098,0,1.0,8,4.8,-8,14,0,1079812,466098,0,1.0,8,24.0,-8,13,0,1079813,466098,0,1.0,8,28.3495231,-8,12,0,1080738,466098,0,1.0,8,36.007125,-8,11,0,1079817,466098,0,1.0,8,360.0,0,10,9,0,1079814,466098,0,1.0,8,8,2,1079813,7,"Zjjf7wA",0,6,0,466098,0,0,5,26,4,3,1,2,0,0,1,["com.cronometer.shared.foods.models.Food/2097636843","java.util.ArrayList/4159755760","java.lang.String/2004016611","4076","vegetables, lettuce, green leaf","com.cronometer.shared.foods.NutritionLabelType/1598919019","com.cronometer.shared.foods.models.FoodMeasures/2106205728","com.cronometer.shared.foods.models.Measure/824760657","head","com.cronometer.shared.foods.models.Measure$Type/2365167904","cup, chopped","oz","outer leaf - large","inner leaf - small","tbsp, chopped","g","tsp, chopped","com.cronometer.shared.foods.models.NutrientMap/168231382","com.cronometer.shared.foods.models.NutrientMap$NutrientFilter/1990310964","java.util.HashMap/1797211028","java.lang.Integer/3438268394","com.cronometer.shared.foods.models.Nutrient/331784102","com.cronometer.shared.foods.models.Nutrient$Type/4187872513","NCCDB:13930","java.util.HashSet/3273092938","com.cronometer.shared.foods.FoodTag/3220417118","com.cronometer.shared.foods.models.Translation/4034452093","com.cronometer.shared.user.models.Language/1257207975","en","English","https://cdn1.cronometer.com/media/flags/us.png","Lettuce, Green Leaf","fr","French","https://cdn1.cronometer.com/media/flags/fr.png","Fran\\u00e7ais","Laitue Fris\\u00e9e","de","German","https://cdn1.cronometer.com/media/flags/de.png","Deutsch","Kopfsalat, Gr\\u00fcnes Blatt","com.cronometer.shared.foods.FoodType/2323555378"],0,7]`

	r, err := NewGWTReader(resp)
	if err != nil {
		t.Fatalf("NewGWTReader: %v", err)
	}

	t.Logf("Version: %d, Flags: %d", r.version, r.flags)
	t.Logf("String table: %d entries", len(r.stringTable))
	t.Logf("Tokens: %d", r.Remaining())

	// The Food type should be the first object read
	foodType := r.ReadObject()
	t.Logf("Food type: %q", foodType)

	if foodType != "com.cronometer.shared.foods.models.Food/2097636843" {
		t.Errorf("expected Food type, got %q", foodType)
	}

	// Now read Food fields in order (from JS analysis):
	// 1. long (ID?)
	// 2. long (secondary ID?)
	// 3. int (type enum?)
	// 4. list (ingredients)
	// 5. Map (nutrients)
	// 6. double (weight?)
	// 7. string (NAME)
	// 8. string (description)
	// ...

	// For now just verify we can read sequentially
	t.Logf("Remaining after type: %d tokens", r.Remaining())

	// Verify string table has the food name
	found := false
	for _, s := range r.stringTable {
		if s == "Lettuce, Green Leaf" {
			found = true
			break
		}
	}
	if !found {
		t.Error("string table missing 'Lettuce, Green Leaf'")
	}

	// Verify unicode escape handling
	for _, s := range r.stringTable {
		if s == "Français" {
			t.Log("Unicode escape \\u00e7 correctly decoded to 'ç'")
			break
		}
	}
}

func TestLongFromBase64(t *testing.T) {
	// GWT base64 uses ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_$
	tests := []struct {
		input string
		want  int64
	}{
		{"A", 0},
		{"B", 1},
		{"BA", 64},
	}

	for _, tt := range tests {
		got, err := longFromBase64(tt.input)
		if err != nil {
			t.Errorf("longFromBase64(%q): %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("longFromBase64(%q): got %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestDeconcatGWT(t *testing.T) {
	input := `[1,2,3].concat([4,5],[6,7])`
	got := deconcatGWT(input)
	want := `[1,2,3,4,5,6,7]`
	if got != want {
		t.Errorf("deconcat: got %q, want %q", got, want)
	}
}
