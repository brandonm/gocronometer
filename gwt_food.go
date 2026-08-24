package gocronometer

import (
	"fmt"
	"strings"
)

// GWTFood represents a deserialized Food object from the GWT response.
type GWTFood struct {
	ID           int
	FoodType     int // FoodType enum ordinal
	Ingredients  []GWTIngredient
	Nutrients    map[int]float64 // USDA nutrient code → value per 100g
	Name         string
	Description  string
	Measures     []GWTMeasure
	Source       string            // e.g. "NCCDB:13930", "CRDB", "Custom"
	Translations map[string]string // language code → translated name
	UserID       int

	// fallbackName is the first translated name seen in stream order, used
	// when the preferred languages are unavailable (e.g. a back-referenced
	// Language object whose code could not be recovered).
	fallbackName string
}

// GWTIngredient represents a recipe ingredient.
type GWTIngredient struct {
	FoodID    int
	MeasureID int
	Amount    float64 // weight in grams
}

// GWTMeasure represents a serving size measure.
type GWTMeasure struct {
	ID     int
	FoodID int
	Name   string
	Grams  float64 // grams per unit of this measure
	// MilliL is the measure's volume in milliliters. Only present in the
	// post-2026-07-16 Measure layout, and only for volume measures ("cup",
	// "Gallon", ...); 0 otherwise.
	MilliL float64
}

// Measure type descriptors pin the known serialization vintages. The trailing
// hash changes whenever Cronometer recompiles the class with different fields,
// so an unknown hash means an unknown field layout — deserializeMeasure treats
// it as a hard error rather than walking the stream blind (the silent-husk
// failure mode of the 2026-07-16 incident).
//
// DerivedMeasure is a Measure subclass whose serialized form is byte-identical
// to its parent's: in the compiled GWT permutation the generated deserialize
// functions for the two classes are literally the same code. A GWT class hash
// also covers the supertype chain, so every Measure field change moves the
// DerivedMeasure hash too — the pair always advances together.
const (
	// measureTypeV1 is the layout Cronometer served until 2026-07-16.
	measureTypeV1 = "com.cronometer.shared.foods.models.Measure/824760657"

	// measureTypeV2 is the layout served from 2026-07-16: V1 plus a nullable
	// java.lang.Double volume-in-millilitres field.
	measureTypeV2        = "com.cronometer.shared.foods.models.Measure/1410168823"
	derivedMeasureTypeV2 = "com.cronometer.shared.measurement.DerivedMeasure/338216045"

	// measureTypeV3 is the layout served from 2026-08-24: V2 plus a Map of
	// localized measure-name translations, inserted between the name string
	// and the Measure$Type enum.
	measureTypeV3        = "com.cronometer.shared.foods.models.Measure/1979099908"
	derivedMeasureTypeV3 = "com.cronometer.shared.measurement.DerivedMeasure/4214796590"
)

