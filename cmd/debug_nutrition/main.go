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
	date, _ := time.Parse("2006-01-02", os.Args[1])
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	c := gocronometer.NewClient(nil)
	if err := c.Login(ctx, os.Getenv("CRONOMETER_EMAIL"), os.Getenv("CRONOMETER_PASSWORD")); err != nil {
		log.Fatal(err)
	}
	servings, _ := c.GetDayServings(ctx, date)
	cache := map[int64]*gocronometer.FoodDetail{}
	get := func(id int64) *gocronometer.FoodDetail {
		if f, ok := cache[id]; ok { return f }
		f, _ := c.GetFood(ctx, id)
		cache[id] = f
		time.Sleep(150 * time.Millisecond)
		return f
	}
	// nutrients for an amount: DB food -> per100g*amount/100 ; recipe -> sum ingredients * amount
	var nut func(id int64, amount float64) (kcal, water float64)
	nut = func(id int64, amount float64) (kcal, water float64) {
		f := get(id)
		if f == nil { return }
		if len(f.Ingredients) > 0 {
			for _, ing := range f.Ingredients {
				k, w := nut(ing.FoodID, ing.Amount) // ingredient amounts are grams
				kcal += k * amount
				water += w * amount
			}
			return
		}
		kcal = f.NutrientsPer100g[208] * amount / 100
		water = f.NutrientsPer100g[255] * amount / 100
		return
	}
	var totK, totW float64
	for _, s := range servings {
		f := get(s.FoodID)
		k, w := nut(s.FoodID, s.Amount)
		totK += k; totW += w
		name := f.Name; if len(name) > 42 { name = name[:42] }
		fmt.Printf("%-43s amt=%-8.2f kcal=%-7.1f water=%-7.1f\n", name, s.Amount, k, w)
	}
	fmt.Printf("\nDAY TOTAL: %.0f kcal | water %.0f mL = %.2f fl oz\n", totK, totW, totW/29.5735)
}
