package gocronometer

// Probe: dump the raw findMyFoods GWT response to inspect whether it carries a
// per-food last-modified timestamp we could use for change detection (the fork
// currently extracts only ID + Name). Run manually, never in CI:
//
//	GOCRONOMETER_WIRE_PROBE=1 infisical run --env dev -- sh -c \
//	  'GOCRONOMETER_TEST_USERNAME=$CRONOMETER_EMAIL GOCRONOMETER_TEST_PASSWORD=$CRONOMETER_PASSWORD \
//	   go test -run TestFindMyFoodsProbe -v .'

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFindMyFoodsProbe(t *testing.T) {
	if os.Getenv("GOCRONOMETER_WIRE_PROBE") != "1" {
		t.Skip("wire probe disabled; set GOCRONOMETER_WIRE_PROBE=1")
	}
	username := os.Getenv("GOCRONOMETER_TEST_USERNAME")
	password := os.Getenv("GOCRONOMETER_TEST_PASSWORD")
	if username == "" || password == "" {
		t.Skip("GOCRONOMETER_TEST_USERNAME / GOCRONOMETER_TEST_PASSWORD not set")
	}
	out := os.Getenv("GOCRONOMETER_PROBE_OUT")
	if out == "" {
		out = "probe-out"
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := NewClient(nil)
	if err := c.Login(ctx, username, password); err != nil {
		t.Fatalf("login: %v", err)
	}
	defer c.Logout(ctx)

	// Raw findMyFoods RPC (the same call FindMyFoods makes, but we keep the body).
	reqBody := fmt.Sprintf(GWTFindMyFoods, c.Nonce, c.UserID)
	req, err := c.NewGWTRequestWithContext(ctx, "POST", GWTBaseURL, strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer closeAndExhaustReader(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	path := out + "/findmyfoods.gwt"
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Logf("wrote %s (%d bytes)", path, len(body))

	// Also parse via the library to confirm the count, so the dump can be
	// cross-referenced against known food IDs.
	foods, err := c.FindMyFoods(ctx)
	if err != nil {
		t.Fatalf("FindMyFoods: %v", err)
	}
	t.Logf("FindMyFoods parsed %d custom foods", len(foods))
	for i, f := range foods {
		if i >= 5 {
			t.Logf("  ... (%d more)", len(foods)-5)
			break
		}
		t.Logf("  id=%d name=%q", f.ID, f.Name)
	}
}