// DeserializeFood reads a Food object from the GWT stream.
// Must be called after ReadObject() returns the Food type signature.
//
// Food has 20 fields (verified by tracing PKi/RKi in compiled GWT JS):
//
//	 1 (b)  int     — unknown (always 0)
//	 2 (c)  double  — unknown (always 0)
//	 3 (d)  object  — barcode/UPC ArrayList
//	 4 (e)  int     — unknown
//	 5 (f)  string  — description
//	 6 (g)  int     — unknown
//	 7 (i)  int     — unknown
//	 8 (j)  int     — Food ID
//	 9 (k)  object  — Ingredients (ArrayList of Ingredient)
//	10 (n)  object  — NutritionLabelType enum
//	11 (o)  long    — timestamp
//	12 (p)  object  — FoodMeasures (wraps ArrayList of Measure)
//	13 (q)  object  — NutrientMap (per 100g)
//	14 (s)  object  — properties HashMap<String,String>
//	15 (t)  double  — unknown (always 0)
//	16 (u)  string  — source/database ("Custom", "CRDB", "NCCDB:...")
//	17 (v)  object  — tags HashSet
//	18 (A)  object  — Translations (ArrayList of Translation)
//	19 (B)  object  — FoodType enum
//	20 (C)  int     — user ID
func DeserializeFood(r *GWTReader) (*GWTFood, error) {
	f := &GWTFood{
		Nutrients:    make(map[int]float64),
		Translations: make(map[string]string),
	}

	// Field 1 (b): int — unknown
	r.ReadInt()
	// Field 2 (c): double — unknown
	r.ReadDouble()
	// Field 3 (d): object — barcode ArrayList (skip)
	skipArrayList(r)
	// Field 4 (e): int — unknown
	r.ReadInt()
	// Field 5 (f): string — description
	f.Description = r.ReadString()
	// Field 6 (g): int — unknown
	r.ReadInt()
	// Field 7 (i): int — unknown
	r.ReadInt()
	// Field 8 (j): int — Food ID
	f.ID = r.ReadInt()
	// Field 9 (k): object — Ingredients ArrayList
	if err := deserializeIngredientList(r, f); err != nil {
		return f, fmt.Errorf("ingredients: %w", err)
	}
	// Field 10 (n): object — NutritionLabelType enum
	skipEnum(r)
	// Field 11 (o): long — timestamp
	r.ReadLong()
	// Field 12 (p): object — FoodMeasures
	if err := deserializeFoodMeasures(r, f); err != nil {
		return f, fmt.Errorf("measures: %w", err)
	}
	// Field 13 (q): object — NutrientMap
	if err := deserializeNutrientMap(r, f); err != nil {
		return f, fmt.Errorf("nutrients: %w", err)
	}
	// Field 14 (s): object — properties HashMap<String,String> (skip)
	skipStringHashMap(r)
	// Field 15 (t): double — unknown
	r.ReadDouble()
	// Field 16 (u): string — source
	f.Source = r.ReadString()
	// Field 17 (v): object — tags HashSet (skip)
	skipHashSet(r)
	// Field 18 (A): object — Translations ArrayList
	deserializeTranslationList(r, f)
	// Field 19 (B): object — FoodType enum
	f.FoodType = readEnumOrdinal(r)
	// Field 20 (C): int — user ID
	f.UserID = r.ReadInt()

	// Set display name from translations (prefer English)
	if en, ok := f.Translations["en"]; ok && en != "" {
		f.Name = en
	} else if de, ok := f.Translations["de"]; ok && de != "" {
		f.Name = de
	} else {
		f.Name = f.fallbackName
	}

	// A zero food ID means the fixed-field walk drifted off the stream (every
	// real food carries its ID at field 8). Surface it instead of returning a
	// husk the caller can't distinguish from data.
	if f.ID == 0 {
		return f, fmt.Errorf("food stream misaligned: deserialized food ID is 0")
	}

	return f, nil
}

// deserializeIngredientList reads an ArrayList of Ingredient objects.
func deserializeIngredientList(r *GWTReader, f *GWTFood) error {
	typ := r.ReadObject()
	if typ == "" || isBackRef(typ) {
		return nil
	}
	if !strings.Contains(typ, "ArrayList") {
		return fmt.Errorf("expected ArrayList, got %q", typ)
	}
	count := r.ReadInt()
	for i := 0; i < count; i++ {
		ing, err := deserializeIngredient(r)
		if err != nil {
			return fmt.Errorf("ingredient %d: %w", i, err)
		}
		f.Ingredients = append(f.Ingredients, ing)
	}
	return nil
}

// deserializeIngredient reads a single Ingredient from the stream.
// Ingredient has 6 fields: double(amount), int(foodID), long(hash), int(measureID), int(?), int(?)
func deserializeIngredient(r *GWTReader) (GWTIngredient, error) {
	typeSig := r.ReadObject()
	if typeSig == "" {
		return GWTIngredient{}, fmt.Errorf("null ingredient")
	}

	amount := r.ReadDouble() // grams
	foodID := r.ReadInt()
	r.ReadLong() // hash (ignored)
	measureID := r.ReadInt()
	r.ReadInt() // unknown
	r.ReadInt() // unknown

	return GWTIngredient{
		FoodID:    foodID,
		MeasureID: measureID,
		Amount:    amount,
	}, nil
}

// deserializeFoodMeasures reads a FoodMeasures object containing an ArrayList of Measure.
func deserializeFoodMeasures(r *GWTReader, f *GWTFood) error {
	typ := r.ReadObject()
	if typ == "" || isBackRef(typ) {
		return nil
	}
	// FoodMeasures: int(defaultMeasureID) + ReadObject(ArrayList of Measure)
	r.ReadInt() // default measure ID

	measListType := r.ReadObject()
	if measListType == "" || isBackRef(measListType) {
		return nil
	}
	count := r.ReadInt()
	for i := 0; i < count; i++ {
		m, err := deserializeMeasure(r)
		if err != nil {
			return fmt.Errorf("measure %d: %w", i, err)
		}
		if m.ID != 0 {
			f.Measures = append(f.Measures, m)
		}
	}
	return nil
}

