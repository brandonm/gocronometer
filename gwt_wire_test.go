package gocronometer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
)

const (
	testPermutation = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	testPolicy      = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
)

func TestRefreshGWTWireStatus_CompatibleBuild(t *testing.T) {
	server := newGWTMetadataServer(t, nil)
	defer server.Close()

	var callback GWTWireStatus
	c := NewClient(&ClientOptions{
		GWTModuleBase: server.URL + "/",
		OnGWTWireStatus: func(status GWTWireStatus) {
			callback = status
		},
	})
	status, err := c.RefreshGWTWireStatus(context.Background())
	if err != nil {
		t.Fatalf("RefreshGWTWireStatus: %v", err)
	}
	if !status.Compatible() {
		t.Fatalf("status should be compatible: %+v", status.ClassChanges)
	}
	if !status.BuildChanged {
		t.Fatal("new permutation/policy should mark BuildChanged")
	}
	if status.SchemaFingerprint == "" {
		t.Fatal("schema fingerprint is empty")
	}
	if c.GWTPermutation != testPermutation || c.GWTHeader != testPolicy {
		t.Fatalf("client did not adopt live values: permutation=%q policy=%q", c.GWTPermutation, c.GWTHeader)
	}
	if callback.SchemaFingerprint != status.SchemaFingerprint {
		t.Fatal("status callback did not receive discovered status")
	}
	if got := c.GWTWireStatus(); got.SchemaFingerprint != status.SchemaFingerprint {
		t.Fatal("client did not retain discovered status")
	}
}

func TestRefreshGWTWireStatus_IncompatibleClass(t *testing.T) {
	server := newGWTMetadataServer(t, map[string]string{
		"com.cronometer.shared.entries.models.Serving": "999999999",
	})
	defer server.Close()

	c := NewClient(&ClientOptions{GWTModuleBase: server.URL + "/"})
	status, err := c.RefreshGWTWireStatus(context.Background())
	var incompatible *GWTWireIncompatibleError
	if !errors.As(err, &incompatible) {
		t.Fatalf("error = %v, want GWTWireIncompatibleError", err)
	}
	if status.Compatible() || len(status.ClassChanges) != 1 {
		t.Fatalf("unexpected class changes: %+v", status.ClassChanges)
	}
	change := status.ClassChanges[0]
	if change.Class != "com.cronometer.shared.entries.models.Serving" || change.Live != "999999999" {
		t.Fatalf("unexpected change: %+v", change)
	}
	// Incompatible metadata is observable but must not be used for requests.
	if c.GWTHeader != GWTHeader || c.GWTPermutation != GWTPermutation {
		t.Fatal("client adopted incompatible wire metadata")
	}
}

func TestFormatGWTRequest_UsesClientPolicy(t *testing.T) {
	c := NewClient(&ClientOptions{GWTHeader: testPolicy, DisableGWTWireCheck: true})
	body := c.formatGWTRequest(GWTGetFood, "nonce", int64(123))
	if !strings.Contains(body, "|"+testPolicy+"|") {
		t.Fatalf("request does not contain client policy: %s", body)
	}
	if strings.Contains(body, "|"+GWTHeader+"|") {
		t.Fatal("request still contains compiled policy")
	}
}

func TestExtractGWTMetadata(t *testing.T) {
	permutation, err := extractSingleGWTStrongName([]byte(`var p='AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA';`))
	if err != nil || permutation != testPermutation {
		t.Fatalf("permutation = %q, %v", permutation, err)
	}
	policy, err := extractGWTAppPolicy([]byte(`call(x,'app','BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB',y)`))
	if err != nil || policy != testPolicy {
		t.Fatalf("policy = %q, %v", policy, err)
	}
	parsed, err := parseGWTSerializationPolicy([]byte("com.example.Food, true, true, true, true, com.example.Food/123, 123\n"))
	if err != nil || parsed["com.example.Food"] != "123" {
		t.Fatalf("policy parse = %v, %v", parsed, err)
	}
}

func newGWTMetadataServer(t *testing.T, overrides map[string]string) *httptest.Server {
	t.Helper()
	policy := testSerializationPolicy(overrides)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + gwtBootstrapFile:
			fmt.Fprintf(w, "var permutation='%s';", testPermutation)
		case "/" + testPermutation + ".cache.js":
			fmt.Fprintf(w, "proxy.call(this,service(),'app','%s',serializer);", testPolicy)
		case "/" + testPolicy + ".gwt.rpc":
			fmt.Fprint(w, policy)
		default:
			http.NotFound(w, r)
		}
	}))
}

func testSerializationPolicy(overrides map[string]string) string {
	classes := make([]string, 0, len(knownGWTClassSignatures))
	for class := range knownGWTClassSignatures {
		classes = append(classes, class)
	}
	sort.Strings(classes)
	var b strings.Builder
	for _, class := range classes {
		signature := knownGWTClassSignatures[class][len(knownGWTClassSignatures[class])-1]
		if override := overrides[class]; override != "" {
			signature = override
		}
		fmt.Fprintf(&b, "%s, true, true, true, true, %s/%s, %s\n", class, class, signature, signature)
	}
	return b.String()
}
