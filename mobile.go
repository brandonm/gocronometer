package gocronometer

// MobileClient talks to mobile.cronometer.com/api/v2 — the JSON API the
// Cronometer Android/iOS app uses — instead of the web app's GWT-RPC.
//
// Why: GWT payloads are positional, so any field Cronometer adds to a Java
// class shifts every value after it, and the web app is redeployed to every
// visitor at once. The mobile API carries named JSON fields and must keep
// serving app builds users have not updated, so it changes far less abruptly;
// the server even announces sunsets in a response header (see
// DeprecationNotice).
//
// The JSON failure mode is different from GWT's and is guarded explicitly: a
// renamed or dropped field decodes as a zero value without any error. Every
// field this client relies on is therefore decoded through a pointer, and an
// absent one is an *APIContractError rather than a zero.
//
// Request shapes were recovered from the 4.60.1 APK (Flutter, libapp.so) and
// confirmed against the live API on 2026-10-28; every value the client
// returns was checked against the GWT decode of the same diary day.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// MobileBaseURL is the production mobile API root.
	MobileBaseURL = "https://mobile.cronometer.com/api/v2/"

	// MobileAppVersion and MobileAppBuild identify the client as the Android
	// app build the request shapes were taken from (4.60.1, build 4085).
	MobileAppVersion = "4.60.1"
	MobileAppBuild   = "4085"

	// DefaultMobileRateLimit paces requests until the published mobile_v2
	// limit is known.
	DefaultMobileRateLimit = 20.0
)

// APIContractError reports a mobile API response that does not have the shape
// this client was built against — a missing or renamed field, an empty value
// where one is required, or a record for something other than what was asked
// for. Nothing from the response should be imported.
type APIContractError struct {
	Endpoint string // e.g. "get_foods"
	Field    string // e.g. "nutrients[].amount"
	Problem  string // e.g. "missing", "empty", "mismatch"
	Detail   string // the record involved, e.g. "food 450856"
}

func (e *APIContractError) Error() string {
	msg := fmt.Sprintf("Cronometer mobile API contract changed: %s.%s %s", e.Endpoint, e.Field, e.Problem)
	if e.Detail != "" {
		msg += " (" + e.Detail + ")"
	}
	return msg
}

// Fingerprint identifies the break independently of which record exposed it,
// so one upstream change maps to one alert.
func (e *APIContractError) Fingerprint() string {
	return e.Endpoint + "." + e.Field + ":" + e.Problem
}

func contractErr(endpoint, field, problem, detail string) *APIContractError {
	return &APIContractError{Endpoint: endpoint, Field: field, Problem: problem, Detail: detail}
}

// DeprecationNotice is an API sunset announcement read from a response header.
// The Cronometer app's DeprecationInterceptor parses a deprecation header as a
// UTC date; the standard Sunset header is captured as well.
type DeprecationNotice struct {
	Header   string // "Deprecation" or "Sunset"
	Value    string // the raw header value, normally a date
	Endpoint string // the endpoint whose response carried it
}

// MobileClientOptions configures NewMobileClient. A nil value uses defaults.
type MobileClientOptions struct {
	// BaseURL overrides MobileBaseURL (tests).
	BaseURL string
	// ThrottleConfigURL overrides ThrottleConfigURL (tests).
	ThrottleConfigURL string
	// DisableThrottleConfig skips fetching the published rate limits at login.
	DisableThrottleConfig bool
	// OnDeprecation is called once per distinct notice the server sends.
	OnDeprecation func(DeprecationNotice)
}

// MobileClient is a Cronometer client for the mobile JSON API.
type MobileClient struct {
	httpClient        *http.Client
	limiter           *rateLimiter
	baseURL           string
	throttleConfigURL string
	useThrottleConfig bool
	onDeprecation     func(DeprecationNotice)

	mu          sync.Mutex
	email       string
	password    string
	userID      int64
	token       string
	deprecation *DeprecationNotice
	seenNotices map[string]bool
	// ownedKinds maps each food the user owns to the get_myfoods list it
	// appears in ("foods", "recipes", or "meals"); nil until first fetched.
	ownedKinds map[int64]string
	// reclassified records owned foods that already triggered a fetch, so a
	// food get_myfoods never lists does not refetch it on every call.
	reclassified map[int64]bool
}

