package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
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

	foodID := int64(466098)
	if len(os.Args) > 1 {
		foodID, _ = strconv.ParseInt(os.Args[1], 10, 64)
	}

	fmt.Printf("=== GetFood(%d):\n", foodID)
	detail, err := client.GetFood(ctx, foodID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("  Name: %q\n", detail.Name)
	fmt.Printf("  Source: %q\n", detail.Source)
	fmt.Printf("  Ingredients: %d\n", len(detail.Ingredients))
	fmt.Printf("  Nutrients (per 100g): %d entries\n", len(detail.NutrientsPer100g))

	// Print nutrients with names
	for code, value := range detail.NutrientsPer100g {
		if info, ok := gocronometer.USDANutrientNames[code]; ok {
			fmt.Printf("    %s (code %d): %.2f %s\n", info.Name, code, value, info.Unit)
		} else {
			fmt.Printf("    code %d: %.4f\n", code, value)
		}
	}

	// Show scaled for 48g (lettuce amount in sandwich)
	if foodID == 466098 {
		fmt.Println("\n=== Scaled for 48g (sandwich amount):")
		scale := 48.0 / 100.0
		for code, value := range detail.NutrientsPer100g {
			if info, ok := gocronometer.USDANutrientNames[code]; ok {
				scaled := value * scale
				if scaled > 0.001 {
					fmt.Printf("    %s: %.2f %s\n", info.Name, scaled, info.Unit)
				}
			}
		}
	}

	// Also test food 450230 (Monin Dark Chocolate - the other 403)
	if foodID == 466098 {
		fmt.Println("\n=== GetFood(450230) - Monin Dark Chocolate:")
		time.Sleep(1 * time.Second)
		detail2, err := client.GetFood(ctx, 450230)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
		} else {
			fmt.Printf("  Name: %q\n", detail2.Name)
			fmt.Printf("  Source: %q\n", detail2.Source)
			fmt.Printf("  Nutrients (per 100g): %d entries\n", len(detail2.NutrientsPer100g))
			for code, value := range detail2.NutrientsPer100g {
				if info, ok := gocronometer.USDANutrientNames[code]; ok {
					fmt.Printf("    %s: %.2f %s\n", info.Name, code, value, info.Unit)
				}
			}
		}
	}
}