// deserializeMeasure reads a single Measure from the stream, dispatching on
// the type descriptor's serialization hash. An unrecognized hash is a hard
// error: it means Cronometer recompiled the class with a different field
// layout and any fixed walk would silently misalign the rest of the stream.
func deserializeMeasure(r *GWTReader) (GWTMeasure, error) {
	typeSig := r.ReadObject()
	if typeSig == "" || isBackRef(typeSig) {
		return GWTMeasure{}, nil
	}

	switch typeSig {
	case measureTypeV1:
		return deserializeMeasureV1(r), nil
	case measureTypeV2, derivedMeasureTypeV2:
		return deserializeMeasureFields(r, false)
	case measureTypeV3, derivedMeasureTypeV3:
		return deserializeMeasureFields(r, true)
	}
	return GWTMeasure{}, fmt.Errorf("unknown Measure vintage %q — Cronometer layout change, re-capture the wire format", typeSig)
}

// deserializeMeasureV1 reads the pre-2026-07-16 Measure layout.
// Measure has 8 fields: double(?), double(grams), int(foodID), int(measureID),
// object(Measure$Type enum), string(name), object(Measure$Type), double(weight)
func deserializeMeasureV1(r *GWTReader) GWTMeasure {
	r.ReadDouble()           // field 1: unknown
	r.ReadDouble()           // field 2: grams per unit (but often 0?)
	foodID := r.ReadInt()    // field 3: food ID
	measureID := r.ReadInt() // field 4: measure ID
	skipEnum(r)              // field 5: Measure$Type enum
	name := r.ReadString()   // field 6: name ("g", "full recipe", etc.)
	skipEnumOrBackRef(r)     // field 7: Measure$Type (usually backref)
	grams := r.ReadDouble()  // field 8: weight in grams

	return GWTMeasure{
		ID:     measureID,
		FoodID: foodID,
		Name:   name,
		Grams:  grams,
	}
}

// deserializeMeasureFields reads a Measure body. The V2 and V3 layouts differ
// by exactly one field, so both share this walk; withTranslations selects V3.
// Field order was recovered from the compiled GWT permutation's generated
// deserializer and confirmed against live captures in testdata/ — a "Gallon"
// DerivedMeasure carries Double 3785.411784 (ml per US gallon) while a "g"
// Measure carries a null ml.
//
//	 1  double  — unknown (always 1.0 in captures)
//	 2  boolean — unknown (0 or 1)
//	 3  int     — food ID
//	 4  boolean — unknown (0 or 1)
//	 5  int     — measure ID
//	 6  object  — java.lang.Double: volume in ml (null for non-volume measures)
//	 7  string  — name ("cup", "Gallon", "g", "full recipe", …)
//	 8  object  — V3 ONLY: Map of localized measure-name translations
//	 9  object  — Measure$Type enum (type+ordinal, or backref)
//	10  double  — weight in grams
//
// Fields 2 and 4 are booleans in the compiled serializer (GWT writes them as
// the same 0/1 token an int uses, so reading them as ints stays aligned).
func deserializeMeasureFields(r *GWTReader, withTranslations bool) (GWTMeasure, error) {
	r.ReadDouble()           // field 1: unknown
	r.ReadInt()              // field 2: unknown boolean
	foodID := r.ReadInt()    // field 3: food ID
	r.ReadInt()              // field 4: unknown boolean
	measureID := r.ReadInt() // field 5: measure ID

	// Field 6: nullable java.lang.Double — volume in millilitres.
	var ml float64
	mlType := r.ReadObject()
	if mlType != "" && !isBackRef(mlType) {
		if !strings.Contains(mlType, "Double") {
			return GWTMeasure{}, fmt.Errorf("measure stream misaligned: expected java.lang.Double at ml field, got %q", mlType)
		}
		ml = r.ReadDouble()
	}

	name := r.ReadString() // field 7: name

	if withTranslations {
		// Field 8: Map<?, MeasureTranslation> of localized measure names.
		if err := skipMeasureTranslations(r); err != nil {
			return GWTMeasure{}, err
		}
	}

	skipEnumOrBackRef(r)    // field 9: Measure$Type enum
	grams := r.ReadDouble() // field 10: weight in grams

	return GWTMeasure{
		ID:     measureID,
		FoodID: foodID,
		Name:   name,
		Grams:  grams,
		MilliL: ml,
	}, nil
}