// NewMobileClient returns an unauthenticated client; call Login before use.
func NewMobileClient(opts *MobileClientOptions) *MobileClient {
	if opts == nil {
		opts = &MobileClientOptions{}
	}
	limiter := newRateLimiter(DefaultMobileRateLimit)
	c := &MobileClient{
		httpClient: &http.Client{
			Timeout:   60 * time.Second,
			Transport: &rateLimitedTransport{base: http.DefaultTransport, limiter: limiter},
		},
		limiter:           limiter,
		baseURL:           MobileBaseURL,
		throttleConfigURL: ThrottleConfigURL,
		useThrottleConfig: !opts.DisableThrottleConfig,
		onDeprecation:     opts.OnDeprecation,
		seenNotices:       map[string]bool{},
		reclassified:      map[int64]bool{},
	}
	if opts.BaseURL != "" {
		c.baseURL = opts.BaseURL
	}
	if opts.ThrottleConfigURL != "" {
		c.throttleConfigURL = opts.ThrottleConfigURL
	}
	return c
}

// Deprecation returns the most recent sunset notice the server sent, or nil.
func (c *MobileClient) Deprecation() *DeprecationNotice {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.deprecation == nil {
		return nil
	}
	n := *c.deprecation
	return &n
}

// appAuth is the auth block every request carries; login sends it with null
// userId and token.
func (c *MobileClient) appAuth(userID, token any) map[string]any {
	return map[string]any{
		"userId":  userID,
		"token":   token,
		"api":     3,
		"os":      "Android",
		"build":   MobileAppBuild,
		"flavour": "free",
	}
}

// Login authenticates and keeps the credentials in memory so an expired
// session can be renewed transparently once.
func (c *MobileClient) Login(ctx context.Context, email, password string) error {
	if c.useThrottleConfig {
		c.applyThrottleConfig(ctx)
	}
	c.mu.Lock()
	c.email, c.password = email, password
	c.mu.Unlock()
	return c.login(ctx)
}

