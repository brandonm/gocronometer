package gocronometer

// Tests for MobileClient against live mobile.cronometer.com/api/v2 responses
// captured on 2026-10-28 for the diary day 2026-10-27 (testdata/mobile-*.json,
// user ID replaced with 10000001, biometric/exercise entries replaced by
// synthetic placeholders). The same day was captured over the web app's
// GWT-RPC API before this library dropped it; testdata/golden-gwt-*.json
// freeze what that decoder returned, and are the independent cross-check for
// every value asserted here.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// mobileTestServer serves the captured fixtures by endpoint name and records
// every request body it receives.
type mobileTestServer struct {
	t        *testing.T
	mu       sync.Mutex
	requests map[string][]map[string]any
	// override, when set for an endpoint, replaces the fixture response.
	override map[string]func(body map[string]any) (int, string)
	headers  http.Header
	srv      *httptest.Server
}

func newMobileTestServer(t *testing.T) *mobileTestServer {
	t.Helper()
	m := &mobileTestServer{
		t:        t,
		requests: map[string][]map[string]any{},
		override: map[string]func(map[string]any) (int, string){},
		headers:  http.Header{},
	}
	m.srv = httptest.NewServer(http.HandlerFunc(m.handle))
	t.Cleanup(m.srv.Close)
	return m
}

func (m *mobileTestServer) handle(w http.ResponseWriter, r *http.Request) {
	endpoint := strings.TrimPrefix(r.URL.Path, "/api/v2/")
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		m.t.Errorf("%s: request body is not JSON: %v", endpoint, err)
	}
	m.mu.Lock()
	m.requests[endpoint] = append(m.requests[endpoint], body)
	override := m.override[endpoint]
	m.mu.Unlock()
	for k, vs := range m.headers {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	if override != nil {
		status, resp := override(body)
		w.WriteHeader(status)
		io.WriteString(w, resp)
		return
	}
	var fixture string
	switch endpoint {
	case "login":
		io.WriteString(w, `{"result":"SUCCESS","id":10000001,"sessionKey":"session-1"}`)
		return
	case "get_diary":
		fixture = "testdata/mobile-get_diary-2026-10-27.json"
	case "get_foods":
		fixture = "testdata/mobile-get_foods-2026-10-27.json"
	case "get_food":
		fixture = fmt.Sprintf("testdata/mobile-get_food-%.0f.json", body["id"])
	case "get_myfoods":
		fixture = "testdata/mobile-get_myfoods.json"
	default:
		http.NotFound(w, r)
		return
	}
	data, err := os.ReadFile(fixture)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Write(data)
}

func (m *mobileTestServer) last(endpoint string) map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	reqs := m.requests[endpoint]
	if len(reqs) == 0 {
		m.t.Fatalf("no %s request was made", endpoint)
	}
	return reqs[len(reqs)-1]
}

func (m *mobileTestServer) count(endpoint string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.requests[endpoint])
}

// client returns a logged-in MobileClient pointed at the test server.
func (m *mobileTestServer) client(t *testing.T) *MobileClient {
	t.Helper()
	c := NewMobileClient(&MobileClientOptions{BaseURL: m.srv.URL + "/api/v2/", DisableThrottleConfig: true})
	if err := c.Login(context.Background(), "user@example.com", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	return c
}

// fixtureWithout loads a fixture and applies mutate to its decoded JSON.
func fixtureWithout(t *testing.T, path string, mutate func(map[string]any)) string {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(loadFixture(t, path)), &doc); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	mutate(doc)
	out, _ := json.Marshal(doc)
	return string(out)
}

// loadGolden decodes a frozen GWT-era decode from testdata.
func loadGolden(t *testing.T, path string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(loadFixture(t, path)), v); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

func asContractError(t *testing.T, err error) *APIContractError {
	t.Helper()
	var ce *APIContractError
	if !errors.As(err, &ce) {
		t.Fatalf("error = %v, want *APIContractError", err)
	}
	return ce
}

var oct27 = time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)

// TestMobileLogin_TimezoneIsNull pins the one login field that is a write:
// a non-null timezone overwrites the account's timezone setting server-side.
func TestMobileLogin_TimezoneIsNull(t *testing.T) {
	srv := newMobileTestServer(t)
	srv.client(t)

	body := srv.last("login")
	tz, present := body["timezone"]
	if !present || tz != nil {
		t.Fatalf("login timezone = %v (present=%v), want an explicit null", tz, present)
	}
	if body["email"] != "user@example.com" || body["password"] != "hunter2" {
		t.Errorf("credentials not sent: email=%v", body["email"])
	}
	auth, _ := body["auth"].(map[string]any)
	if auth == nil || auth["userId"] != nil || auth["token"] != nil {
		t.Errorf("login auth block should carry null userId/token, got %v", auth)
	}
}

