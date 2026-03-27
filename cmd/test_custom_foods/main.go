package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/brandonm/gocronometer"
)

func main() {
	username := os.Getenv("CRONOMETER_EMAIL")
	password := os.Getenv("CRONOMETER_PASSWORD")
	if username == "" || password == "" {
		log.Fatal("Set CRONOMETER_EMAIL and CRONOMETER_PASSWORD env vars")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := gocronometer.NewClient(nil)

	// 1. Login
	fmt.Println("=== Logging in...")
	if err := client.Login(ctx, username, password); err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	fmt.Printf("Logged in. UserID=%s Nonce=%s\n\n", client.UserID, client.Nonce)

	// 2. FindMyFoods
	fmt.Println("=== FindMyFoods...")
	foods, err := client.FindMyFoods(ctx)
	if err != nil {
		log.Fatalf("FindMyFoods failed: %v", err)
	}
	fmt.Printf("Found %d custom foods:\n", len(foods))
	for _, f := range foods {
		fmt.Printf("  ID=%d  Name=%q\n", f.ID, f.Name)
	}
	fmt.Println()

	if len(foods) == 0 {
		fmt.Println("No custom foods to test with.")
		return
	}

	// 3. GetFood (get ingredients for first custom food)
	testFood := foods[0]
	fmt.Printf("=== GetFood(%d) [%s]...\n", testFood.ID, testFood.Name)
	ingredients, err := client.GetFood(ctx, testFood.ID)
	if err != nil {
		log.Fatalf("GetFood failed: %v", err)
	}
	if len(ingredients) == 0 {
		fmt.Println("  No ingredients (simple food, not a recipe)")
	} else {
		fmt.Printf("  %d ingredients:\n", len(ingredients))
		for _, ing := range ingredients {
			fmt.Printf("    FoodID=%d  MeasureID=%d  Amount=%.2f\n", ing.FoodID, ing.MeasureID, ing.Amount)
		}
	}
	fmt.Println()

	// 4. ExportFood for each ingredient
	// Build a set of custom food IDs for detecting nested recipes
	customFoodIDs := make(map[int64]bool)
	for _, f := range foods {
		customFoodIDs[f.ID] = true
	}

	if len(ingredients) > 0 {
		fmt.Println("=== ExportFood for each ingredient...")
		for _, ing := range ingredients {
			time.Sleep(1 * time.Second) // rate limit

			// For nested recipes (ingredient is itself a custom food), use grams=0
			// to get the full recipe serving. For regular foods, use the gram amount.
			grams := ing.Amount
			if customFoodIDs[ing.FoodID] {
				grams = 0 // use default serving (full recipe)
				fmt.Printf("  [nested recipe] ")
			} else {
				fmt.Printf("  ")
			}

			export, err := client.ExportFood(ctx, ing.FoodID, ing.MeasureID, grams)
			if err != nil {
				fmt.Printf("FoodID=%d: export failed: %v\n", ing.FoodID, err)
				continue
			}
			fmt.Printf("%s (ID=%d, %.1f): %.1f kcal, %.1fg protein, %.1fg carbs, %.1fg fat\n",
				export.FoodName, export.FoodID, ing.Amount,
				export.Nutrients["Energy (kcal)"],
				export.Nutrients["Protein (g)"],
				export.Nutrients["Carbs (g)"],
				export.Nutrients["Fat (g)"],
			)
		}
	}

	// 5. Also export the top-level food itself for comparison
	fmt.Printf("\n=== ExportFood for %s (aggregate)...\n", testFood.Name)
	// Need a measure ID for the top-level food — use GetFood to find it
	// For now, try exporting with measure ID 0 (grams=1 mode)
	// Actually we need the real measure ID. Let's get it from GetFood response.
	// The measure IDs for the food itself appear in the GWT response but our parser
	// only extracts ingredient measure IDs. For now, skip this.
	fmt.Println("  (skipped — need measure ID from recipe definition)")

	fmt.Println("\n=== Done!")
}