func (c *MobileClient) login(ctx context.Context) error {
	c.mu.Lock()
	email, password := c.email, c.password
	c.mu.Unlock()

	payload := map[string]any{
		"email":    email,
		"password": password,
		// Must stay null: login treats a non-null timezone as a WRITE and
		// overwrites the account's timezone setting.
		"timezone":      nil,
		"userCode":      nil,
		"build":         MobileAppVersion + " b" + MobileAppBuild + "-a",
		"device":        "Android 14 (SDK 34), Google Pixel 6 Pro",
		"firebaseToken": "",
		"auth":          c.appAuth(nil, nil),
		"lastSeen":      0,
		"config":        map[string]any{"call_version": 2},
	}
	raw, err := c.post(ctx, "login", payload)
	if err != nil {
		return err
	}
	var resp struct {
		Result     string  `json:"result"`
		Error      string  `json:"error"`
		ID         *int64  `json:"id"`
		SessionKey *string `json:"sessionKey"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return contractErr("login", "body", "not a JSON object", err.Error())
	}
	if resp.Result != "SUCCESS" {
		if resp.Error == "TOTP_CODE_REQUIRED" {
			return fmt.Errorf("cronometer login: account has two-factor authentication enabled, which this client does not support")
		}
		return fmt.Errorf("cronometer login failed: %s %s", resp.Result, resp.Error)
	}
	if resp.ID == nil || *resp.ID <= 0 {
		return contractErr("login", "id", "missing", "")
	}
	if resp.SessionKey == nil || *resp.SessionKey == "" {
		return contractErr("login", "sessionKey", "missing", "")
	}
	c.mu.Lock()
	c.userID, c.token = *resp.ID, *resp.SessionKey
	c.mu.Unlock()
	return nil
}

func (c *MobileClient) applyThrottleConfig(ctx context.Context) {
	tc, err := fetchThrottleConfig(ctx, c.httpClient, c.throttleConfigURL)
	if err != nil || tc == nil {
		return // best-effort: the conservative default stays in effect
	}
	if api, ok := tc.APIs["mobile_v2"]; ok && api.Global.RPS > 0 {
		c.limiter.SetRate(api.Global.RPS * ThrottleSafetyFactor)
	}
}

// call sends an authenticated request, renewing the session once if the
// server rejects it.
func (c *MobileClient) call(ctx context.Context, endpoint string, payload map[string]any, callVersion int) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		c.mu.Lock()
		userID, token := c.userID, c.token
		c.mu.Unlock()
		if token == "" {
			return nil, fmt.Errorf("cronometer %s: not logged in", endpoint)
		}
		payload["auth"] = c.appAuth(userID, token)
		payload["lastSeen"] = 0
		payload["config"] = map[string]any{"call_version": callVersion}

		raw, err := c.post(ctx, endpoint, payload)
		expired := false
		if err != nil {
			var se *mobileStatusError
			if !errors.As(err, &se) || (se.status != http.StatusUnauthorized && se.status != http.StatusForbidden) {
				return nil, err
			}
			expired = true
		} else {
			var status struct {
				Result string `json:"result"`
				Error  string `json:"error"`
			}
			if json.Unmarshal(raw, &status) == nil && (status.Result == "FAIL" || status.Result == "FAILURE") {
				if attempt > 0 {
					return nil, fmt.Errorf("cronometer %s failed: %s", endpoint, status.Error)
				}
				expired = true
			}
		}
		if !expired {
			return raw, nil
		}
		if attempt > 0 {
			return nil, fmt.Errorf("cronometer %s: session rejected after re-login: %w", endpoint, err)
		}
		if lerr := c.login(ctx); lerr != nil {
			return nil, fmt.Errorf("cronometer %s: re-login after rejected session: %w", endpoint, lerr)
		}
	}
}

type mobileStatusError struct {
	endpoint string
	status   int
}

func (e *mobileStatusError) Error() string {
	return fmt.Sprintf("cronometer %s returned status %d", e.endpoint, e.status)
}

func (c *MobileClient) post(ctx context.Context, endpoint string, payload map[string]any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("cronometer %s: encode request: %w", endpoint, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("cronometer %s: build request: %w", endpoint, err)
	}
	// The app sends JSON as text/plain; match it rather than look different.
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	req.Header.Set("User-Agent", "Dart/3.9 (dart:io)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cronometer %s: %w", endpoint, err)
	}
	defer closeAndExhaustReader(resp.Body)
	c.recordDeprecation(endpoint, resp.Header)

	if resp.StatusCode != http.StatusOK {
		return nil, &mobileStatusError{endpoint: endpoint, status: resp.StatusCode}
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cronometer %s: read response: %w", endpoint, err)
	}
	return raw, nil
}

func (c *MobileClient) recordDeprecation(endpoint string, h http.Header) {
	for _, name := range []string{"Deprecation", "Sunset"} {
		value := strings.TrimSpace(h.Get(name))
		if value == "" {
			continue
		}
		n := DeprecationNotice{Header: name, Value: value, Endpoint: endpoint}
		c.mu.Lock()
		c.deprecation = &n
		fresh := !c.seenNotices[name+"="+value]
		c.seenNotices[name+"="+value] = true
		cb := c.onDeprecation
		c.mu.Unlock()
		if fresh && cb != nil {
			cb(n)
		}
	}
}

// mobileDay formats a date the way the mobile API expects: YYYY-M-D, not
// zero-padded.
func mobileDay(d time.Time) string {
	return fmt.Sprintf("%d-%d-%d", d.Year(), int(d.Month()), d.Day())
}

// GetDayServings returns the food servings logged on date. Biometric,
// exercise, and other diary entries are skipped.
func (c *MobileClient) GetDayServings(ctx context.Context, date time.Time) ([]DayServing, error) {
	const ep = "get_diary"
	raw, err := c.call(ctx, ep, map[string]any{"day": mobileDay(date)}, 1)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Diary *[]json.RawMessage `json:"diary"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, contractErr(ep, "body", "not a JSON object", err.Error())
	}
	if resp.Diary == nil {
		return nil, contractErr(ep, "diary", "missing", "")
	}

	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	want := date.Format("2006-01-02")
	servings := []DayServing{}
	for i, entry := range *resp.Diary {
		var e struct {
			Type      *string  `json:"type"`
			FoodID    *int64   `json:"foodId"`
			MeasureID *int64   `json:"measureId"`
			Grams     *float64 `json:"grams"`
			Order     *int64   `json:"order"`
			Day       *string  `json:"day"`
		}
		if err := json.Unmarshal(entry, &e); err != nil {
			return nil, contractErr(ep, "diary[]", "not a JSON object", fmt.Sprintf("entry %d", i))
		}
		if e.Type == nil {
			return nil, contractErr(ep, "type", "missing", fmt.Sprintf("entry %d", i))
		}
		if *e.Type != "Serving" {
			continue
		}
		detail := fmt.Sprintf("entry %d", i)
		switch {
		case e.FoodID == nil:
			return nil, contractErr(ep, "foodId", "missing", detail)
		case e.MeasureID == nil:
			return nil, contractErr(ep, "measureId", "missing", detail)
		case e.Grams == nil:
			return nil, contractErr(ep, "grams", "missing", detail)
		case e.Order == nil:
			return nil, contractErr(ep, "order", "missing", detail)
		case e.Day == nil:
			return nil, contractErr(ep, "day", "missing", detail)
		}
		if *e.FoodID <= 0 {
			return nil, contractErr(ep, "foodId", "not positive", detail)
		}
		if *e.Day != want {
			return nil, contractErr(ep, "day", "mismatch", fmt.Sprintf("%s: requested %s, got %s", detail, want, *e.Day))
		}
		servings = append(servings, DayServing{
			FoodID:    *e.FoodID,
			Amount:    *e.Grams, // despite the name, the amount in the measure's units — same value GWT carried
			MeasureID: *e.MeasureID,
			MealGroup: int(*e.Order >> 16), // the high 16 bits of the row order are the meal group
			Date:      day,
		})
	}
	return servings, nil
}

