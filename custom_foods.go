package gocronometer

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

// CustomFood represents a custom food/recipe/meal from Cronometer.
type CustomFood struct {
	ID   int64
	Name string
}

// RecipeIngredient represents an ingredient within a custom recipe or meal.
type RecipeIngredient struct {
	FoodID    int64
	MeasureID int64
	Amount    float64
	Name      string // populated after resolving via GetFood or ExportFood
}

// FoodDetail is the full result from GetFood, including the food's own info,
// its ingredients (if a recipe), and per-100g nutrient data.
type FoodDetail struct {
	FoodID      int64
	Name        string
	Source      string // e.g. "NCCDB", "CRDB", "custom"
	Ingredients []RecipeIngredient
	// NutrientsPer100g contains nutrient data keyed by USDA nutrient code.
	// Values are per 100g. Scale by (amount/100) for actual serving.
	NutrientsPer100g map[int]float64
	// Measures maps a measure ID to its weight in grams, used to convert a
	// serving's amount (in measure units, e.g. "1 full recipe") into grams.
	Measures []FoodMeasure
}

// FoodMeasure is a serving-size measure for a food (e.g. "g" = 1g, "full recipe" = 100.7g).
type FoodMeasure struct {
	ID    int64
	Name  string
	Grams float64
}

// FoodExport represents the nutrient data from a food CSV export.
type FoodExport struct {
	FoodID    int64
	FoodName  string
	Amount    string
	Nutrients map[string]float64 // nutrient name -> value (raw from CSV)
}

// USDANutrientNames maps USDA nutrient codes to human-readable names and units.
var USDANutrientNames = map[int]struct{ Name, Unit string }{
	// Every entry below was verified against Cronometer's own food-detail panel
	// by matching cached per-100g values for a known food. Do not add a code on
	// the strength of a USDA table alone: three entries in the first version of
	// this map were wrong for five months because they were never checked
	// against what Cronometer actually returns.

	// General
	208:   {"calories", "kcal"},
	207:   {"ash", "g"},
	255:   {"water", "g"},
	262:   {"caffeine", "mg"},
	10012: {"oxalate", "mg"},

	// Macronutrients
	203: {"protein", "g"},
	204: {"fat", "g"},
	205: {"carbs", "g"},

	// Carbohydrates
	291:   {"fiber", "g"},
	269:   {"sugar", "g"},
	10009: {"added_sugars", "g"},
	// 10007, not 10005 — 10005 is iodine. Cronometer's panel for a food with a
	// non-zero 10007 shows it under "Sugar Alcohol"; 10005 lines up with Iodine.
	10007: {"sugar_alcohol", "g"},

	// Lipids
	606:   {"saturated_fat", "g"},
	645:   {"monounsaturated_fat", "g"},
	646:   {"polyunsaturated_fat", "g"},
	605:   {"trans_fat", "g"},
	601:   {"cholesterol", "mg"},
	10001: {"omega_3", "g"},
	10002: {"omega_6", "g"},
	// Omega-3 and omega-6 subfractions. EPA and DHA are the marine forms that
	// carry the cardiovascular evidence; total omega_3 also contains ALA, which
	// converts poorly, so the total alone cannot answer "am I getting enough".
	629: {"epa", "g"},
	621: {"dha", "g"},
	675: {"la_linoleic", "g"},
	853: {"aa_arachidonic", "g"},

	// Minerals
	307:   {"sodium", "mg"},
	306:   {"potassium", "mg"},
	301:   {"calcium", "mg"},
	303:   {"iron", "mg"},
	304:   {"magnesium", "mg"},
	305:   {"phosphorus", "mg"},
	309:   {"zinc", "mg"},
	312:   {"copper", "mg"},
	315:   {"manganese", "mg"},
	317:   {"selenium", "mcg"},
	10005: {"iodine", "mcg"},

	// Vitamins
	// 320 is vitamin A as RAE, 319 is preformed retinol and 321 is beta-carotene.
	// The relationship RAE = retinol + beta_carotene/12 holds exactly in the
	// data, which is what distinguishes them: a plant food has 320 > 0 with
	// 319 = 0. 318 (vitamin A in IU) is deliberately absent — admitting both it
	// and 320 would sum international units into micrograms.
	320: {"vitamin_a", "mcg"},
	319: {"retinol", "mcg"},
	321: {"beta_carotene", "mcg"},
	401: {"vitamin_c", "mg"},
	324: {"vitamin_d", "IU"},
	323: {"vitamin_e", "mg"},
	430: {"vitamin_k", "mcg"},
	404: {"b1_thiamine", "mg"},
	405: {"b2_riboflavin", "mg"},
	406: {"b3_niacin", "mg"},
	410: {"b5_pantothenic", "mg"},
	415: {"b6", "mg"},
	418: {"b12", "mcg"},
	417: {"folate", "mcg"},
	421: {"choline", "mg"},

	// Amino acids. The block runs 501-518 contiguously in USDA numbering.
	501: {"tryptophan", "g"},
	502: {"threonine", "g"},
	503: {"isoleucine", "g"},
	504: {"leucine", "g"},
	505: {"lysine", "g"},
	506: {"methionine", "g"},
	507: {"cystine", "g"},
	508: {"phenylalanine", "g"},
	509: {"tyrosine", "g"},
	510: {"valine", "g"},
	511: {"arginine", "g"},
	512: {"histidine", "g"},
	513: {"alanine", "g"},
	514: {"aspartic_acid", "g"},
	515: {"glutamic_acid", "g"},
	516: {"glycine", "g"},
	517: {"proline", "g"},
	518: {"serine", "g"},
}

// ParseFoodExport parses a food CSV export into a FoodExport struct.
func ParseFoodExport(csvData string) (*FoodExport, error) {
	r := csv.NewReader(stringReader(csvData))

	headers, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("reading headers: %w", err)
	}

	record, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("reading data row: %w", err)
	}

	if len(record) != len(headers) {
		return nil, fmt.Errorf("header/data length mismatch: %d vs %d", len(headers), len(record))
	}

	export := &FoodExport{
		Nutrients: make(map[string]float64),
	}

	for i, header := range headers {
		value := record[i]
		switch header {
		case "Food ID":
			id, _ := strconv.ParseInt(value, 10, 64)
			export.FoodID = id
		case "Food Name":
			export.FoodName = value
		case "Amount":
			export.Amount = value
		default:
			if value != "" {
				f, err := strconv.ParseFloat(value, 64)
				if err == nil && f != 0 {
					export.Nutrients[header] = f
				}
			}
		}
	}

	return export, nil
}

func stringReader(s string) io.Reader {
	return io.NopCloser(ioStringReader(s))
}

type ioStringReaderType struct {
	s string
	i int
}

func ioStringReader(s string) *ioStringReaderType {
	return &ioStringReaderType{s: s}
}

func (r *ioStringReaderType) Read(p []byte) (n int, err error) {
	if r.i >= len(r.s) {
		return 0, io.EOF
	}
	n = copy(p, r.s[r.i:])
	r.i += n
	return
}

func (r *ioStringReaderType) Close() error {
	return nil
}
