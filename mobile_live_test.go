package gocronometer

// Live smoke test of MobileClient against the real account. Opt-in, because it
// needs credentials and makes network calls:
//
//	GOCRONOMETER_MOBILE_PROBE=1 GOCRONOMETER_TEST_USERNAME=... GOCRONOMETER_TEST_PASSWORD=... \
//	  go test -run TestMobileLive -v .
//
// Before GWT was removed this cross-checked every value against the GWT
// client on live data (identical on 2026-10-25..27); testdata/golden-gwt-*.json
// preserve that comparison for the unit tests.

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestMobileLive(t *testing.T) {
	if os.Getenv("GOCRONOMETER_MOBILE_PROBE") == "" {
		t.Skip("set GOCRONOMETER_MOBILE_PROBE=1 and test credentials to run")
	}
	user, pass := os.Getenv("GOCRONOMETER_TEST_USERNAME"), os.Getenv("GOCRONOMETER_TEST_PASSWORD")
	if user == "" || pass == "" {
		t.Fatal("GOCRONOMETER_TEST_USERNAME / GOCRONOMETER_TEST_PASSWORD not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	c := NewMobileClient(&MobileClientOptions{
		OnDeprecation: func(n DeprecationNotice) { t.Logf("DEPRECATION NOTICE: %+v", n) },
	})
	if err := c.Login(ctx, user, pass); err != nil {
		t.Fatalf("login: %v", err)
	}
	myFoods, err := c.FindMyFoods(ctx)
	if err != nil {
		t.Fatalf("FindMyFoods: %v", err)
	}
	t.Logf("FindMyFoods: %d owned foods", len(myFoods))

	today := time.Now()
	checkedDays := 0
	for back := 1; back <= 14 && checkedDays < 3; back++ {
		day := time.Date(today.Year(), today.Month(), today.Day()-back, 0, 0, 0, 0, time.UTC)
		servings, err := c.GetDayServings(ctx, day)
		if err != nil {
			t.Fatalf("%s GetDayServings: %v", day.Format("2006-01-02"), err)
		}
		if len(servings) == 0 {
			continue
		}
		ids := map[int64]bool{}
		for _, s := range servings {
			if _, ok := mealGroupNames[s.MealGroup]; !ok {
				t.Errorf("serving of food %d has unknown meal group %d", s.FoodID, s.MealGroup)
			}
			ids[s.FoodID] = true
		}
		var list []int64
		for id := range ids {
			list = append(list, id)
		}
		foods, err := c.GetAllFoods(ctx, list)
		if err != nil {
			t.Fatalf("%s GetAllFoods: %v", day.Format("2006-01-02"), err)
		}
		nutrients := 0
		for _, id := range list {
			f := foods[id]
			if f == nil || f.Name == "" || len(f.NutrientsPer100g) == 0 || len(f.Measures) == 0 {
				t.Errorf("food %d did not resolve fully: %+v", id, f)
				continue
			}
			nutrients += len(f.NutrientsPer100g)
			// Database foods carry physical constants — the sharpest check
			// that measure weights landed in the right field.
			if f.Source == "Custom" {
				continue
			}
			for _, m := range f.Measures {
				if (m.Name == "g" && m.Grams != 1) || (m.Name == "oz" && m.Grams != 28.3495231) {
					t.Errorf("food %d (%s): measure %q weighs %v g", id, f.Name, m.Name, m.Grams)
				}
			}
		}
		t.Logf("%s: %d servings, %d foods, %d nutrient values", day.Format("2006-01-02"), len(servings), len(list), nutrients)
		checkedDays++
	}
	if checkedDays == 0 {
		t.Fatal("no day with servings in the last 14 days")
	}
	if n := c.Deprecation(); n != nil {
		t.Logf("server deprecation notice: %+v", *n)
	} else {
		t.Log("no deprecation notice from the server")
	}
}
