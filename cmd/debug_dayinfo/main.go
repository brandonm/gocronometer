// Command debug_dayinfo fetches a day's diary via GWT-RPC (getDayInfo) and prints
// the decoded servings with resolved food names — to validate the RPC path against
// the Cronometer web diary.
//
//	CRONOMETER_EMAIL=... CRONOMETER_PASSWORD=... go run ./cmd/debug_dayinfo 2026-06-28
//
// Run under `infisical run` so credentials are injected, never typed/printed.
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
	if len(os.Args) < 2 {
		log.Fatal("usage: debug_dayinfo YYYY-MM-DD")
	}
	date, err := time.Parse("2006-01-02", os.Args[1])
	if err != nil {
		log.Fatalf("bad date: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	c := gocronometer.NewClient(nil)
	if err := c.Login(ctx, os.Getenv("CRONOMETER_EMAIL"), os.Getenv("CRONOMETER_PASSWORD")); err != nil {
		log.Fatalf("login: %v", err)
	}

	servings, err := c.GetDayServings(ctx, date)
	if err != nil {
		log.Fatalf("GetDayServings: %v", err)
	}

	// Resolve food names in one batch.
	ids := make([]int64, 0, len(servings))
	for _, s := range servings {
		ids = append(ids, s.FoodID)
	}
	foods, _ := c.GetAllFoods(ctx, ids)

	fmt.Printf("\n=== decoded diary for %s — %d servings ===\n", date.Format("2006-01-02"), len(servings))
	for i, s := range servings {
		name := "(unresolved)"
		if f := foods[s.FoodID]; f != nil && f.Name != "" {
			name = f.Name
		}
		fmt.Printf("  %2d. amount=%-7g foodID=%-10d %s\n", i+1, s.Amount, s.FoodID, name)
	}
}
