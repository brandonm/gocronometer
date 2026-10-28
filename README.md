# gocronometer

gocronometer is a GPLv2-licensed Go module that provides a client for reading your
own data out of [Cronometer](https://cronometer.com). It speaks the same unpublished
JSON API the Cronometer phone app uses (`mobile.cronometer.com/api/v2`).

> Originally derived from [`jrmycanady/gocronometer`](https://github.com/jrmycanady/gocronometer),
> which scraped the web app and its CSV export. Since v2 this is an independent
> module: a client for the mobile API with quota-free diary reading,
> custom-food/recipe resolution, and automatic rate limiting.

**IMPORTANT:** This module uses the same API the Cronometer app uses. For that reason
it should only be used by single users wanting to read/back-up their *own* personal
data. It should never be used for integrations that the enterprise plan would cover.
The library is licensed under the GPLv2 to help discourage unacceptable usage.

**IMPORTANT:** Always scope your requests to a reasonable date range. Never
continually re-read all data or data you already have.

---

## Why the mobile API

Through v1.7, this fork read Cronometer's web app over GWT-RPC. GWT payloads are
positional token streams, and the web app is redeployed to every visitor at once,
so each Cronometer release that added a field to a serialized class broke every
reader (2026-07-16, 2026-08-24, 2026-10-28). The mobile API carries named JSON
fields and has to keep serving app builds users have not updated, so it changes far
less abruptly — and it announces sunsets in a response header ahead of time.

**v2.0.0 removed the web/GWT client** (`Client`, `NewClient`, the GWT decoders, and
the CSV `Export*` calls, which depended on the web login) and moved the module to
its own path, `github.com/brandonm/gocronometer/v2`. The CSV `Parse*` functions
remain for exports obtained elsewhere. v1.x was published under the upstream path
`github.com/jrmycanady/gocronometer` and consumed through a `replace` directive.

---

## Install

```bash
go get github.com/brandonm/gocronometer/v2
```

```go
import "github.com/brandonm/gocronometer/v2" // package gocronometer
```

---

## Quick start

```go
c := gocronometer.NewMobileClient(&gocronometer.MobileClientOptions{
	// Cronometer announces API sunsets in a response header; act before the date.
	OnDeprecation: func(n gocronometer.DeprecationNotice) { log.Printf("deprecation: %+v", n) },
})
if err := c.Login(ctx, username, password); err != nil {
	log.Fatal(err)
}

day := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
servings, err := c.GetDayServings(ctx, day) // get_diary
if err != nil {
	log.Fatal(err)
}
for _, s := range servings {
	fmt.Printf("%-12s food=%d amount=%.2f\n", gocronometer.MealGroupName(s.MealGroup), s.FoodID, s.Amount)
}
```

`GetDayServings` is a **single-day** call; loop it across the dates you need.

| method | endpoint | returns |
|---|---|---|
| `Login(ctx, email, password)` | `login` | — |
| `GetDayServings(ctx, day)` | `get_diary` | `[]DayServing` (biometric/exercise entries skipped) |
| `GetFood(ctx, foodID)` | `get_food` | `*FoodDetail`, with ingredients for recipes/meals |
| `GetAllFoods(ctx, foodIDs)` | `get_foods` (+ `get_food` per owned food) | `map[int64]*FoodDetail` |
| `FindMyFoods(ctx)` | `get_myfoods` | `[]CustomFood`: your custom foods, recipes, and meals |
| `Deprecation()` | any response | the latest sunset notice, or nil |

```go
type DayServing struct {
	FoodID    int64     // Cronometer food ID (resolve nutrients via GetAllFoods)
	Amount    float64   // quantity in the measure's units (grams when MeasureID is the gram measure)
	MeasureID int64     // which measure the amount is expressed in
	MealGroup int       // 1=Breakfast 2=Lunch 3=Dinner 4=Snacks 5=Supplements 6=Water
	Date      time.Time // the diary day this serving belongs to
}

type FoodDetail struct {
	FoodID           int64
	Name             string
	Source           string             // e.g. "NCCDB:4974", "CRDB", "Custom"
	Ingredients      []RecipeIngredient // populated for recipes/meals
	NutrientsPer100g map[int]float64    // keyed by USDA nutrient code; per 100 g
	Measures         []FoodMeasure      // measure ID -> grams (convert amount → grams)
}
```

`USDANutrientNames` maps nutrient codes to names and units.

---

## Behavior worth knowing

- **Fails loudly on shape changes.** A renamed or dropped JSON field would otherwise
  decode as a silent zero. Every field the client relies on is checked, and a
  missing one returns `*APIContractError{Endpoint, Field, Problem, Detail}` — with a
  `Fingerprint()` for alert de-duplication — instead of data.
- **Recipes keep their ingredients.** `get_foods` omits ingredient lists, so
  `GetAllFoods` re-fetches every food you own with `get_food`. `get_food` omits the
  `ingredients` key for plain custom foods, so its absence is an error only for foods
  `get_myfoods` lists as recipes or meals.
- **Login never writes your timezone.** The login endpoint treats a non-null
  `timezone` as a write to the account setting; the client always sends null.
- **Sessions renew once, transparently** when the server rejects a session.
- **Rate limiting is automatic.** Requests are paced to `DefaultMobileRateLimit`
  until `Login` fetches Cronometer's published `throttle-config`, then to
  `ThrottleSafetyFactor` (half) of the `mobile_v2` limit.
- Accounts with two-factor authentication are not supported.

---

## Parsing CSV exports

For CSV exports downloaded from Cronometer yourself:

| func | parses |
|---|---|
| `ParseServingsExport()` | daily nutrition / servings exports |
| `ParseExerciseExport()` | exercise exports |
| `ParseBiometricRecordsExport()` | biometric exports |
| `ParseFoodExport()` | single-food exports |

---

## Testing

```bash
go test ./...
```

Unit tests run offline against live responses captured from the mobile API
(`testdata/mobile-*.json`, user ID scrubbed). Before GWT was removed, the same diary
day was decoded through the GWT client; `testdata/golden-gwt-*.json` freeze that
decode, and the tests require the mobile client to match it exactly.

The live test skips unless explicitly enabled:

```bash
GOCRONOMETER_MOBILE_PROBE=1 GOCRONOMETER_TEST_USERNAME=... GOCRONOMETER_TEST_PASSWORD=... \
  go test -run TestMobileLive -v .
```

---

## License

GPLv2. Originally derived from [jrmycanady/gocronometer](https://github.com/jrmycanady/gocronometer).