type mobileFood struct {
	ID       *int64  `json:"id"`
	Name     *string `json:"name"`
	Source   *string `json:"source"`
	Owner    *int64  `json:"owner"`
	Measures *[]struct {
		ID    *int64   `json:"id"`
		Name  *string  `json:"name"`
		Value *float64 `json:"value"`
	} `json:"measures"`
	Nutrients *[]struct {
		ID     *int     `json:"id"`
		Amount *float64 `json:"amount"`
	} `json:"nutrients"`
	Ingredients *[]struct {
		FoodID    *int64   `json:"foodId"`
		MeasureID *int64   `json:"measureId"`
		Grams     *float64 `json:"grams"`
	} `json:"ingredients"`
}

// toDetail validates a food and converts it. Whether an absent ingredient
// list is legitimate depends on what kind of food it is, which the food
// object does not say; GetFood decides that.
func (f *mobileFood) toDetail(ep string) (*FoodDetail, error) {
	if f.ID == nil {
		return nil, contractErr(ep, "id", "missing", "")
	}
	detail := fmt.Sprintf("food %d", *f.ID)
	switch {
	case f.Name == nil:
		return nil, contractErr(ep, "name", "missing", detail)
	case *f.Name == "":
		return nil, contractErr(ep, "name", "empty", detail)
	case f.Source == nil:
		return nil, contractErr(ep, "source", "missing", detail)
	case f.Owner == nil:
		return nil, contractErr(ep, "owner", "missing", detail)
	case f.Nutrients == nil:
		return nil, contractErr(ep, "nutrients", "missing", detail)
	case len(*f.Nutrients) == 0:
		return nil, contractErr(ep, "nutrients", "empty", detail)
	case f.Measures == nil:
		return nil, contractErr(ep, "measures", "missing", detail)
	case len(*f.Measures) == 0:
		return nil, contractErr(ep, "measures", "empty", detail)
	}

	out := &FoodDetail{
		FoodID:           *f.ID,
		Name:             *f.Name,
		Source:           *f.Source,
		NutrientsPer100g: make(map[int]float64, len(*f.Nutrients)),
	}
	for _, n := range *f.Nutrients {
		if n.ID == nil {
			return nil, contractErr(ep, "nutrients[].id", "missing", detail)
		}
		if n.Amount == nil {
			return nil, contractErr(ep, "nutrients[].amount", "missing", detail)
		}
		out.NutrientsPer100g[*n.ID] = *n.Amount
	}
	for _, m := range *f.Measures {
		switch {
		case m.ID == nil:
			return nil, contractErr(ep, "measures[].id", "missing", detail)
		case m.Name == nil:
			return nil, contractErr(ep, "measures[].name", "missing", detail)
		case m.Value == nil:
			return nil, contractErr(ep, "measures[].value", "missing", detail)
		}
		// value is the measure's weight in grams, the same figure GWT's
		// Measure carried (e.g. "oz" = 28.3495231).
		out.Measures = append(out.Measures, FoodMeasure{ID: *m.ID, Name: *m.Name, Grams: *m.Value})
	}
	if f.Ingredients != nil {
		for _, ing := range *f.Ingredients {
			switch {
			case ing.FoodID == nil:
				return nil, contractErr(ep, "ingredients[].foodId", "missing", detail)
			case ing.MeasureID == nil:
				return nil, contractErr(ep, "ingredients[].measureId", "missing", detail)
			case ing.Grams == nil:
				return nil, contractErr(ep, "ingredients[].grams", "missing", detail)
			}
			out.Ingredients = append(out.Ingredients, RecipeIngredient{FoodID: *ing.FoodID, MeasureID: *ing.MeasureID, Amount: *ing.Grams})
		}
	}
	return out, nil
}