// skipMeasureTranslations consumes the V3 measure-name translation map.
//
// Every measure observed on 2026-08-24 carried an empty map, so the entry
// layout has never been seen on the wire. Rather than guess at it and risk the
// silent misalignment that produced the 2026-07-16 husks, a non-empty map is a
// hard, actionable error: re-capture the wire format and decode it for real.
//
// For whoever gets that error: the value type is almost certainly
// MeasureTranslation/3000345244, whose compiled deserializer reads four fields
// — object (a Language), int, long, string.
func skipMeasureTranslations(r *GWTReader) error {
	typeSig := r.ReadObject()
	if typeSig == "" || isBackRef(typeSig) {
		return nil
	}
	if !strings.Contains(typeSig, "Map") {
		return fmt.Errorf("measure stream misaligned: expected a Map at the translations field, got %q", typeSig)
	}
	if n := r.ReadInt(); n != 0 {
		return fmt.Errorf("measure translations map has %d entries — this layout has only ever been observed empty; re-capture the wire format and decode it before trusting the stream", n)
	}
	return nil
}

// deserializeNutrientMap reads a NutrientMap from the stream.
// Structure: NutrientFilter enum + HashMap<Integer(code), Nutrient(value, code, Nutrient$Type)>
func deserializeNutrientMap(r *GWTReader, f *GWTFood) error {
	typ := r.ReadObject()
	if typ == "" || isBackRef(typ) {
		return nil
	}

	// NutrientFilter enum
	skipEnum(r)

	// HashMap
	mapType := r.ReadObject()
	if mapType == "" || isBackRef(mapType) {
		return nil
	}

	count := r.ReadInt()
	for i := 0; i < count; i++ {
		// Save position before reading the key so we can restore if we've
		// overrun into non-nutrient data (happens when count > actual entries).
		saved := r.SavePosition()

		// Key: Integer object wrapping the nutrient code
		keyType := r.ReadObject()
		if keyType == "" {
			continue
		}
		// If the key type isn't Integer or a backref to Integer, we've overrun
		// the nutrient data into subsequent Food fields. Restore position and stop.
		if !isBackRef(keyType) && !strings.Contains(keyType, "Integer") {
			r.RestorePosition(saved)
			break
		}
		code := r.ReadInt() // Integer value = nutrient code

		// Value: Nutrient object with 3 fields: double(value), int(code), object(Nutrient$Type)
		valType := r.ReadObject()
		if valType == "" {
			// Null nutrient value — no fields to read
			continue
		}
		if isBackRef(valType) {
			continue
		}
		if !strings.Contains(valType, "models.Nutrient/") {
			// A different class where a Nutrient belongs means the layout
			// changed — walking it blind would corrupt every later field.
			return fmt.Errorf("nutrient map misaligned: expected Nutrient value, got %q", valType)
		}
		value := r.ReadDouble()
		r.ReadInt() // nutrient code (duplicated)
		// Nutrient$Type enum (first occurrence is new, rest are backrefs)
		ntType := r.ReadObject()
		if ntType != "" && !isBackRef(ntType) {
			r.ReadInt() // enum ordinal (only for first occurrence)
		}

		if code != 0 {
			f.Nutrients[code] = value
		}
	}
	return nil
}

// deserializeTranslationList reads an ArrayList of Translation objects.
func deserializeTranslationList(r *GWTReader, f *GWTFood) {
	typ := r.ReadObject()
	if typ == "" || isBackRef(typ) {
		return
	}
	if !strings.Contains(typ, "ArrayList") {
		return
	}
	count := r.ReadInt()
	for i := 0; i < count; i++ {
		lang, name := deserializeTranslation(r)
		if lang != "" && !isBackRef(lang) {
			f.Translations[lang] = name
		}
		// Keep the first name seen regardless of language so a food whose
		// preferred-language Language object arrived as a back-reference
		// (code unrecoverable) still resolves to a real name.
		if f.fallbackName == "" && name != "" {
			f.fallbackName = name
		}
	}
}