func TestMobileLogin_TwoFactorRequired(t *testing.T) {
	srv := newMobileTestServer(t)
	srv.override["login"] = func(map[string]any) (int, string) {
		return 200, `{"result":"FAIL","error":"TOTP_CODE_REQUIRED"}`
	}
	c := NewMobileClient(&MobileClientOptions{BaseURL: srv.srv.URL + "/api/v2/", DisableThrottleConfig: true})
	err := c.Login(context.Background(), "user@example.com", "hunter2")
	if err == nil || !strings.Contains(err.Error(), "two-factor") {
		t.Fatalf("expected a two-factor error, got %v", err)
	}
}

func TestMobileLogin_MissingSessionKeyIsContractError(t *testing.T) {
	srv := newMobileTestServer(t)
	srv.override["login"] = func(map[string]any) (int, string) {
		return 200, `{"result":"SUCCESS","id":10000001}`
	}
	c := NewMobileClient(&MobileClientOptions{BaseURL: srv.srv.URL + "/api/v2/", DisableThrottleConfig: true})
	ce := asContractError(t, c.Login(context.Background(), "user@example.com", "hunter2"))
	if ce.Endpoint != "login" || ce.Field != "sessionKey" {
		t.Errorf("contract error = %+v, want login/sessionKey", ce)
	}
}

func TestMobileRequests_CarryAuthBlock(t *testing.T) {
	srv := newMobileTestServer(t)
	c := srv.client(t)
	if _, err := c.GetDayServings(context.Background(), oct27); err != nil {
		t.Fatalf("GetDayServings: %v", err)
	}
	body := srv.last("get_diary")
	auth, _ := body["auth"].(map[string]any)
	if auth["userId"] != float64(10000001) || auth["token"] != "session-1" {
		t.Errorf("auth block = %v, want userId 10000001 and the login sessionKey", auth)
	}
	if body["day"] != "2026-10-27" {
		t.Errorf("day = %v", body["day"])
	}
}

func TestMobileDayFormat_NotZeroPadded(t *testing.T) {
	if got := mobileDay(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)); got != "2026-1-5" {
		t.Errorf("mobileDay = %q, want 2026-1-5", got)
	}
}

