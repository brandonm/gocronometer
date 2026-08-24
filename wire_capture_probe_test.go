package gocronometer

// Live capture + token dump for GWT schema breaks.
//
// When the schema canary fires, this is the first thing to run: it discovers
// whatever build Cronometer is currently serving, logs in against it despite
// the incompatibility, and writes raw responses plus a decoded token dump that
// the field layout can be read straight off.
//
// Capture (needs credentials):
//
//	GOCRONOMETER_WIRE_PROBE=1 infisical run --env=dev -- sh -c \
//	  'GOCRONOMETER_TEST_USERNAME=$CRONOMETER_EMAIL GOCRONOMETER_TEST_PASSWORD=$CRONOMETER_PASSWORD \
//	   go test -run TestWireCaptureProbe -v .'
//
// Dump a capture (no credentials needed):
//
//	GOCRONOMETER_DUMP=1 DUMP_FILE=probe-out/getallfood.txt go test -run TestWireTokenDump -v .
//
// Neither test runs in CI: both are gated behind environment variables.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWireCaptureProbe(t *testing.T) {
	if os.Getenv("GOCRONOMETER_WIRE_PROBE") != "1" {
		t.Skip("wire probe disabled; set GOCRONOMETER_WIRE_PROBE=1")
	}
	username, password := os.Getenv("GOCRONOMETER_TEST_USERNAME"), os.Getenv("GOCRONOMETER_TEST_PASSWORD")
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Discover the live build first. An incompatible policy is the reason this
	// probe is being run, so adopt the discovered values and keep going rather
	// than letting the gate abort the capture.
	discovery := NewClient(&ClientOptions{DisableGWTWireCheck: true})
	status, err := discovery.RefreshGWTWireStatus(ctx)
	if err != nil && status.Permutation == "" {
		t.Fatalf("discover live GWT metadata: %v", err)
	}
	t.Logf("live permutation=%s policy=%s compatible=%v", status.Permutation, status.Policy, status.Compatible())
	for _, c := range status.ClassChanges {
		live := c.Live
		if live == "" {
			live = "missing"
		}
		t.Logf("  CHANGED %s: library knows %v, live is %s", c.Class, c.Known, live)
	}

	c := NewClient(&ClientOptions{
		DisableGWTWireCheck: true,
		GWTPermutation:      status.Permutation,
		GWTHeader:           status.Policy,
	})
	if err := c.Login(ctx, username, password); err != nil {
		t.Fatalf("login: %v", err)
	}
	defer c.Logout(ctx)

	// Walk back to the most recent day that actually has diary entries.
	var servings []DayServing
	var day time.Time
	for i := 0; i < 14; i++ {
		d := time.Now().AddDate(0, 0, -i).Truncate(24 * time.Hour)
		s, err := c.GetDayServings(ctx, d)
		if err != nil {
			t.Logf("day %s: %v", d.Format("2006-01-02"), err)
			continue
		}
		if len(s) > 0 {
			servings, day = s, d
			break
		}
	}
	if len(servings) == 0 {
		t.Fatal("no servings found in the last 14 days")
	}
	t.Logf("capturing from %s (%d servings)", day.Format("2006-01-02"), len(servings))

	ids := make([]int64, 0, len(servings))
	seen := make(map[int64]bool, len(servings))
	for _, s := range servings {
		if !seen[s.FoodID] {
			seen[s.FoodID] = true
			ids = append(ids, s.FoodID)
		}
	}
	t.Logf("food IDs: %v", ids)

	if raw, err := c.GetDayInfoRaw(ctx, day); err != nil {
		t.Errorf("getDayInfo: %v", err)
	} else {
		writeCapture(t, filepath.Join(outDir, "dayinfo.txt"), raw)
	}

	batch := c.formatGWTRequest(GWTGetAllFoodPrefix, c.Nonce) + strconv.Itoa(len(ids))
	for _, id := range ids {
		batch += "|8|" + strconv.FormatInt(id, 10)
	}
	batch += "|"
	writeCapture(t, filepath.Join(outDir, "getallfood.txt"), postRaw(t, ctx, c, batch))
	writeCapture(t, filepath.Join(outDir, "getfood.txt"),
		postRaw(t, ctx, c, c.formatGWTRequest(GWTGetFood, c.Nonce, ids[0])))

	t.Logf("captures written to %s/ — dump one with TestWireTokenDump", outDir)
	t.Logf("scrub the account user ID (%s) before committing anything as a fixture", c.UserID)
}

// TestWireTokenDump prints a capture's string table and every token in read
// order, annotating tokens that resolve to a string-table entry. Walking a
// class is then a matter of finding its type signature and counting fields
// until the next one.
func TestWireTokenDump(t *testing.T) {
	if os.Getenv("GOCRONOMETER_DUMP") != "1" {
		t.Skip("token dump disabled; set GOCRONOMETER_DUMP=1")
	}
	path := os.Getenv("DUMP_FILE")
	if path == "" {
		t.Skip("DUMP_FILE not set")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	r, err := NewGWTReader(string(body))
	if err != nil {
		t.Fatalf("GWT reader: %v", err)
	}

	fmt.Printf("version=%d flags=%d tokens=%d strings=%d\n", r.version, r.flags, len(r.tokens), len(r.stringTable))
	fmt.Println("--- string table ---")
	for i, s := range r.stringTable {
		fmt.Printf("  [%d] %s\n", i+1, s)
	}
	fmt.Println("--- tokens in read order ---")
	for n := 0; r.index > 0; n++ {
		r.index--
		tok := r.tokens[r.index]
		note := ""
		if v, err := strconv.Atoi(tok); err == nil && v > 0 && v <= len(r.stringTable) {
			note = "  => " + r.stringTable[v-1]
		}
		fmt.Printf("%5d  %-24s%s\n", n, tok, note)
	}
}

func postRaw(t *testing.T, ctx context.Context, c *Client, body string) string {
	t.Helper()
	req, err := c.NewGWTRequestWithContext(ctx, "POST", GWTBaseURL, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("execute request: %v", err)
	}
	defer closeAndExhaustReader(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func writeCapture(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	t.Logf("wrote %s (%d bytes)", path, len(body))
}
