package gocronometer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DayServing is a single food entry from a day's diary, as returned by the GWT
// getDayInfo API (the same call the web app uses to render the diary). It is the
// RPC replacement for a row of the legacy CSV /export servings dump.
//
// Nutrient values are NOT embedded here — a serving references a food by ID plus
// an amount; resolve nutrients via GetAllFoods (per-100g) × grams, reusing the
// existing food cache. (This mirrors how custom-food ingredients are resolved.)
type DayServing struct {
	FoodID    int64     // Cronometer food ID (resolve nutrients via GetAllFoods)
	Amount    float64   // quantity in the measure's units (grams when MeasureID is the gram measure)
	MeasureID int64     // which measure the amount is expressed in
	Date      time.Time // the diary day this serving belongs to (date only)
}

// GetDayInfoRaw returns the raw GWT-RPC getDayInfo response for a date. Exposed
// for reverse-engineering / debugging the response format.
//
// Requires an authenticated client (c.Nonce + c.UserID set by Login).
func (c *Client) GetDayInfoRaw(ctx context.Context, date time.Time) (string, error) {
	if c.Nonce == "" || c.UserID == "" {
		return "", fmt.Errorf("not authenticated: call Login first")
	}

	// Day serializes as day|month|year (month 1-based), userID is the trailing int param.
	reqBody := fmt.Sprintf(GWTGetDayInfo, c.Nonce, date.Day(), int(date.Month()), date.Year(), c.UserID)

	req, err := c.NewGWTRequestWithContext(ctx, "POST", GWTBaseURL, strings.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to build getDayInfo request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute getDayInfo request: %w", err)
	}
	defer closeAndExhaustReader(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("getDayInfo returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read getDayInfo response: %w", err)
	}
	return string(bodyBytes), nil
}

// GetDayServings retrieves one day's food servings via GWT-RPC (getDayInfo),
// avoiding the rate-limited CSV /export endpoint entirely. getDayInfo is a
// single-day call, so callers loop it across a date range.
//
// Requires an authenticated client (c.Nonce + c.UserID set by Login).
func (c *Client) GetDayServings(ctx context.Context, date time.Time) ([]DayServing, error) {
	raw, err := c.GetDayInfoRaw(ctx, date)
	if err != nil {
		return nil, err
	}
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	return parseDayInfoServings(raw, day, c.UserID)
}

// parseDayInfoServings deserializes a getDayInfo response into the day's servings.
//
// PROVISIONAL: the DayInfo wrapper and Serving field layout are reverse-engineered
// from a single captured response and verified by diary_test.go against that
// fixture. The Serving block is ~20 fields; the fields we extract (foodID, amount,
// date) are decoded from the observed wire layout. Other DayInfo contents
// (biometrics, exercise, sleep) are intentionally ignored — Apple Health covers them.
func parseDayInfoServings(body string, day time.Time, userID string) ([]DayServing, error) {
	r, err := NewGWTReader(body)
	if err != nil {
		return nil, fmt.Errorf("GWT reader: %w", err)
	}
	if dayInfoType := r.ReadObject(); dayInfoType != "" && !strings.Contains(dayInfoType, "DayInfo") {
		return nil, fmt.Errorf("expected DayInfo, got %q", dayInfoType)
	}

	// Each Serving carries the consecutive read-order signature
	//   userID, amount(double), foodID(int), "base64-longhash"
	// (verified live: foodID resolves via GetFood — e.g. 69796849 -> "Mocha (20oz)").
	// Scanning for this signature sidesteps deserializing the biometric/exercise
	// objects that precede the servings list, and is robust to the variable serving
	// size (some servings embed an object back-ref that shifts field positions).
	toks := r.Tokens()
	var servings []DayServing
	seen := make(map[string]bool)
	for i := len(toks) - 1; i >= 3; i-- {
		if toks[i] != userID {
			continue
		}
		amtTok, foodTok, hashTok := toks[i-1], toks[i-2], toks[i-3]
		// The longhash is a short quoted base64 (e.g. "EctxK_") — not a JSON blob,
		// which is how biometric SampleStats entries are encoded.
		if !strings.HasPrefix(hashTok, `"`) || strings.Contains(hashTok, "{") || len(hashTok) > 16 {
			continue
		}
		amount, err1 := strconv.ParseFloat(amtTok, 64)
		foodID, err2 := strconv.ParseInt(foodTok, 10, 64)
		if err1 != nil || err2 != nil || foodID <= 0 {
			continue
		}
		if seen[hashTok] {
			continue
		}
		seen[hashTok] = true
		// The token just past the longhash (read-order) is the serving's measure ID,
		// used to convert amount → grams via the food's measures.
		var measureID int64
		if i-4 >= 0 {
			measureID, _ = strconv.ParseInt(toks[i-4], 10, 64)
		}
		servings = append(servings, DayServing{FoodID: foodID, Amount: amount, MeasureID: measureID, Date: day})
	}
	return servings, nil
}