// deserializeTranslation reads a Translation object.
// 3 fields: object(Language), string(translatedName), int(?)
func deserializeTranslation(r *GWTReader) (langCode string, translatedName string) {
	typeSig := r.ReadObject()
	if typeSig == "" || isBackRef(typeSig) {
		return "", ""
	}

	// Language object (NOT an enum): 1 field which is a Locale-like inner object.
	//
	// Since the 2026-07-16 format the server reuses Language instances across
	// foods, so this can be a back-reference (observed: two foods sharing one
	// "de" instance). A back-reference carries no inner tokens and the code is
	// unrecoverable without full object-identity tracking — but the
	// Translation's remaining fields (name string, int) are still on the
	// stream and MUST be consumed, or every later field misaligns (this exact
	// early-return previously shifted the stream by 2 tokens per backref).
	langType := r.ReadObject()
	if langType != "" && !isBackRef(langType) {
		// Language inner object token resolves to the language code ("en", "fr", "de")
		// The inner object then has 3 string fields: displayName, flagURL, localizedName
		langCode = r.ReadObject() // type token = language code string
		if langCode != "" && !isBackRef(langCode) {
			r.ReadString() // displayName ("English")
			r.ReadString() // flagURL
			r.ReadString() // localizedName ("English")
		}
	}

	translatedName = r.ReadString()
	r.ReadInt() // unknown

	return langCode, translatedName
}

// skipArrayList reads and discards an ArrayList and all its elements.
// Handles String elements (barcode/UPC lists) and FoodTag elements (empty deserializer).
func skipArrayList(r *GWTReader) {
	typ := r.ReadObject()
	if typ == "" || isBackRef(typ) {
		return
	}
	count := r.ReadInt()
	for i := 0; i < count; i++ {
		elemType := r.ReadObject()
		if elemType == "" || isBackRef(elemType) {
			continue
		}
		// String objects have one string field
		if strings.Contains(elemType, "String") {
			r.ReadString()
		}
		// FoodTag has empty deserializer — nothing extra to read
	}
}

// skipEnum reads an enum object (type token + ordinal).
func skipEnum(r *GWTReader) {
	typ := r.ReadObject()
	if typ == "" || isBackRef(typ) {
		return
	}
	r.ReadInt() // ordinal
}

// readEnumOrdinal reads an enum and returns its ordinal. Returns -1 for null.
func readEnumOrdinal(r *GWTReader) int {
	typ := r.ReadObject()
	if typ == "" || isBackRef(typ) {
		return -1
	}
	return r.ReadInt()
}

// skipEnumOrBackRef reads an enum type token. If it's a new enum, reads the ordinal.
// If it's a back-reference, does nothing extra.
func skipEnumOrBackRef(r *GWTReader) {
	typ := r.ReadObject()
	if typ == "" || isBackRef(typ) {
		return
	}
	r.ReadInt()
}

// skipHashSet reads and discards a HashSet of FoodTag enums.
func skipHashSet(r *GWTReader) {
	typ := r.ReadObject()
	if typ == "" || isBackRef(typ) {
		return
	}
	count := r.ReadInt()
	for i := 0; i < count; i++ {
		// FoodTag is an enum: instantiator reads ordinal, deserializer is empty.
		// So each element = type token + ordinal.
		skipEnum(r)
	}
}

// skipStringHashMap reads and discards a HashMap<String,String>.
// If the object is not a HashMap (null or different type), does nothing.
func skipStringHashMap(r *GWTReader) {
	typ := r.ReadObject()
	if typ == "" || isBackRef(typ) {
		return
	}
	if !strings.Contains(typ, "HashMap") {
		// Not a HashMap — might be null or a different object type.
		// For non-HashMap objects, we need to handle them properly.
		// Check if it's a known skippable type, otherwise this is a problem.
		if strings.Contains(typ, "HashSet") {
			// Accidentally read into field 17's HashSet — put it back? No, can't.
			// This means field 14 was null and we consumed field 17's token.
			// We need to handle this differently.
		}
		return
	}
	count := r.ReadInt()
	for i := 0; i < count; i++ {
		// Key: String object + string value
		keyType := r.ReadObject()
		if keyType != "" && !isBackRef(keyType) {
			r.ReadString()
		} else if isBackRef(keyType) {
			r.ReadString()
		}
		// Value: String object + string value
		valType := r.ReadObject()
		if valType != "" && !isBackRef(valType) {
			r.ReadString()
		} else if isBackRef(valType) {
			r.ReadString()
		}
	}
}

// isBackRef returns true if the type string is a back-reference marker.
func isBackRef(typ string) bool {
	return strings.HasPrefix(typ, "__backref")
}
