package gocronometer

import "testing"

// The first version of USDANutrientNames carried five wrong entries for five
// months. Three amino-acid codes were transposed or mislabelled, and two codes
// were assigned to nutrients they do not represent. Nothing caught it because
// the table was written from a USDA reference rather than checked against what
// Cronometer actually returns.
//
// Every expectation below was verified by matching a cached per-100g value
// against Cronometer's own food-detail panel for a known food. The comments
// record which observation pins which entry, so a future edit has to argue with
// evidence rather than with a table.
func TestUSDANutrientNames_VerifiedAgainstCronometer(t *testing.T) {
	tests := []struct {
		code     int
		wantName string
		wantUnit string
		evidence string
	}{
		// Food #458426 "Sashimi, Raw Fish": Cronometer's Lipids section shows
		// Omega-3 1.337 and Omega-6 0.057 per 100g, matching these codes exactly.
		{10001, "omega_3", "g", "sashimi omega-3 = 1.337"},
		{10002, "omega_6", "g", "sashimi omega-6 = 0.057"},
		// Same food, the omega-3/omega-6 subtrees.
		{629, "epa", "g", "sashimi EPA = 0.283"},
		{621, "dha", "g", "sashimi DHA = 0.890"},
		{675, "la_linoleic", "g", "sashimi LA = 0.017"},
		{853, "aa_arachidonic", "g", "sashimi AA = 0.040"},

		// 518 was labelled omega_3. It is Serine: the same food shows Serine
		// 0.952 under Protein while Omega-3 sits separately under Lipids.
		{518, "serine", "g", "sashimi serine = 0.952, distinct from omega-3 1.337"},
		// 502 and 507 were transposed.
		{502, "threonine", "g", "sashimi threonine = 1.023"},
		{507, "cystine", "g", "sashimi cystine = 0.250"},

		// 10005 was labelled sugar_alcohol. It is Iodine — a Fairlife shake
		// lists Iodine 100mcg per bottle and carries 10005 = 100, the only
		// nutrient in that food equal to 100. The real sugar alcohol is 10007.
		{10005, "iodine", "mcg", "Fairlife shake iodine = 100mcg/bottle"},
		{10007, "sugar_alcohol", "g", "sashimi sugar alcohol = 0.012"},

		// Vitamin A forms. RAE = retinol + beta_carotene/12 holds exactly:
		// romaine is 0 + 5226/12 = 435.5, butter is 744 + 168/12 = 758.
		{320, "vitamin_a", "mcg", "romaine RAE = 435.5"},
		{319, "retinol", "mcg", "romaine retinol = 0 while RAE > 0"},
		{321, "beta_carotene", "mcg", "romaine beta-carotene = 5226"},

		// Remaining amino acids, completing the 501-518 block.
		{511, "arginine", "g", "sashimi arginine = 1.396"},
		{513, "alanine", "g", "sashimi alanine = 1.411"},
		{514, "aspartic_acid", "g", "sashimi aspartic acid = 2.388"},
		{515, "glutamic_acid", "g", "sashimi glutamic acid = 3.482"},
		{516, "glycine", "g", "sashimi glycine = 1.120"},
		{517, "proline", "g", "sashimi proline = 0.825"},

		{207, "ash", "g", "sashimi ash = 1.18"},
		{10012, "oxalate", "mg", "sashimi oxalate = 0.90mg"},
	}

	for _, tt := range tests {
		got, ok := USDANutrientNames[tt.code]
		if !ok {
			t.Errorf("code %d missing — expected %q (%s)", tt.code, tt.wantName, tt.evidence)
			continue
		}
		if got.Name != tt.wantName {
			t.Errorf("code %d name = %q, want %q (%s)", tt.code, got.Name, tt.wantName, tt.evidence)
		}
		if got.Unit != tt.wantUnit {
			t.Errorf("code %d unit = %q, want %q", tt.code, got.Unit, tt.wantUnit)
		}
	}
}

// 318 is vitamin A in international units and 320 is the same nutrient in mcg
// RAE. Consumers sum by nutrient name, so admitting both would add IU to
// micrograms and produce a figure that is not a quantity of anything.
func TestUSDANutrientNames_ExcludesVitaminAInIU(t *testing.T) {
	if v, ok := USDANutrientNames[318]; ok {
		t.Errorf("code 318 (vitamin A, IU) must not be mapped — it collides with 320 (mcg RAE); got %+v", v)
	}
}

// No two codes may share a name. A duplicate silently sums two different
// measurements into one figure.
func TestUSDANutrientNames_NamesAreUnique(t *testing.T) {
	seen := map[string]int{}
	for code, info := range USDANutrientNames {
		if prev, dup := seen[info.Name]; dup {
			t.Errorf("name %q assigned to both code %d and code %d", info.Name, prev, code)
		}
		seen[info.Name] = code
	}
}

// The amino-acid block is contiguous in USDA numbering. A gap means a code was
// dropped; a name outside the expected set means one was mislabelled.
func TestUSDANutrientNames_AminoAcidBlockIsComplete(t *testing.T) {
	want := map[int]string{
		501: "tryptophan", 502: "threonine", 503: "isoleucine", 504: "leucine",
		505: "lysine", 506: "methionine", 507: "cystine", 508: "phenylalanine",
		509: "tyrosine", 510: "valine", 511: "arginine", 512: "histidine",
		513: "alanine", 514: "aspartic_acid", 515: "glutamic_acid",
		516: "glycine", 517: "proline", 518: "serine",
	}
	for code, name := range want {
		got, ok := USDANutrientNames[code]
		if !ok {
			t.Errorf("amino acid code %d (%s) is missing", code, name)
			continue
		}
		if got.Name != name {
			t.Errorf("code %d = %q, want %q", code, got.Name, name)
		}
		if got.Unit != "g" {
			t.Errorf("code %d unit = %q, want g", code, got.Unit)
		}
	}
}
