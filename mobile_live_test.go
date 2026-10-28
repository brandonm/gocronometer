package gocronometer

// Live cross-check of MobileClient against the GWT Client: both log in to the
// real account and must return identical servings and foods for recent days.
// Opt-in, because it needs credentials and makes network calls:
//
//	GOCRONOMETER_MOBILE_PROBE=1 GOCRONOMETER_TEST_USERNAME=... GOCRONOMETER_TEST_PASSWORD=... \
//	  go test -run TestMobileLiveMatchesGWT -v .

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"
)

func TestMobileLiveMatchesGWT(t *testing.T) {
	if os.Getenv("GOCRONOMETER_MOBILE_PROBE") == "" {
		t.Skip("set GOCRONOMETER_MOBILE_PROBE=1 and test credentials to run")
	}
	user, pass := os.Getenv("GOCRONOMETER_TEST_USERNAME"), os.Getenv("GOCRONOMETER_TEST_PASSWORD")
	if user == "" || pass == "" {
		t.Fatal("GOCRONOMETER_TEST_USERNAME / GOCRONOMETER_TEST_PASSWORD not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	mobile := NewMobileClient(&MobileClientOptions{
		OnDeprecation: func(n DeprecationNotice) { t.Logf("DEPRECATION NOTICE: %+v", n) },
	})
	if err := mobile.Login(ctx, user, pass); err != nil {
		t.Fatalf("mobile login: %v", err)
	}
	gwt := NewClient(nil)
	if err := gwt.Login(ctx, user, pass); err != nil {
		t.Fatalf("GWT login: %v", err)
	}

	myFoods, err := mobile.FindMyFoods(ctx)
	if err != nil {
		t.Fatalf("mobile FindMyFoods: %v", err)
	}
	t.Logf("FindMyFoods: %d owned foods", len(myFoods))

	today := time.Now()
	checkedDays := 0
	for back := 1; back <= 14 && checkedDays < 3; back++ {
		day := time.Date(today.Year(), today.Month(), today.Day()-back, 0, 0, 0, 0, time.UTC)
		ms, err := mobile.GetDayServings(ctx, day)
		if err != nil {
			t.Fatalf("%s mobile GetDayServings: %v", day.Format("2006-01-02"), err)
		}
		if len(ms) == 0 {
			continue
		}
		gs, err := gwt.GetDayServings(ctx, day)
		if err != nil {
			t.Fatalf("%s GWT GetDayServings: %v", day.Format("2006-01-02"), err)
		}
		if a, b := servingKeys(ms), servingKeys(gs); fmt.Sprint(a) != fmt.Sprint(b) {
			t.Errorf("%s servings differ:\n mobile %v\n GWT    %v", day.Format("2006-01-02"), a, b)
		}

		ids := map[int64]bool{}
		for _, s := range ms {
			ids[s.FoodID] = true
		}
		var list []int64
		for id := range ids {
			list = append(list, id)
		}
		mf, err := mobile.GetAllFoods(ctx, list)
		if err != nil {
			t.Fatalf("%s mobile GetAllFoods: %v", day.Format("2006-01-02"), err)
		}
		gf, err := gwt.GetAllFoods(ctx, list)
		if err != nil {
			t.Fatalf("%s GWT GetAllFoods: %v", day.Format("2006-01-02"), err)
		}
		nutrients := 0
		for _, id := range list {
			m, g := mf[id], gf[id]
			if m == nil || g == nil {
				t.Errorf("food %d: mobile=%v GWT=%v", id, m != nil, g != nil)
				continue
			}
			if fmt.Sprint(m.NutrientsPer100g) != fmt.Sprint(g.NutrientsPer100g) {
				t.Errorf("food %d (%s): nutrients differ", id, m.Name)
			}
			if fmt.Sprint(m.Measures) != fmt.Sprint(g.Measures) {
				t.Errorf("food %d (%s): measures differ", id, m.Name)
			}
			if fmt.Sprint(m.Ingredients) != fmt.Sprint(g.Ingredients) {
				t.Errorf("food %d (%s): ingredients differ", id, m.Name)
			}
			if m.Source != g.Source {
				t.Errorf("food %d: source mobile %q GWT %q", id, m.Source, g.Source)
			}
			if m.Name != g.Name {
				t.Logf("food %d: display name mobile %q, GWT %q", id, m.Name, g.Name)
			}
			nutrients += len(m.NutrientsPer100g)
		}
		t.Logf("%s: %d servings, %d foods, %d nutrient values compared", day.Format("2006-01-02"), len(ms), len(list), nutrients)
		checkedDays++
	}
	if checkedDays == 0 {
		t.Fatal("no day with servings in the last 14 days")
	}
	if n := mobile.Deprecation(); n != nil {
		t.Logf("server deprecation notice: %+v", *n)
	} else {
		t.Log("no deprecation notice from the server")
	}
}

func servingKeys(ss []DayServing) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		out = append(out, fmt.Sprintf("%d/%d/%v/%d", s.FoodID, s.MeasureID, s.Amount, s.MealGroup))
	}
	sort.Strings(out)
	return out
}
