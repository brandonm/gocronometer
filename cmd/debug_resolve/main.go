package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jrmycanady/gocronometer"
)

func main() {
	c := gocronometer.NewClient(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := c.Login(ctx, os.Getenv("CRONOMETER_EMAIL"), os.Getenv("CRONOMETER_PASSWORD")); err != nil {
		fmt.Println("login:", err)
		return
	}
	for _, a := range os.Args[1:] {
		id, _ := strconv.ParseInt(a, 10, 64)
		d, err := c.GetFood(ctx, id)
		if err != nil || d == nil || d.Name == "" {
			fmt.Printf("  %12s -> (not a food: %v)\n", a, err)
		} else {
			fmt.Printf("  %12s -> %q (source=%s)\n", a, d.Name, d.Source)
		}
		time.Sleep(300 * time.Millisecond)
	}
}
