package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jrmycanady/gocronometer"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := gocronometer.NewClient(nil)
	if err := client.Login(ctx, os.Getenv("CRONOMETER_EMAIL"), os.Getenv("CRONOMETER_PASSWORD")); err != nil {
		log.Fatal(err)
	}

	// Get Basic Sandwich ingredients
	fmt.Println("=== Basic Sandwich (67861120) ingredients:")
	detail, err := client.GetFood(ctx, 67861120)
	ingredients := detail.Ingredients
	if err != nil {
		log.Fatal(err)
	}
	for _, ing := range ingredients {
		fmt.Printf("  FoodID=%d  MeasureID=%d  Amount=%.1f\n", ing.FoodID, ing.MeasureID, ing.Amount)
	}

	// Try to export each ingredient to get its name
	fmt.Println("\n=== Exporting each ingredient:")
	for _, ing := range ingredients {
		time.Sleep(2 * time.Second)
		export, err := client.ExportFood(ctx, ing.FoodID, ing.MeasureID, ing.Amount)
		if err != nil {
			fmt.Printf("  FoodID=%d (%.1fg): FAILED - %v\n", ing.FoodID, ing.Amount, err)
			continue
		}
		fmt.Printf("  FoodID=%d: %q (%.1fg) - %.1f kcal\n", ing.FoodID, export.FoodName, ing.Amount, export.Nutrients["Energy (kcal)"])
	}
}
