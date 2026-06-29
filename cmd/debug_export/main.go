// Command debug_export isolates the Cronometer CSV /export call that
// auto-journal's daily collection depends on. It logs in and runs
// ExportServings for a date range, reporting whether the request SUCCEEDED or
// was RATE LIMITED (HTTP 429) — with none of the auto-journal app around it.
//
// The point: run it from BOTH your normal machine and the box where
// auto-journal is deployed. If export works from one IP but 429s from the
// other, the limit is account/IP-quota related, not a code bug. If a single
// fresh export 429s everywhere, the account's daily export quota is exhausted.
//
// Usage:
//
//	CRONOMETER_EMAIL=you@example.com CRONOMETER_PASSWORD=secret \
//	    go run ./cmd/debug_export [-n attempts] [start YYYY-MM-DD] [end YYYY-MM-DD]
//
// -n probes the daily quota by exporting repeatedly; EACH attempt counts
// against the ~10/day limit, so use it sparingly.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jrmycanady/gocronometer"
)

func main() {
	count := flag.Int("n", 1, "number of export attempts (each counts against the daily quota)")
	flag.Parse()

	user := os.Getenv("CRONOMETER_EMAIL")
	pass := os.Getenv("CRONOMETER_PASSWORD")
	if user == "" || pass == "" {
		log.Fatal("Set CRONOMETER_EMAIL and CRONOMETER_PASSWORD env vars")
	}

	// Date range: optional [start] [end] args, else yesterday..today.
	now := time.Now().UTC()
	start, end := now.AddDate(0, 0, -1), now
	if a := flag.Args(); len(a) >= 1 {
		start = mustDate(a[0])
		if len(a) >= 2 {
			end = mustDate(a[1])
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	client := gocronometer.NewClient(nil)

	fmt.Println("=== Login")
	if err := client.Login(ctx, user, pass); err != nil {
		log.Fatalf("login FAILED: %v", err)
	}
	fmt.Printf("login OK (UserID=%s)\n\n", client.UserID)

	fmt.Printf("=== ExportServings %s .. %s  (%d attempt(s))\n",
		start.Format("2006-01-02"), end.Format("2006-01-02"), *count)

	for i := 1; i <= *count; i++ {
		csv, err := client.ExportServings(ctx, start, end)
		switch {
		case err == nil:
			fmt.Printf("  attempt %d: OK — %d bytes, %d rows of CSV\n", i, len(csv), strings.Count(csv, "\n"))
		case strings.Contains(err.Error(), "429"):
			fmt.Printf("  attempt %d: RATE LIMITED (429) — export quota exhausted for this account today\n", i)
		default:
			fmt.Printf("  attempt %d: ERROR — %v\n", i, err)
		}
		if i < *count {
			time.Sleep(2 * time.Second)
		}
	}
}

func mustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		log.Fatalf("invalid date %q (want YYYY-MM-DD): %v", s, err)
	}
	return t
}
