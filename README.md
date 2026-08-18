# gocronometer

gocronometer is a GPLv2-licensed Go module that provides a client for reading your
own data out of [Cronometer](https://cronometer.com). It speaks the same unpublished
API the Cronometer single-page app uses.

> **This is [@brandonm](https://github.com/brandonm)'s fork**, ahead of upstream
> `jrmycanady/gocronometer` (v1.5.1). It adds quota-free diary reading over GWT-RPC,
> custom-food/recipe resolution, and automatic rate limiting. The module path is kept
> as `github.com/jrmycanady/gocronometer` so it drops into existing code via a
> `replace` directive — see [Using this fork](#using-this-fork).

**IMPORTANT:** This module uses the same API the SPA uses. For that reason it should
only be used by single users wanting to read/back-up their *own* personal data. It
should never be used for integrations that the enterprise plan would cover. The
library is licensed under the GPLv2 to help discourage unacceptable usage.

**IMPORTANT:** Always scope your requests to a reasonable date range. Never
continually re-read all data or data you already have.

---

## What's new in this fork

| Capability | Why it matters |
|---|---|
| **Diary reading over GWT-RPC** (`GetDayServings`) | The legacy CSV `/export` endpoint is capped at roughly **10 exports per day**. `getDayInfo` is the same call the web diary uses — **no daily quota** — and it returns **water servings**, which the CSV export drops. |
| **Custom foods & recipes** (`FindMyFoods`, `GetFood`, `GetAllFoods`) | Resolve a serving's food ID to its name, source, per-100 g nutrients, measures, and (for recipes) its ingredient list. |
| **Automatic rate limiting** | The client throttles every request and, on login, fetches Cronometer's published `throttle-config` to honor the real `gwt_rpc` limit. Nothing to wire up. |

---

## Install

```bash
go get github.com/jrmycanady/gocronometer
```

### Using this fork

The module keeps the upstream path, so point your `go.mod` at this fork with a
`replace` directive and import the original path:

```
// go.mod
require github.com/jrmycanady/gocronometer v1.5.1
replace github.com/jrmycanady/gocronometer => github.com/brandonm/gocronometer v1.6.0
```

```go
import "github.com/jrmycanady/gocronometer"
```

---

## Quick start — read a day's diary (recommended)

```go
// NewClient installs an automatic rate limiter (see Rate limiting below).
c := gocronometer.NewClient(nil)

// Login also fetches Cronometer's published throttle-config and tunes the limiter.
if err := c.Login(context.Background(), username, password); err != nil {
	log.Fatalf("login failed: %s", err)
}
defer c.Logout(context.Background())

// Read one day's servings via GWT-RPC (getDayInfo) — no daily export quota.
day := time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC)
servings, err := c.GetDayServings(context.Background(), day)
if err != nil {
	log.Fatalf("getDayInfo failed: %s", err)
}

for _, s := range servings {
	fmt.Printf("%-12s food=%d amount=%.2f\n",
		gocronometer.MealGroupName(s.MealGroup), s.FoodID, s.Amount)
}
```

`getDayInfo` is a **single-day** call, so to cover a range, loop it across the dates
you need.

### `DayServing`

```go
type DayServing struct {
	FoodID    int64     // Cronometer food ID (resolve nutrients via GetAllFoods)
	Amount    float64   // quantity in the measure's units (grams when MeasureID is the gram measure)
	MeasureID int64     // which measure the amount is expressed in
	MealGroup int       // 1=Breakfast 2=Lunch 3=Dinner 4=Snacks 5=Supplements 6=Water
	Date      time.Time // the diary day this serving belongs to
}
```

`MealGroupName(group int) string` maps the `MealGroup` index to its display name.

A `DayServing` references a food by ID plus an amount; nutrient values are not
embedded. Resolve them with the custom-food calls below (per-100 g × grams), reusing
the food cache.

---

## Custom foods, recipes & nutrients

| func | description |
|---|---|
| `FindMyFoods(ctx) ([]CustomFood, error)` | Lists your custom foods/recipes/meals (`ID`, `Name`). |
| `GetFood(ctx, foodID) (*FoodDetail, error)` | Full detail for one food: name, source, ingredients (if a recipe), per-100 g nutrients, and measures. |
| `GetAllFoods(ctx, foodIDs) (map[int64]*FoodDetail, error)` | Batch `GetFood` — resolve many food IDs at once. |
| `GetFoodIngredients(ctx, foodID) ([]RecipeIngredient, error)` | Just the ingredient list for a recipe/meal. |

```go
type FoodDetail struct {
	FoodID           int64
	Name             string
	Source           string             // e.g. "NCCDB", "CRDB", "custom"
	Ingredients      []RecipeIngredient // populated for recipes/meals
	NutrientsPer100g map[int]float64    // keyed by USDA nutrient code; per 100 g
	Measures         []FoodMeasure      // measure ID -> grams (convert amount → grams)
}
```

---

## Rate limiting

`NewClient` installs an `http.RoundTripper` that paces every request to a safe
default (`DefaultGWTRateLimit`). On `Login`, the client fetches Cronometer's published
limits from `/api/v3/throttle-config` and tunes itself to a fraction of the real
`gwt_rpc` rate, leaving headroom for any concurrent web/app sessions. This is
automatic — you don't need to call anything.

You can inspect the published limits yourself:

```go
cfg, err := c.GetThrottleConfig(context.Background())
```

---

## Legacy CSV exports

The original CSV `/export` endpoint is still available, but is subject to
Cronometer's **~10 exports/day** quota and **omits water servings**. Prefer
`GetDayServings` for diary data. The exports return raw CSV.

| func | description |
|---|---|
| `ExportDailyNutrition()` | Daily nutrition for the date range. |
| `ExportServings()` | Servings for the date range. |
| `ExportExercises()` | Exercises for the date range. |
| `ExportBiometrics()` | Biometrics for the date range. |
| `ExportNotes()` | Notes for the date range. |

### Parsing CSV exports

| func | parses output of |
|---|---|
| `ParseServingsExport()` | `ExportDailyNutrition()`, `ExportServings()` |
| `ParseExerciseExport()` | `ExportExercises()` |
| `ParseBiometricRecordsExport()` | `ExportBiometrics()` |

The `*Parsed` helpers (`ExportServingsParsed`, etc.) export and parse in one call.

---

## API Magic Values

This library mimics the GWT HTTP requests Cronometer's deployed GWT application makes.
Several values can only be obtained by loading the application itself, and they change
over time with application updates. The library ships known-good defaults, but `Login`
also discovers the live permutation and serialization policy from Cronometer's public
GWT assets before authenticating.

Compatible application rebuilds are adopted automatically. The library compares the
serialization signatures of every class it decodes; if a watched layout changed,
`Login` returns `*GWTWireIncompatibleError` before collecting diary data. The observed
`GWTWireStatus` includes the live build identifiers, watched class hashes, a stable
schema fingerprint, and the incompatible classes (if any).

Call `RefreshGWTWireStatus` to repeat the check on a long-lived client, or inspect the
last result with `GWTWireStatus`. `ClientOptions.OnGWTWireStatus` can persist or report
every discovered status. `DisableGWTWireCheck` exists for controlled offline use; it
removes the pre-login compatibility guard.

| Name | Location | Changes |
|---|---|---|
| `GWTContentType` | Request header. | false |
| `GWTModuleBase` | Request header. | false |
| `GWTPermutation` | Request header. | true |
| `GWTHeader` | GWT request body. | true |

Explicit `GWTPermutation` and `GWTHeader` options remain useful for tests and emergency
overrides. With the default compatibility check enabled, compatible live values replace
those request identifiers during login.

---

## Testing

```bash
go test ./...
```

Offline unit tests (GWT deserializers, custom-food parsing, diary decoding) run
with no setup. The live integration tests in `gocronometer_test.go` hit the real
Cronometer API and **skip** unless `GOCRONOMETER_TEST_USERNAME` and
`GOCRONOMETER_TEST_PASSWORD` are set — so CI stays green without secrets, and you can
run the full suite against your own account by exporting both.

---

## License

GPLv2. Forked from [jrmycanady/gocronometer](https://github.com/jrmycanady/gocronometer).
