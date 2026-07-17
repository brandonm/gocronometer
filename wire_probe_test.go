package gocronometer

// Wire probe: captures raw GWT responses for offline diffing when Cronometer
// ships a new web build and the fixed-layout deserializers stop matching.
// Run manually (never in CI):
//
//	GOCRONOMETER_WIRE_PROBE=1 infisical run --env dev -- go test -run TestWireProbe -v .
//
// Credentials come from GOCRONOMETER_TEST_USERNAME / GOCRONOMETER_TEST_PASSWORD.
// Raw response bodies are written to $GOCRONOMETER_PROBE_OUT (default ./probe-out).

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWireProbe(t *testing.T) {
	if os.Getenv("GOCRONOMETER_WIRE_PROBE") != "1" {
		t.Skip("wire probe disabled; set GOCRONOMETER_WIRE_PROBE=1")
	}
	username := os.Getenv("GOCRONOMETER_TEST_USERNAME")
	password := os.Getenv("GOCRONOMETER_TEST_PASSWORD")
	if username == "" || password == "" {
		t.Skip("GOCRONOMETER_TEST_USERNAME / GOCRONOMETER_TEST_PASSWORD not set")
	}
	outDir := os.Getenv("GOCRONOMETER_PROBE_OUT")
	if outDir == "" {
		outDir = "probe-out"
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", outDir, err)
	}

	// Two configurations: the library's hardcoded GWT version values, and the
	// current web app's values captured live on 2026-07-17. If "old" husks and
	// "new" parses, the fix is a constants bump; if both husk, the Food wire
	// layout itself changed.
	configs := []struct {
		name string
		opts *ClientOptions
	}{
		{"old", nil},
		{"new", &ClientOptions{
			GWTHeader:      "8119D24F8CC7814B83B62DD87A7C62D8",
			GWTPermutation: "6B907FFD872F5DEE50BB42A45CEFDDDD",
		}},
	}
	for _, cfg := range configs {
		t.Run(cfg.name, func(t *testing.T) {
			runWireProbe(t, username, password, filepath.Join(outDir, cfg.name), cfg.opts)
		})
	}
}

func runWireProbe(t *testing.T, username, password, outDir string, opts *ClientOptions) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", outDir, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := NewClient(opts)
	if err := c.Login(ctx, username, password); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer c.Logout(ctx)
	t.Logf("login OK (userID=%s)", c.UserID)

	day := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	raw, err := c.GetDayInfoRaw(ctx, day)
	if err != nil {
		t.Fatalf("getDayInfo: %v", err)
	}
	probeDump(t, outDir, "dayinfo-2026-07-16.gwt", raw)

	servings, err := c.GetDayServings(ctx, day)
	if err != nil {
		t.Fatalf("parse day servings: %v", err)
	}
	t.Logf("servings on %s: %d", day.Format("2006-01-02"), len(servings))
	seen := map[int64]bool{}
	var ids []int64
	for _, s := range servings {
		t.Logf("  serving: %+v", s)
		if !seen[s.FoodID] {
			seen[s.FoodID] = true
			ids = append(ids, s.FoodID)
		}
	}
	if len(ids) > 3 {
		ids = ids[:3]
	}

	for _, id := range ids {
		body := probeRawGWT(ctx, t, c, fmt.Sprintf(GWTGetFood, c.Nonce, id))
		probeDump(t, outDir, fmt.Sprintf("getfood-%d.gwt", id), body)
		f, err := parseGetFoodResponse(body, id)
		if err != nil {
			t.Logf("  parseGetFoodResponse(%d): ERROR %v", id, err)
			continue
		}
		t.Logf("  parsed getFood %d: name=%q nutrients=%d ingredients=%d", id, f.Name, len(f.NutrientsPer100g), len(f.Ingredients))
	}

	if len(ids) > 0 {
		req := fmt.Sprintf(GWTGetAllFoodPrefix, c.Nonce) + strconv.Itoa(len(ids))
		for _, id := range ids {
			req += "|8|" + strconv.FormatInt(id, 10)
		}
		req += "|"
		body := probeRawGWT(ctx, t, c, req)
		probeDump(t, outDir, "getallfood.gwt", body)
		m, err := parseGetAllFoodsResponse(body, ids)
		if err != nil {
			t.Logf("  parseGetAllFoodsResponse: ERROR %v", err)
		} else {
			for id, f := range m {
				t.Logf("  parsed batch food %d: name=%q nutrients=%d", id, f.Name, len(f.NutrientsPer100g))
			}
		}
	}
}

func probeRawGWT(ctx context.Context, t *testing.T, c *Client, reqBody string) string {
	t.Helper()
	req, err := c.NewGWTRequestWithContext(ctx, "POST", GWTBaseURL, strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("build GWT request: %v", err)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("execute GWT request: %v", err)
	}
	defer closeAndExhaustReader(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read GWT response: %v", err)
	}
	t.Logf("raw GWT response: status=%d bytes=%d prefix=%q", resp.StatusCode, len(body), truncateForLog(string(body), 40))
	return string(body)
}

func probeDump(t *testing.T, outDir, name, body string) {
	t.Helper()
	path := filepath.Join(outDir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	t.Logf("wrote %s (%d bytes)", path, len(body))
}

func truncateForLog(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