// TestMobileGetDayServings_MatchesGWT asserts every serving of the captured day
// against the values GWT getDayInfo decoded for the same day.
func TestMobileGetDayServings_MatchesGWT(t *testing.T) {
	srv := newMobileTestServer(t)
	servings, err := srv.client(t).GetDayServings(context.Background(), oct27)
	if err != nil {
		t.Fatalf("GetDayServings: %v", err)
	}
	type row struct {
		food, measure int64
		amount        float64
		group         int
	}
	var got []row
	for _, s := range servings {
		if !s.Date.Equal(oct27) {
			t.Errorf("serving %d dated %v", s.FoodID, s.Date)
		}
		got = append(got, row{s.FoodID, s.MeasureID, s.Amount, s.MealGroup})
	}
	sort.Slice(got, func(i, j int) bool {
		if got[i].food != got[j].food {
			return got[i].food < got[j].food
		}
		return got[i].amount < got[j].amount
	})
	want := []row{
		{450856, 998940, 118, 1},
		{14570120, 90510270, 99.96686049295775, 4},
		{20118617, 55621460, 6, 5},
		{20398297, 56355200, 100, 5},
		{22688784, 62333753, 182, 2},
		{27536487, 75114657, 499.8006, 6},
		{27536487, 75114657, 499.8006, 6},
		{27536487, 75114657, 499.8006, 6},
		{28932880, 214568703, 100, 5},
		{68943617, 241899897, 1, 1},
		{80069621, 290575524, 1, 2},
		{81323197, 296029014, 1, 1},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d servings, want %d (biometric/exercise entries must be skipped)", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("serving %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestMobileGetDayServings_MissingFieldIsContractError(t *testing.T) {
	for _, field := range []string{"foodId", "measureId", "grams", "order", "day"} {
		t.Run(field, func(t *testing.T) {
			srv := newMobileTestServer(t)
			body := fixtureWithout(t, "testdata/mobile-get_diary-2026-10-27.json", func(doc map[string]any) {
				delete(doc["diary"].([]any)[0].(map[string]any), field)
			})
			srv.override["get_diary"] = func(map[string]any) (int, string) { return 200, body }
			_, err := srv.client(t).GetDayServings(context.Background(), oct27)
			ce := asContractError(t, err)
			if ce.Endpoint != "get_diary" || ce.Field != field {
				t.Errorf("contract error = %+v, want get_diary/%s", ce, field)
			}
		})
	}
}

func TestMobileGetDayServings_MissingDiaryIsContractError(t *testing.T) {
	srv := newMobileTestServer(t)
	srv.override["get_diary"] = func(map[string]any) (int, string) { return 200, `{"entries":[]}` }
	ce := asContractError(t, func() error { _, err := srv.client(t).GetDayServings(context.Background(), oct27); return err }())
	if ce.Field != "diary" {
		t.Errorf("contract error = %+v, want field diary", ce)
	}
}

func TestMobileGetDayServings_WrongDayIsContractError(t *testing.T) {
	srv := newMobileTestServer(t)
	_, err := srv.client(t).GetDayServings(context.Background(), oct27.AddDate(0, 0, 1))
	ce := asContractError(t, err)
	if ce.Field != "day" {
		t.Errorf("contract error = %+v, want field day", ce)
	}
}

// TestMobileGetAllFoods_MatchesGWT resolves the day's ten foods over the mobile
// API and checks each against the GWT decode of the same foods: identical
// nutrient maps, measures, and (for owned recipes) ingredients.
func TestMobileGetAllFoods_MatchesGWT(t *testing.T) {
	ids := []int64{81323197, 68943617, 450856, 80069621, 22688784, 14570120, 20398297, 20118617, 28932880, 27536487}
	var gwt map[int64]*FoodDetail
	loadGolden(t, "testdata/golden-gwt-getallfood-2026-10-27.json", &gwt)

	srv := newMobileTestServer(t)
	foods, err := srv.client(t).GetAllFoods(context.Background(), ids)
	if err != nil {
		t.Fatalf("GetAllFoods: %v", err)
	}
	if len(foods) != len(ids) {
		t.Fatalf("got %d foods, want %d", len(foods), len(ids))
	}
	for _, id := range ids {
		m, g := foods[id], gwt[id]
		if m == nil {
			t.Errorf("food %d missing", id)
			continue
		}
		if m.FoodID != id || m.Name == "" || m.Source != g.Source {
			t.Errorf("food %d: id=%d name=%q source=%q (GWT source %q)", id, m.FoodID, m.Name, m.Source, g.Source)
		}
		if len(m.NutrientsPer100g) != len(g.NutrientsPer100g) {
			t.Errorf("food %d: %d nutrients, GWT has %d", id, len(m.NutrientsPer100g), len(g.NutrientsPer100g))
		}
		for code, v := range g.NutrientsPer100g {
			if mv, ok := m.NutrientsPer100g[code]; !ok || mv != v {
				t.Errorf("food %d nutrient %d: mobile %v (present=%v), GWT %v", id, code, mv, ok, v)
			}
		}
		if fmt.Sprint(m.Measures) != fmt.Sprint(g.Measures) {
			t.Errorf("food %d measures:\n mobile %v\n GWT    %v", id, m.Measures, g.Measures)
		}
		if fmt.Sprint(m.Ingredients) != fmt.Sprint(g.Ingredients) {
			t.Errorf("food %d ingredients:\n mobile %v\n GWT    %v", id, m.Ingredients, g.Ingredients)
		}
	}

	// Owned foods are re-fetched individually: get_foods omits ingredients,
	// and without them a recipe would import as an ingredient-less leaf.
	if n := srv.count("get_food"); n != 3 {
		t.Errorf("get_food called %d times, want 3 (one per owned food)", n)
	}
	if n := len(foods[81323197].Ingredients); n != 3 {
		t.Errorf("mocha recipe has %d ingredients, want 3", n)
	}
}

func TestMobileGetFood_MatchesGWT(t *testing.T) {
	var gwt *FoodDetail
	loadGolden(t, "testdata/golden-gwt-getfood-81323197.json", &gwt)
	srv := newMobileTestServer(t)
	m, err := srv.client(t).GetFood(context.Background(), 81323197)
	if err != nil {
		t.Fatalf("GetFood: %v", err)
	}
	if m.Name != gwt.Name || m.Source != gwt.Source {
		t.Errorf("name/source: mobile %q/%q, GWT %q/%q", m.Name, m.Source, gwt.Name, gwt.Source)
	}
	if fmt.Sprint(m.Ingredients) != fmt.Sprint(gwt.Ingredients) {
		t.Errorf("ingredients:\n mobile %v\n GWT    %v", m.Ingredients, gwt.Ingredients)
	}
	if fmt.Sprint(m.NutrientsPer100g) != fmt.Sprint(gwt.NutrientsPer100g) {
		t.Error("nutrients differ from GWT")
	}
}

// TestMobileGetFood_PlainCustomFoodHasNoIngredients covers a custom food that
// is not a recipe: get_food omits the ingredients key entirely for it (live
// 2026-10-26, food 83483646, listed under get_myfoods "foods").
func TestMobileGetFood_PlainCustomFoodHasNoIngredients(t *testing.T) {
	srv := newMobileTestServer(t)
	d, err := srv.client(t).GetFood(context.Background(), 83483646)
	if err != nil {
		t.Fatalf("GetFood: %v", err)
	}
	if len(d.Ingredients) != 0 || d.Name == "" || len(d.NutrientsPer100g) == 0 {
		t.Errorf("plain custom food: name=%q ingredients=%d nutrients=%d", d.Name, len(d.Ingredients), len(d.NutrientsPer100g))
	}
}

// TestMobileGetFood_RecipeWithoutIngredientsIsContractError covers the
// silent-loss case: get_myfoods lists the food as a recipe, so an absent
// ingredients key means the field moved, not that the recipe is empty.
func TestMobileGetFood_RecipeWithoutIngredientsIsContractError(t *testing.T) {
	for _, id := range []int64{81323197, 80069621} { // a recipe and a meal
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			srv := newMobileTestServer(t)
			body := fixtureWithout(t, fmt.Sprintf("testdata/mobile-get_food-%d.json", id), func(doc map[string]any) { delete(doc, "ingredients") })
			srv.override["get_food"] = func(map[string]any) (int, string) { return 200, body }
			_, err := srv.client(t).GetFood(context.Background(), id)
			if ce := asContractError(t, err); ce.Field != "ingredients" {
				t.Errorf("contract error = %+v, want field ingredients", ce)
			}
		})
	}
}

