package gocronometer

// Ad-hoc offline check: run the batch deserializer against a getAllFood
// response captured from the live web app (e.g. via Playwright devtools).
// Use this to validate a fresh capture after a Cronometer deploy before
// promoting it to a permanent fixture in testdata/ (see
// gwt_food_newformat_test.go for the committed regression tests).
//
//	GOCRONOMETER_WIRE_FIXTURE=/path/to/body.txt \
//	GOCRONOMETER_WIRE_FIXTURE_IDS=450230,7345420 \
//	go test -run TestWireOfflineFixture -v .

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestWireOfflineFixture(t *testing.T) {
	path := os.Getenv("GOCRONOMETER_WIRE_FIXTURE")
	if path == "" {
		t.Skip("GOCRONOMETER_WIRE_FIXTURE not set")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var ids []int64
	if raw := os.Getenv("GOCRONOMETER_WIRE_FIXTURE_IDS"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
			if err != nil {
				t.Fatalf("bad id %q in GOCRONOMETER_WIRE_FIXTURE_IDS: %v", part, err)
			}
			ids = append(ids, id)
		}
	} else {
		// Default: the 2026-07-17 burrito capture's request IDs.
		ids = []int64{450230, 7345420, 15431433, 67859683, 450856, 67861120, 13605711, 67915870, 462802, 31290980}
	}

	foods, err := parseGetAllFoodsResponse(string(body), ids)
	if err != nil {
		t.Fatalf("parseGetAllFoodsResponse: %v", err)
	}
	t.Logf("parsed %d foods from fixture", len(foods))
	for _, id := range ids {
		f, ok := foods[id]
		if !ok {
			t.Errorf("food %d: MISSING from result", id)
			continue
		}
		t.Logf("food %d: name=%q source=%q nutrients=%d ingredients=%d measures=%d",
			id, f.Name, f.Source, len(f.NutrientsPer100g), len(f.Ingredients), len(f.Measures))
		if f.Name == "" {
			t.Errorf("food %d: EMPTY NAME (husk)", id)
		}
	}
}
