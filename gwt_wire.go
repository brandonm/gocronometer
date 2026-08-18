package gocronometer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

const (
	gwtBootstrapFile = "cronometer.nocache.js"
	maxBootstrapSize = 2 << 20
	maxCompiledSize  = 32 << 20
	maxPolicySize    = 4 << 20
)

var (
	gwtStrongNameRegexp = regexp.MustCompile(`['"]([A-F0-9]{32})['"]`)
	gwtAppPolicyRegexp  = regexp.MustCompile(`['"]app['"]\s*,\s*['"]([A-F0-9]{32})['"]`)
)

// knownGWTClassSignatures lists the serialization hashes whose layouts this
// library understands. A class hash changes when GWT's serialized fields
// change, making the public .gwt.rpc policy an authoritative schema canary.
var knownGWTClassSignatures = map[string][]string{
	"com.cronometer.shared.entries.models.Day":         {"782579793"},
	"com.cronometer.shared.entries.models.DayInfo":     {"416556043"},
	"com.cronometer.shared.entries.models.Serving":     {"2553599101"},
	"com.cronometer.shared.foods.models.Food":          {"2097636843"},
	"com.cronometer.shared.foods.models.FoodMeasures":  {"2106205728"},
	"com.cronometer.shared.foods.models.Ingredient":    {"1280520736"},
	"com.cronometer.shared.foods.models.Measure":       {"824760657", "1410168823"},
	"com.cronometer.shared.foods.models.Nutrient":      {"331784102"},
	"com.cronometer.shared.foods.models.NutrientMap":   {"168231382"},
	"com.cronometer.shared.foods.models.Translation":   {"4034452093"},
	"com.cronometer.shared.measurement.DerivedMeasure": {"338216045"},
}

// GWTClassSignatureChange describes a watched serialized class whose live
// signature is missing or is not one of the layouts supported by the library.
type GWTClassSignatureChange struct {
	Class string   `json:"class"`
	Known []string `json:"known"`
	Live  string   `json:"live"`
}

// GWTWireStatus describes Cronometer's currently deployed GWT build and its
// serialization policy. BuildChanged is informational when Compatible is true:
// Cronometer deployed new code, but the class layouts this library reads did
// not change.
type GWTWireStatus struct {
	Permutation       string                    `json:"permutation"`
	Policy            string                    `json:"policy"`
	BuildChanged      bool                      `json:"build_changed"`
	SchemaFingerprint string                    `json:"schema_fingerprint"`
	ClassSignatures   map[string]string         `json:"class_signatures"`
	ClassChanges      []GWTClassSignatureChange `json:"class_changes"`
}

// Compatible reports whether every watched serialized class has a known hash.
func (s GWTWireStatus) Compatible() bool {
	return len(s.ClassChanges) == 0
}

// GWTWireIncompatibleError is returned before authentication when Cronometer's
// public serialization policy contains an unsupported watched class layout.
type GWTWireIncompatibleError struct {
	Status GWTWireStatus
}

func (e *GWTWireIncompatibleError) Error() string {
	classes := make([]string, 0, len(e.Status.ClassChanges))
	for _, change := range e.Status.ClassChanges {
		live := change.Live
		if live == "" {
			live = "missing"
		}
		classes = append(classes, fmt.Sprintf("%s=%s", change.Class, live))
	}
	return "Cronometer GWT serialization policy changed: " + strings.Join(classes, ", ")
}

// RefreshGWTWireStatus discovers Cronometer's live build and serialization
// policy from its public GWT assets. Compatible live request values replace the
// compiled defaults for subsequent calls. An incompatible watched class fails
// loudly before any diary data is collected.
func (c *Client) RefreshGWTWireStatus(ctx context.Context) (GWTWireStatus, error) {
	permutation, policy, signatures, err := c.discoverGWTWireMetadata(ctx)
	if err != nil {
		return GWTWireStatus{}, fmt.Errorf("discover Cronometer GWT metadata: %w", err)
	}

	status := GWTWireStatus{
		Permutation:     permutation,
		Policy:          policy,
		BuildChanged:    permutation != GWTPermutation || policy != GWTHeader,
		ClassSignatures: watchedGWTClassSignatures(signatures),
		ClassChanges:    compareGWTClassSignatures(signatures),
	}
	status.SchemaFingerprint = fingerprintGWTClassSignatures(status.ClassSignatures)
	c.setGWTWireStatus(status)
	if c.onGWTWireStatus != nil {
		c.onGWTWireStatus(cloneGWTWireStatus(status))
	}
	if !status.Compatible() {
		return status, &GWTWireIncompatibleError{Status: cloneGWTWireStatus(status)}
	}

	// These values identify the selected compiled permutation and the
	// CronometerService serialization policy. Use the live pair rather than
	// requiring a library release for every compatible Cronometer deployment.
	c.GWTPermutation = permutation
	c.GWTHeader = policy
	return status, nil
}

// GWTWireStatus returns the most recently discovered status. It is empty until
// RefreshGWTWireStatus or Login performs discovery.
func (c *Client) GWTWireStatus() GWTWireStatus {
	c.wireMu.RLock()
	defer c.wireMu.RUnlock()
	return cloneGWTWireStatus(c.wireStatus)
}