// TestMobileGetFood_ClassificationFetchedOnce checks that the recipe/meal
// classification comes from one get_myfoods call, not one per food.
func TestMobileGetFood_ClassificationFetchedOnce(t *testing.T) {
	srv := newMobileTestServer(t)
	c := srv.client(t)
	for _, id := range []int64{83483646, 81323197, 68943617, 80069621} {
		if _, err := c.GetFood(context.Background(), id); err != nil {
			t.Fatalf("GetFood %d: %v", id, err)
		}
	}
	if n := srv.count("get_myfoods"); n != 1 {
		t.Errorf("get_myfoods called %d times, want 1", n)
	}
}

func TestMobileGetFood_IDMismatchIsContractError(t *testing.T) {
	srv := newMobileTestServer(t)
	srv.override["get_food"] = func(map[string]any) (int, string) {
		return 200, loadFixture(t, "testdata/mobile-get_food-68943617.json")
	}
	_, err := srv.client(t).GetFood(context.Background(), 81323197)
	if ce := asContractError(t, err); ce.Field != "id" {
		t.Errorf("contract error = %+v, want field id", ce)
	}
}

func TestMobileGetAllFoods_MissingFieldIsContractError(t *testing.T) {
	cases := map[string]func(food map[string]any){
		"name":      func(f map[string]any) { delete(f, "name") },
		"nutrients": func(f map[string]any) { delete(f, "nutrients") },
		"measures":  func(f map[string]any) { delete(f, "measures") },
		"source":    func(f map[string]any) { delete(f, "source") },
		"owner":     func(f map[string]any) { delete(f, "owner") },
		"nutrients[].amount": func(f map[string]any) {
			delete(f["nutrients"].([]any)[0].(map[string]any), "amount")
		},
		"measures[].value": func(f map[string]any) {
			delete(f["measures"].([]any)[0].(map[string]any), "value")
		},
	}
	for field, mutate := range cases {
		t.Run(field, func(t *testing.T) {
			srv := newMobileTestServer(t)
			body := fixtureWithout(t, "testdata/mobile-get_foods-2026-10-27.json", func(doc map[string]any) {
				for _, f := range doc["foods"].([]any) {
					if food := f.(map[string]any); food["id"] == float64(450856) {
						mutate(food)
					}
				}
			})
			srv.override["get_foods"] = func(map[string]any) (int, string) { return 200, body }
			_, err := srv.client(t).GetAllFoods(context.Background(), []int64{450856})
			ce := asContractError(t, err)
			if ce.Endpoint != "get_foods" || ce.Field != field {
				t.Errorf("contract error = %+v, want get_foods/%s", ce, field)
			}
		})
	}
}

