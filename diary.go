package gocronometer

import (
	"time"
)

// DayServing is a single food entry from a day's diary, as returned by the
// mobile API's get_diary (the call the phone app uses to render the diary).
//
// Nutrient values are NOT embedded here — a serving references a food by ID plus
// an amount; resolve nutrients via GetAllFoods (per-100g) × grams, reusing the
// existing food cache. (This mirrors how custom-food ingredients are resolved.)
type DayServing struct {
	FoodID    int64     // Cronometer food ID (resolve nutrients via GetAllFoods)
	Amount    float64   // quantity in the measure's units (grams when MeasureID is the gram measure)
	MeasureID int64     // which measure the amount is expressed in
	MealGroup int       // meal category index (1=Breakfast, 2=Lunch, 3=Dinner, 4=Snacks, 5=Supplements, …)
	Date      time.Time // the diary day this serving belongs to (date only)
}

// mealGroupNames maps Cronometer's meal-group index to its display name.
var mealGroupNames = map[int]string{
	1: "Breakfast",
	2: "Lunch",
	3: "Dinner",
	4: "Snacks",
	5: "Supplements",
	6: "Water",
}

// MealGroupName returns the display name for a serving's MealGroup index,
// or "Uncategorized" for unknown/zero.
func MealGroupName(group int) string {
	if n, ok := mealGroupNames[group]; ok {
		return n
	}
	return "Uncategorized"
}