// GetFood fetches one food with its recipe ingredients.
func (c *MobileClient) GetFood(ctx context.Context, foodID int64) (*FoodDetail, error) {
	const ep = "get_food"
	raw, err := c.call(ctx, ep, map[string]any{"id": foodID}, 1)
	if err != nil {
		return nil, err
	}
	var f mobileFood
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, contractErr(ep, "body", "not a JSON object", err.Error())
	}
	d, err := f.toDetail(ep)
	if err != nil {
		return nil, err
	}
	if d.FoodID != foodID {
		return nil, contractErr(ep, "id", "mismatch", fmt.Sprintf("requested %d, got %d", foodID, d.FoodID))
	}
	// get_food omits the ingredients key for a plain custom food, so its
	// absence is only a contract break for a recipe or meal. Importing a
	// recipe without its ingredients would lose the per-ingredient breakdown.
	if f.Ingredients == nil && *f.Owner != 0 {
		kind, err := c.ownedKind(ctx, foodID)
		if err != nil {
			return nil, err
		}
		if kind == "recipes" || kind == "meals" {
			return nil, contractErr(ep, "ingredients", "missing", fmt.Sprintf("food %d (%s)", foodID, kind))
		}
	}
	return d, nil
}

// ownedKind reports which get_myfoods list an owned food is in, fetching the
// classification at most once per food that the cached copy lacks (a recipe
// created since the last fetch). "" means get_myfoods does not list it, as
// for foods the user has since deleted.
func (c *MobileClient) ownedKind(ctx context.Context, foodID int64) (string, error) {
	c.mu.Lock()
	kind, known := c.ownedKinds[foodID]
	tried := c.reclassified[foodID]
	c.mu.Unlock()
	if known || tried {
		return kind, nil
	}
	if _, err := c.FindMyFoods(ctx); err != nil {
		return "", fmt.Errorf("classify owned food %d: %w", foodID, err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reclassified[foodID] = true
	return c.ownedKinds[foodID], nil
}

// GetAllFoods fetches many foods in one call. get_foods omits recipe
// ingredients, so foods the user owns (custom foods, recipes, meals) are
// re-fetched individually with GetFood; database foods never have ingredients.
func (c *MobileClient) GetAllFoods(ctx context.Context, foodIDs []int64) (map[int64]*FoodDetail, error) {
	const ep = "get_foods"
	out := make(map[int64]*FoodDetail, len(foodIDs))
	if len(foodIDs) == 0 {
		return out, nil
	}
	raw, err := c.call(ctx, ep, map[string]any{"ids": foodIDs}, 1)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Foods *[]mobileFood `json:"foods"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, contractErr(ep, "body", "not a JSON object", err.Error())
	}
	if resp.Foods == nil {
		return nil, contractErr(ep, "foods", "missing", "")
	}

	wanted := make(map[int64]bool, len(foodIDs))
	for _, id := range foodIDs {
		wanted[id] = true
	}
	var owned []int64
	for i := range *resp.Foods {
		f := &(*resp.Foods)[i]
		if f.ID != nil && !wanted[*f.ID] {
			continue
		}
		d, err := f.toDetail(ep)
		if err != nil {
			return nil, err
		}
		out[d.FoodID] = d
		if *f.Owner != 0 {
			owned = append(owned, d.FoodID)
		}
	}
	for _, id := range foodIDs {
		if out[id] == nil {
			return nil, fmt.Errorf("cronometer %s: food %d not returned", ep, id)
		}
	}
	for _, id := range owned {
		d, err := c.GetFood(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("cronometer: resolve owned food %d: %w", id, err)
		}
		out[id] = d
	}
	return out, nil
}

// FindMyFoods lists every food the user owns: custom foods, recipes, and
// meals.
func (c *MobileClient) FindMyFoods(ctx context.Context) ([]CustomFood, error) {
	const ep = "get_myfoods"
	raw, err := c.call(ctx, ep, map[string]any{}, 1)
	if err != nil {
		return nil, err
	}
	type entry struct {
		ID   *int64  `json:"id"`
		Name *string `json:"name"`
	}
	var resp struct {
		Foods   *[]entry `json:"foods"`
		Recipes *[]entry `json:"recipes"`
		Meals   *[]entry `json:"meals"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, contractErr(ep, "body", "unexpected shape", err.Error())
	}
	var out []CustomFood
	seen := map[int64]bool{}
	kinds := map[int64]string{}
	for _, l := range []struct {
		name    string
		entries *[]entry
	}{{"foods", resp.Foods}, {"recipes", resp.Recipes}, {"meals", resp.Meals}} {
		list, entries := l.name, l.entries
		if entries == nil {
			return nil, contractErr(ep, list, "missing", "")
		}
		for i, e := range *entries {
			if e.ID == nil || *e.ID <= 0 {
				return nil, contractErr(ep, list+"[].id", "missing", fmt.Sprintf("entry %d", i))
			}
			if e.Name == nil {
				return nil, contractErr(ep, list+"[].name", "missing", fmt.Sprintf("food %d", *e.ID))
			}
			if seen[*e.ID] {
				continue
			}
			seen[*e.ID] = true
			kinds[*e.ID] = list
			out = append(out, CustomFood{ID: *e.ID, Name: *e.Name})
		}
	}
	c.mu.Lock()
	c.ownedKinds = kinds
	c.mu.Unlock()
	return out, nil
}

// closeAndExhaustReader drains and closes a response body so the connection
// can be reused. Errors are ignored: it only runs deferred, where there is no
// action to take on failure.
func closeAndExhaustReader(r io.ReadCloser) {
	_, _ = io.Copy(io.Discard, r)
	_ = r.Close()
}