func TestMobileGetAllFoods_RequestsOnlyGivenIDs(t *testing.T) {
	srv := newMobileTestServer(t)
	foods, err := srv.client(t).GetAllFoods(context.Background(), []int64{450856})
	if err != nil {
		t.Fatalf("GetAllFoods: %v", err)
	}
	// The fixture returns all ten foods; only the requested one may be kept.
	if len(foods) != 1 || foods[450856] == nil {
		t.Errorf("got %d foods, want only 450856", len(foods))
	}
	ids, _ := srv.last("get_foods")["ids"].([]any)
	if len(ids) != 1 || ids[0] != float64(450856) {
		t.Errorf("requested ids = %v", ids)
	}
}

func TestMobileGetAllFoods_MissingRequestedFoodIsError(t *testing.T) {
	srv := newMobileTestServer(t)
	srv.override["get_foods"] = func(map[string]any) (int, string) { return 200, `{"foods":[],"messages":[]}` }
	if _, err := srv.client(t).GetAllFoods(context.Background(), []int64{450856}); err == nil {
		t.Fatal("expected an error when a requested food is absent")
	}
}

func TestMobileFindMyFoods(t *testing.T) {
	srv := newMobileTestServer(t)
	foods, err := srv.client(t).FindMyFoods(context.Background())
	if err != nil {
		t.Fatalf("FindMyFoods: %v", err)
	}
	// 15 foods + 26 recipes + 20 meals, all distinct.
	if len(foods) != 61 {
		t.Fatalf("got %d custom foods, want 61", len(foods))
	}
	seen := map[int64]bool{}
	for _, f := range foods {
		if f.ID <= 0 || f.Name == "" || seen[f.ID] {
			t.Errorf("bad or duplicate custom food %+v", f)
		}
		seen[f.ID] = true
	}
	for _, id := range []int64{81323197, 68943617, 80069621} {
		if !seen[id] {
			t.Errorf("owned food %d (recipe/meal) missing from FindMyFoods", id)
		}
	}
}

func TestMobileFindMyFoods_MissingListIsContractError(t *testing.T) {
	srv := newMobileTestServer(t)
	body := fixtureWithout(t, "testdata/mobile-get_myfoods.json", func(doc map[string]any) { delete(doc, "recipes") })
	srv.override["get_myfoods"] = func(map[string]any) (int, string) { return 200, body }
	_, err := srv.client(t).FindMyFoods(context.Background())
	if ce := asContractError(t, err); ce.Field != "recipes" {
		t.Errorf("contract error = %+v, want field recipes", ce)
	}
}

// TestMobileExpiredSession_RelogsInOnce covers a session the server no longer
// accepts: the client logs in again once and retries with the new key.
func TestMobileExpiredSession_RelogsInOnce(t *testing.T) {
	srv := newMobileTestServer(t)
	c := srv.client(t)
	logins := 0
	srv.override["login"] = func(map[string]any) (int, string) {
		logins++
		return 200, `{"result":"SUCCESS","id":10000001,"sessionKey":"session-2"}`
	}
	srv.override["get_myfoods"] = func(body map[string]any) (int, string) {
		if body["auth"].(map[string]any)["token"] == "session-1" {
			return 200, `{"result":"FAIL","error":"Invalid session"}`
		}
		return 200, loadFixture(t, "testdata/mobile-get_myfoods.json")
	}
	if _, err := c.FindMyFoods(context.Background()); err != nil {
		t.Fatalf("FindMyFoods after expiry: %v", err)
	}
	if logins != 1 {
		t.Errorf("re-logged in %d times, want 1", logins)
	}
}

func TestMobileExpiredSession_GivesUpAfterOneRetry(t *testing.T) {
	srv := newMobileTestServer(t)
	c := srv.client(t)
	srv.override["get_myfoods"] = func(map[string]any) (int, string) {
		return 200, `{"result":"FAIL","error":"Invalid session"}`
	}
	if _, err := c.FindMyFoods(context.Background()); err == nil {
		t.Fatal("expected an error after the retry also fails")
	}
	if n := srv.count("get_myfoods"); n != 2 {
		t.Errorf("get_myfoods called %d times, want 2", n)
	}
}