func (c *Client) setGWTWireStatus(status GWTWireStatus) {
	c.wireMu.Lock()
	c.wireStatus = cloneGWTWireStatus(status)
	c.wireMu.Unlock()
}

func cloneGWTWireStatus(status GWTWireStatus) GWTWireStatus {
	clone := status
	clone.ClassSignatures = make(map[string]string, len(status.ClassSignatures))
	for class, signature := range status.ClassSignatures {
		clone.ClassSignatures[class] = signature
	}
	clone.ClassChanges = make([]GWTClassSignatureChange, len(status.ClassChanges))
	for i, change := range status.ClassChanges {
		clone.ClassChanges[i] = change
		clone.ClassChanges[i].Known = append([]string(nil), change.Known...)
	}
	return clone
}

func (c *Client) discoverGWTWireMetadata(ctx context.Context) (string, string, map[string]string, error) {
	base := strings.TrimRight(c.GWTModuleBase, "/") + "/"
	bootstrap, err := c.fetchGWTAsset(ctx, base+gwtBootstrapFile, maxBootstrapSize)
	if err != nil {
		return "", "", nil, fmt.Errorf("bootstrap: %w", err)
	}
	permutation, err := extractSingleGWTStrongName(bootstrap)
	if err != nil {
		return "", "", nil, fmt.Errorf("bootstrap permutation: %w", err)
	}

	compiled, err := c.fetchGWTAsset(ctx, base+permutation+".cache.js", maxCompiledSize)
	if err != nil {
		return "", "", nil, fmt.Errorf("compiled permutation: %w", err)
	}
	policy, err := extractGWTAppPolicy(compiled)
	if err != nil {
		return "", "", nil, fmt.Errorf("CronometerService policy: %w", err)
	}

	policyBody, err := c.fetchGWTAsset(ctx, base+policy+".gwt.rpc", maxPolicySize)
	if err != nil {
		return "", "", nil, fmt.Errorf("serialization policy: %w", err)
	}
	signatures, err := parseGWTSerializationPolicy(policyBody)
	if err != nil {
		return "", "", nil, err
	}
	return permutation, policy, signatures, nil
}

func (c *Client) fetchGWTAsset(ctx context.Context, url string, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeAndExhaustReader(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s returned status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("GET %s exceeded %d bytes", url, maxBytes)
	}
	return body, nil
}

func extractSingleGWTStrongName(body []byte) (string, error) {
	matches := gwtStrongNameRegexp.FindAllSubmatch(body, -1)
	unique := make(map[string]bool)
	for _, match := range matches {
		unique[string(match[1])] = true
	}
	if len(unique) != 1 {
		return "", fmt.Errorf("expected one permutation strong name, found %d", len(unique))
	}
	for value := range unique {
		return value, nil
	}
	return "", fmt.Errorf("permutation strong name not found")
}

func extractGWTAppPolicy(body []byte) (string, error) {
	matches := gwtAppPolicyRegexp.FindAllSubmatch(body, -1)
	unique := make(map[string]bool)
	for _, match := range matches {
		unique[string(match[1])] = true
	}
	if len(unique) != 1 {
		return "", fmt.Errorf("expected one app policy strong name, found %d", len(unique))
	}
	for value := range unique {
		return value, nil
	}
	return "", fmt.Errorf("app policy strong name not found")
}

func parseGWTSerializationPolicy(body []byte) (map[string]string, error) {
	signatures := make(map[string]string)
	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Split(line, ",")
		if len(parts) < 7 {
			continue
		}
		class := strings.TrimSpace(parts[0])
		descriptor := strings.TrimSpace(parts[len(parts)-2])
		signature := strings.TrimSpace(parts[len(parts)-1])
		if class == "" || descriptor != class+"/"+signature {
			continue
		}
		signatures[class] = signature
	}
	if len(signatures) == 0 {
		return nil, fmt.Errorf("serialization policy contained no class signatures")
	}
	return signatures, nil
}

func compareGWTClassSignatures(live map[string]string) []GWTClassSignatureChange {
	classes := make([]string, 0, len(knownGWTClassSignatures))
	for class := range knownGWTClassSignatures {
		classes = append(classes, class)
	}
	sort.Strings(classes)

	var changes []GWTClassSignatureChange
	for _, class := range classes {
		known := knownGWTClassSignatures[class]
		actual := live[class]
		matched := false
		for _, expected := range known {
			if actual == expected {
				matched = true
				break
			}
		}
		if !matched {
			changes = append(changes, GWTClassSignatureChange{
				Class: class,
				Known: append([]string(nil), known...),
				Live:  actual,
			})
		}
	}
	return changes
}

func watchedGWTClassSignatures(live map[string]string) map[string]string {
	watched := make(map[string]string, len(knownGWTClassSignatures))
	for class := range knownGWTClassSignatures {
		watched[class] = live[class]
	}
	return watched
}

func fingerprintGWTClassSignatures(signatures map[string]string) string {
	classes := make([]string, 0, len(signatures))
	for class := range signatures {
		classes = append(classes, class)
	}
	sort.Strings(classes)
	h := sha256.New()
	for _, class := range classes {
		fmt.Fprintf(h, "%s=%s\n", class, signatures[class])
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
