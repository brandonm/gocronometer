package gocronometer

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
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
// its ingredients (if a recipe), and per-100g nutrient data from the GWT response.
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

// FindMyFoods returns all custom foods for the logged-in user.
func (c *Client) FindMyFoods(ctx context.Context) ([]CustomFood, error) {
	reqBody := c.formatGWTRequest(GWTFindMyFoods, c.Nonce, c.UserID)

	req, err := c.NewGWTRequestWithContext(ctx, "POST", GWTBaseURL, strings.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to build findMyFoods request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute findMyFoods request: %w", err)
	}
	defer closeAndExhaustReader(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("findMyFoods returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read findMyFoods response: %w", err)
	}

	return parseFindMyFoodsResponse(string(body))
}

// gwtResponseRegex matches the GWT OK response format: //OK[data,["strings"],flags]
var gwtResponseRegex = regexp.MustCompile(`^//OK\[(.+)\]$`)

// parseFindMyFoodsResponse parses the GWT response from findMyFoods.
// The response contains an array of SearchHit objects with food IDs and names.
//
// Response format: //OK[numeric_data,["string_table"],0,7]
// The string table contains type descriptors and food names.
// Food names are strings that don't contain "/" (type descriptors do).
// Food IDs are large positive integers (>1000000) in the numeric data.
func parseFindMyFoodsResponse(body string) ([]CustomFood, error) {
	// Extract string table
	strings, err := extractGWTStringTable(body)
	if err != nil {
		return nil, fmt.Errorf("parsing string table: %w", err)
	}

	// Extract numeric data (before the string table)
	numericData, err := extractGWTNumericData(body)
	if err != nil {
		return nil, fmt.Errorf("parsing numeric data: %w", err)
	}

	// Food names are strings that aren't type descriptors (don't contain "/" or "[")
	var foodNames []string
	nameIndices := make(map[int]bool) // 1-based string table index -> is food name
	for i, s := range strings {
		if !isGWTTypeDescriptor(s) && s != "" {
			foodNames = append(foodNames, s)
			nameIndices[i+1] = true // GWT uses 1-based indexing
		}
	}

	// Extract food IDs: large positive integers (custom food IDs are > 1,000,000)
	// They appear in the numeric data near name references.
	// Strategy: scan for patterns where a food ID appears near a name reference.
	var foodIDs []int64
	for _, v := range numericData {
		id, err := strconv.ParseInt(v, 10, 64)
		if err == nil && id > 1000000 && id < 1000000000 {
			// Check it's not a measure ID or other large number by looking at context
			foodIDs = append(foodIDs, id)
		}
	}

	// Deduplicate food IDs
	seen := make(map[int64]bool)
	var uniqueIDs []int64
	for _, id := range foodIDs {
		if !seen[id] {
			seen[id] = true
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	// Match IDs to names: they appear in reverse order in the response
	// (last food in string table = first ID in data)
	var foods []CustomFood
	if len(uniqueIDs) == len(foodNames) {
		for i, id := range uniqueIDs {
			// IDs are in reverse order relative to names
			nameIdx := len(foodNames) - 1 - i
			foods = append(foods, CustomFood{
				ID:   id,
				Name: foodNames[nameIdx],
			})
		}
	} else {
		// Fallback: try to match by position in the data stream
		// This handles cases where the counts don't perfectly match
		for i := 0; i < len(uniqueIDs) && i < len(foodNames); i++ {
			nameIdx := len(foodNames) - 1 - i
			foods = append(foods, CustomFood{
				ID:   uniqueIDs[i],
				Name: foodNames[nameIdx],
			})
		}
	}

	return foods, nil
}

// extractGWTStringTable extracts the string array from a GWT response.
func extractGWTStringTable(body string) ([]string, error) {
	// Find the string table: ["...", "...", ...]
	start := findStringTableStart(body)
	if start == -1 {
		return nil, fmt.Errorf("no string table found in response")
	}

	end := findMatchingBracket(body, start)
	if end == -1 {
		return nil, fmt.Errorf("malformed string table")
	}

	tableStr := body[start : end+1]

	// Parse as CSV-like (JSON array of strings)
	// Remove brackets
	inner := tableStr[1 : len(tableStr)-1]
	if inner == "" {
		return nil, nil
	}

	r := csv.NewReader(stringReader(inner))
	r.LazyQuotes = true
	record, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("parsing string table: %w", err)
	}

	return record, nil
}

// extractGWTNumericData extracts the numeric values before the string table.
func extractGWTNumericData(body string) ([]string, error) {
	// Strip //OK[ prefix
	if !hasPrefix(body, "//OK[") {
		return nil, fmt.Errorf("not a GWT OK response")
	}

	// Find where string table starts
	start := findStringTableStart(body)
	if start == -1 {
		return nil, fmt.Errorf("no string table found")
	}

	// Get everything between //OK[ and the string table
	numericPart := body[5 : start-1] // -1 for the comma before [
	if numericPart == "" {
		return nil, nil
	}

	return splitCSV(numericPart), nil
}

// findStringTableStart finds the start of the string table array in a GWT response.
func findStringTableStart(body string) int {
	// The string table is the last [...] before ,0,7]
	// Find it by looking for ,[" pattern
	idx := stringIndex(body, `,["`)
	if idx == -1 {
		return -1
	}
	return idx + 1 // skip the comma
}

// findMatchingBracket finds the matching ] for a [ at position start.
func findMatchingBracket(s string, start int) int {
	if start >= len(s) || s[start] != '[' {
		return -1
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		if escaped {
			escaped = false
			continue
		}
		ch := s[i]
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '[' {
			depth++
		} else if ch == ']' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// isGWTTypeDescriptor returns true if the string looks like a GWT type descriptor.
// GWT type descriptors follow patterns like "com.package.Class/1234567890" or
// "[Lcom.package.Class;/1234567890" — they contain "." before "/" and digits after "/".
func isGWTTypeDescriptor(s string) bool {
	if stringContains(s, "[L") {
		return true
	}
	idx := stringIndex(s, "/")
	if idx == -1 {
		return false
	}
	// Type descriptors have "." before the "/" (package name) and digits after
	hasDotBefore := stringIndex(s[:idx], ".") >= 0
	hasDigitAfter := idx+1 < len(s) && s[idx+1] >= '0' && s[idx+1] <= '9'
	return hasDotBefore && hasDigitAfter
}

// ExportFood exports a single food's nutrient data as CSV.
// foodID is the Cronometer food ID, measureID is the measure ID for the serving size.
// grams is the amount in grams (use 0 for the default serving size).
func (c *Client) ExportFood(ctx context.Context, foodID int64, measureID int64, grams float64) (*FoodExport, error) {
	token, err := c.GenerateAuthToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	req, err := c.NewExportRequest(ctx, "GET", APIExportURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build export food request: %w", err)
	}

	q := req.URL.Query()
	q.Add("nonce", token)
	q.Add("generate", "food")
	q.Add("id", strconv.FormatInt(foodID, 10))
	if grams > 0 {
		q.Add("grams", strconv.FormatFloat(grams, 'f', -1, 64))
	} else {
		q.Add("grams", "1")
	}
	q.Add("mid", strconv.FormatInt(measureID, 10))
	req.URL.RawQuery = q.Encode()

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute export food request: %w", err)
	}
	defer closeAndExhaustReader(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("export food returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read export food response: %w", err)
	}

	return ParseFoodExport(string(bodyBytes))
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

// GetFood retrieves a single food's details including ingredients and per-100g nutrients.
func (c *Client) GetFood(ctx context.Context, foodID int64) (*FoodDetail, error) {
	reqBody := c.formatGWTRequest(GWTGetFood, c.Nonce, foodID)

	req, err := c.NewGWTRequestWithContext(ctx, "POST", GWTBaseURL, strings.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to build getFood request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute getFood request: %w", err)
	}
	defer closeAndExhaustReader(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getFood returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read getFood response: %w", err)
	}

	return parseGetFoodResponse(string(bodyBytes), foodID)
}

// GetFoodIngredients is a convenience method that returns just the ingredients.
func (c *Client) GetFoodIngredients(ctx context.Context, foodID int64) ([]RecipeIngredient, error) {
	detail, err := c.GetFood(ctx, foodID)
	if err != nil {
		return nil, err
	}
	return detail.Ingredients, nil
}

// GetAllFoods retrieves multiple foods' details in a single batch call.
// This is more reliable than individual GetFood calls for resolving ingredient details,
// as the GWT session state can cause individual calls to return mismatched data.
func (c *Client) GetAllFoods(ctx context.Context, foodIDs []int64) (map[int64]*FoodDetail, error) {
	if len(foodIDs) == 0 {
		return nil, nil
	}

	// Build the dynamic GWT request body
	// Format: prefix + count + |8|id1|8|id2|...|
	reqBody := c.formatGWTRequest(GWTGetAllFoodPrefix, c.Nonce)
	reqBody += strconv.Itoa(len(foodIDs))
	for _, id := range foodIDs {
		reqBody += "|8|" + strconv.FormatInt(id, 10)
	}
	reqBody += "|"

	req, err := c.NewGWTRequestWithContext(ctx, "POST", GWTBaseURL, strings.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to build getAllFood request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute getAllFood request: %w", err)
	}
	defer closeAndExhaustReader(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getAllFood returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read getAllFood response: %w", err)
	}

	return parseGetAllFoodsResponse(string(bodyBytes), foodIDs)
}

// parseGetAllFoodsResponse extracts food details for multiple foods from a batch GWT response.
// The response contains an ArrayList of Food objects. Uses the proper GWT deserializer.
func parseGetAllFoodsResponse(body string, requestedIDs []int64) (map[int64]*FoodDetail, error) {
	r, err := NewGWTReader(body)
	if err != nil {
		return nil, fmt.Errorf("GWT reader: %w", err)
	}

	// Outer wrapper: ArrayList of Food objects
	outerType := r.ReadObject()
	if outerType == "" {
		return nil, fmt.Errorf("null response")
	}

	var foods []*GWTFood
	if strings.Contains(outerType, "ArrayList") {
		count := r.ReadInt()
		for i := 0; i < count; i++ {
			foodType := r.ReadObject()
			// Require the Food class itself — a Contains("Food") check would
			// also accept FoodMeasures/FoodTag/FoodType and walk garbage.
			if foodType == "" || !strings.Contains(foodType, "models.Food/") {
				return nil, fmt.Errorf("food %d: expected Food type, got %q", i, foodType)
			}
			food, deserErr := DeserializeFood(r)
			if deserErr != nil {
				return nil, fmt.Errorf("food %d: %w", i, deserErr)
			}
			if food.Name == "" {
				return nil, fmt.Errorf("food %d (id %d): deserialized with empty name — stream misaligned or unknown layout", i, food.ID)
			}
			foods = append(foods, food)
		}
	}

	results := make(map[int64]*FoodDetail)
	for _, food := range foods {
		results[int64(food.ID)] = gwtFoodToDetail(food)
	}
	return results, nil
}

// parseGetFoodResponse extracts food details from a getFood GWT response.
// Uses the proper GWT deserializer with verified 20-field mapping.
//
// Fails loudly on anything that is not a well-formed Food: wrong response
// type, empty name, or an ID that doesn't match the request. Before
// 2026-07-17 this function accepted any parseable stream and backfilled the
// requested ID, which converted the 2026-07-16 Cronometer layout change into
// silently-inserted empty foods downstream.
func parseGetFoodResponse(body string, requestedFoodID int64) (*FoodDetail, error) {
	r, err := NewGWTReader(body)
	if err != nil {
		return nil, fmt.Errorf("GWT reader: %w", err)
	}

	// Read the Food type signature
	typeSig := r.ReadObject()
	if typeSig == "" {
		return nil, fmt.Errorf("null Food object in response")
	}
	if !strings.Contains(typeSig, "models.Food/") {
		return nil, fmt.Errorf("expected Food type, got %q", typeSig)
	}

	food, err := DeserializeFood(r)
	if err != nil {
		return nil, fmt.Errorf("deserialize food: %w", err)
	}
	if food.Name == "" {
		return nil, fmt.Errorf("food %d: deserialized with empty name — stream misaligned or unknown layout", food.ID)
	}
	// The GWT session is known to occasionally return data for a different
	// food than requested (the reason GetAllFoods exists) — reject it.
	if requestedFoodID > 0 && int64(food.ID) != requestedFoodID {
		return nil, fmt.Errorf("food ID mismatch: requested %d, got %d (GWT session state)", requestedFoodID, food.ID)
	}

	return gwtFoodToDetail(food), nil
}

// gwtFoodToDetail converts a deserialized GWTFood to the FoodDetail type used by callers.
func gwtFoodToDetail(food *GWTFood) *FoodDetail {
	detail := &FoodDetail{
		FoodID:           int64(food.ID),
		Name:             food.Name,
		Source:           food.Source,
		NutrientsPer100g: food.Nutrients,
	}
	for _, ing := range food.Ingredients {
		detail.Ingredients = append(detail.Ingredients, RecipeIngredient{
			FoodID:    int64(ing.FoodID),
			MeasureID: int64(ing.MeasureID),
			Amount:    ing.Amount,
		})
	}
	for _, m := range food.Measures {
		detail.Measures = append(detail.Measures, FoodMeasure{
			ID:    int64(m.ID),
			Name:  m.Name,
			Grams: m.Grams,
		})
	}
	return detail
}

// parseGetFoodIngredients is kept for backward compatibility with tests.
func parseGetFoodIngredients(body string) ([]RecipeIngredient, error) {
	detail, err := parseGetFoodResponse(body, 0)
	if err != nil {
		return nil, err
	}
	return detail.Ingredients, nil
}

// Helper functions to avoid import conflicts with the strings package

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

func stringContains(s, substr string) bool {
	return stringIndex(s, substr) >= 0
}

func stringIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func splitCSV(s string) []string {
	var result []string
	var current []byte
	inQuote := false
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			current = append(current, ch)
			escaped = false
			continue
		}
		if ch == '\\' && inQuote {
			current = append(current, ch)
			escaped = true
			continue
		}
		if ch == '"' {
			current = append(current, ch)
			inQuote = !inQuote
			continue
		}
		if ch == ',' && !inQuote {
			result = append(result, trimString(string(current)))
			current = current[:0]
			continue
		}
		current = append(current, ch)
	}
	if len(current) > 0 {
		result = append(result, trimString(string(current)))
	}
	return result
}

func trimString(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