func TestMobileRateLimited(t *testing.T) {
	srv := newMobileTestServer(t)
	c := srv.client(t)
	srv.override["get_myfoods"] = func(map[string]any) (int, string) { return 429, `` }
	_, err := c.FindMyFoods(context.Background())
	if err == nil || !strings.Contains(err.Error(), "status 429") {
		t.Fatalf("expected a status 429 error, got %v", err)
	}
}

// TestMobileDeprecationHeader covers Cronometer's in-band sunset notice: the
// app's DeprecationInterceptor parses a "deprecation" response header as a UTC
// date, so the client must surface it before the API is switched off.
func TestMobileDeprecationHeader(t *testing.T) {
	srv := newMobileTestServer(t)
	srv.headers.Set("Deprecation", "2027-03-01T00:00:00Z")
	var got []DeprecationNotice
	c := NewMobileClient(&MobileClientOptions{
		BaseURL:               srv.srv.URL + "/api/v2/",
		DisableThrottleConfig: true,
		OnDeprecation:         func(n DeprecationNotice) { got = append(got, n) },
	})
	if err := c.Login(context.Background(), "user@example.com", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if len(got) != 1 || got[0].Header != "Deprecation" || got[0].Value != "2027-03-01T00:00:00Z" || got[0].Endpoint != "login" {
		t.Fatalf("deprecation notices = %+v", got)
	}
	if n := c.Deprecation(); n == nil || n.Value != "2027-03-01T00:00:00Z" {
		t.Errorf("Deprecation() = %+v", n)
	}
}

func TestMobileNoDeprecationHeader(t *testing.T) {
	srv := newMobileTestServer(t)
	if n := srv.client(t).Deprecation(); n != nil {
		t.Errorf("Deprecation() = %+v, want nil", n)
	}
}

func TestMobileThrottleConfig_UsesMobileLimit(t *testing.T) {
	throttle := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"apis":{"gwt_rpc":{"global":{"rps":40}},"mobile_v2":{"global":{"rps":250}}}}`)
	}))
	defer throttle.Close()
	srv := newMobileTestServer(t)
	c := NewMobileClient(&MobileClientOptions{BaseURL: srv.srv.URL + "/api/v2/", ThrottleConfigURL: throttle.URL})
	if err := c.Login(context.Background(), "user@example.com", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if want := intervalForRPS(250 * ThrottleSafetyFactor); c.limiter.interval != want {
		t.Errorf("limiter interval = %v, want %v (half the published mobile_v2 rate)", c.limiter.interval, want)
	}
}

func TestAPIContractError_Fingerprint(t *testing.T) {
	a := &APIContractError{Endpoint: "get_foods", Field: "name", Problem: "missing"}
	b := &APIContractError{Endpoint: "get_foods", Field: "name", Problem: "missing", Detail: "food 1"}
	c := &APIContractError{Endpoint: "get_diary", Field: "name", Problem: "missing"}
	if a.Fingerprint() != b.Fingerprint() {
		t.Error("fingerprint must ignore per-record detail so one break alerts once")
	}
	if a.Fingerprint() == c.Fingerprint() {
		t.Error("different endpoints must fingerprint differently")
	}
	if !strings.Contains(a.Error(), "get_foods") || !strings.Contains(a.Error(), "name") {
		t.Errorf("Error() = %q", a.Error())
	}
}

// TestMobileGetFood_UnlistedOwnedFood covers an owned food get_myfoods does not
// list (deleted since it was logged): it resolves without ingredients, and the
// classification is fetched once rather than on every lookup.
func TestMobileGetFood_UnlistedOwnedFood(t *testing.T) {
	srv := newMobileTestServer(t)
	body := fixtureWithout(t, "testdata/mobile-get_food-83483646.json", func(doc map[string]any) { doc["id"] = 99999 })
	srv.override["get_food"] = func(map[string]any) (int, string) { return 200, body }
	c := srv.client(t)
	for i := 0; i < 3; i++ {
		if _, err := c.GetFood(context.Background(), 99999); err != nil {
			t.Fatalf("GetFood: %v", err)
		}
	}
	if n := srv.count("get_myfoods"); n != 1 {
		t.Errorf("get_myfoods called %d times, want 1", n)
	}
}
