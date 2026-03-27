package gocronometer

import (
	"fmt"
	"strings"
)

// GWTFood represents a deserialized Food object from the GWT response.
type GWTFood struct {
	ID           int64
	ID2          int64
	TypeEnum     int // FoodType enum ordinal
	Ingredients  []GWTIngredient
	Nutrients    map[int]float64 // USDA nutrient code → value per 100g
	Weight       float64
	Name         string
	Description  string
	Measures     []GWTMeasure
	TagName      string
	Source       string // e.g. "NCCDB:13930", "CRDB"
	Translations map[string]string // language code → translated name
}

// GWTIngredient represents a deserialized Ingredient from the GWT response.
type GWTIngredient struct {
	FoodID    int64
	MeasureID int64
	Amount    float64
	Hash      string
}

// GWTMeasure represents a serving size measure.
type GWTMeasure struct {
	ID     int64
	FoodID int64
	Amount float64
	Name   string
}

// DeserializeFood reads a Food object from the GWT stream.
// Must be called after ReadObject() returns the Food type signature.
//
// Food fields (16 total, from GWT JS analysis):
//  1. long   — ID
//  2. long   — secondary ID
//  3. int    — FoodType enum ordinal
//  4. list   — ingredients (ArrayList of Ingredient)
//  5. object — NutrientMap
//  6. double — weight
//  7. string — name
//  8. string — description
//  9. list   — measures (ArrayList of Measure)
// 10. string — tag name
// 11. object — unknown (type 507 — nutrition label?)
// 12. long   — unknown
// 13. string — source tag
// 14. object — unknown (type 105 — FoodTag set?)
// 15. long   — unknown
// 16. list   — translations
func DeserializeFood(r *GWTReader) (*GWTFood, error) {
	f := &GWTFood{
		Nutrients:    make(map[int]float64),
		Translations: make(map[string]string),
	}

	// From the raw data trace, reading backwards from the type token:
	// [0]=0 [1]=0 [2]=2→ArrayList [3]=1→Food [4]=3→String [5]=4→"4076"
	// [6]=26→FoodTag [7]=5→description [8]=0 [9]=0 [10]=466098(ID)
	// [11]=0 [12]=6→NutritionLabel [13]=0 [14]="Zjjf7wA" [15]=7→FoodMeasures
	//
	// Revised field mapping based on actual data:
	// The JS has: Mn, Mn, Jn, Ln, Bn, Nn, Pn, Pn, Ln, Pn, Bn, Mn, Pn, Bn, Mn, Ln
	// Where Mn/Jn might both be readInt, and Ln/Bn are readObject variants
	//
	// Actual interpretation from data:
	// Field 1 (Mn): 0 → null object or zero int
	// Field 2 (Mn): 0 → null object or zero int
	// Field 3 (Jn): 2 → BUT 2=ArrayList type sig, so this is NOT readInt!
	//
	// Reinterpretation: Mn=readInt, Jn=readInt, but Field 3 value=2 just happens
	// to equal the ArrayList string index. The NEXT field (Ln=readObject) reads
	// the REAL ArrayList token.
	//
	// Wait — [2]=2 as readInt is just the integer 2 (FoodType ordinal).
	// [3]=1 as readObject → getString(1) = Food type (back-ref?)
	//
	// Let's try: maybe Ln() IS different from Bn(). Let me just read all 16 fields
	// as raw tokens and match against the known data.

	// Simple approach: dump remaining tokens until we find recognizable landmarks
	// Field 1-2: Mn = readInt
	r.ReadInt() // 0
	r.ReadInt() // 0

	// Field 3: Jn = readInt → 2 (FoodType ordinal)
	f.TypeEnum = r.ReadInt()

	// Field 4: Ln = readObject → for null list, token=0; for list, token=ArrayList type index
	ingList := r.ReadObject()
	if ingList != "" && !strings.HasPrefix(ingList, "__backref") &&
		strings.Contains(ingList, "ArrayList") {
		count := r.ReadInt()
		for i := 0; i < count; i++ {
			ing, err := deserializeIngredient(r)
			if err != nil {
				return f, fmt.Errorf("ingredient %d: %w", i, err)
			}
			f.Ingredients = append(f.Ingredients, ing)
		}
	}

	// Field 5: Bn = readObject → NutrientMap (or FoodMeasures?)
	f5type := r.ReadObject()
	if f5type != "" && !strings.HasPrefix(f5type, "__backref") {
		if strings.Contains(f5type, "NutrientMap") {
			deserializeNutrientMap(r, f)
		} else if strings.Contains(f5type, "String") {
			// This might be a string field, not an object
			// Back up and re-read
		}
	}

	// Field 6: Nn = readDouble
	f.Weight = r.ReadDouble()

	// Field 7: Pn = readString → name
	f.Name = r.ReadString()

	// Field 8: Pn = readString → description
	f.Description = r.ReadString()

	// Field 9: Ln = readObject → measures list
	measList := r.ReadObject()
	if measList != "" && !strings.HasPrefix(measList, "__backref") &&
		strings.Contains(measList, "ArrayList") {
		count := r.ReadInt()
		for i := 0; i < count; i++ {
			m, err := deserializeMeasure(r)
			if err != nil {
				break
			}
			f.Measures = append(f.Measures, m)
		}
	}

	// Field 10: Pn = readString → tag name
	f.TagName = r.ReadString()

	// Field 11: Bn = readObject → NutritionLabelType
	skipObject(r)

	// Field 12: Mn = readInt
	r.ReadInt()

	// Field 13: Pn = readString → source tag
	f.Source = r.ReadString()

	// Field 14: Bn = readObject → FoodTag set
	tagSetType := r.ReadObject()
	if tagSetType != "" && !strings.HasPrefix(tagSetType, "__backref") {
		if strings.Contains(tagSetType, "HashSet") {
			count := r.ReadInt()
			for i := 0; i < count; i++ {
				skipObject(r)
			}
		}
	}

	// Field 15: Mn = readInt
	r.ReadInt()

	// Field 16: Ln = readObject → translations list
	transList := r.ReadObject()
	if transList != "" && !strings.HasPrefix(transList, "__backref") &&
		strings.Contains(transList, "ArrayList") {
		count := r.ReadInt()
		for i := 0; i < count; i++ {
			lang, name := deserializeTranslation(r)
			if lang != "" {
				f.Translations[lang] = name
			}
		}
	}

	// Set display name: prefer English translation, fall back to German, then TagName, then Name
	if en, ok := f.Translations["en"]; ok && en != "" {
		f.Name = en
	} else if de, ok := f.Translations["de"]; ok && de != "" && f.Name == "" {
		f.Name = de
	}
	if f.Name == "" {
		f.Name = f.TagName
	}

	return f, nil
}

// deserializeIngredient reads an Ingredient from the stream.
//
// Ingredient fields (14 total, from GWT JS analysis):
//  1. object (long wrapper)  — unknown
//  2. object (long wrapper)  — food ID
//  3. object (long wrapper)  — measure ID
//  4. object (Map)           — nutrient overrides
//  5. object (long wrapper)  — unknown
//  6. object (long wrapper)  — unknown
//  7. object (long wrapper)  — unknown
//  8. int                    — unknown
//  9. object                 — Measure reference
// 10. string                 — hash
// 11. long                   — unknown
// 12. string                 — name/description
// 13. object (long wrapper)  — unknown
// 14. object (long wrapper)  — unknown
func deserializeIngredient(r *GWTReader) (GWTIngredient, error) {
	ing := GWTIngredient{}

	// Read type signature
	typeSig := r.ReadObject()
	if typeSig == "" {
		return ing, fmt.Errorf("null ingredient")
	}

	// Field 1: object (long wrapper) — unknown
	readLongWrapper(r)

	// Field 2: object (long wrapper) — food ID
	ing.FoodID = readLongWrapper(r)

	// Field 3: object (long wrapper) — measure ID
	ing.MeasureID = readLongWrapper(r)

	// Field 4: object (Map) — nutrient overrides
	skipObject(r)

	// Field 5-7: object (long wrappers) — unknown
	readLongWrapper(r)
	readLongWrapper(r)
	readLongWrapper(r)

	// Field 8: int — unknown
	r.ReadInt()

	// Field 9: object — Measure reference
	skipObject(r)

	// Field 10: string — hash
	ing.Hash = r.ReadString()

	// Field 11: long — amount (as raw long)
	ing.Amount = r.ReadDouble() // actually stored as a long that represents the amount

	// Field 12: string — name/description
	// skip
	r.ReadString()

	// Field 13-14: object (long wrappers) — unknown
	readLongWrapper(r)
	readLongWrapper(r)

	return ing, nil
}

// deserializeNutrientMap reads a NutrientMap from the stream.
// NutrientMap contains a NutrientFilter enum and a HashMap of nutrient code → value.
func deserializeNutrientMap(r *GWTReader, f *GWTFood) {
	// NutrientMap has a filter field
	filterType := r.ReadObject() // NutrientFilter enum
	if filterType != "" && !strings.HasPrefix(filterType, "__backref") {
		r.ReadInt() // enum ordinal
	}

	// HashMap of Integer → Nutrient
	mapType := r.ReadObject()
	if mapType == "" || strings.HasPrefix(mapType, "__backref") {
		return
	}

	count := r.ReadInt()
	for i := 0; i < count; i++ {
		// Key: Integer (nutrient code)
		keyType := r.ReadObject()
		if keyType == "" {
			continue
		}
		code := r.ReadInt()

		// Value: Nutrient object
		nutrientType := r.ReadObject()
		if nutrientType == "" || strings.HasPrefix(nutrientType, "__backref") {
			continue
		}

		// Nutrient has: type enum ordinal, then double value
		nutTypeEnum := r.ReadObject() // Nutrient$Type enum
		if nutTypeEnum != "" && !strings.HasPrefix(nutTypeEnum, "__backref") {
			r.ReadInt() // enum ordinal
		}
		value := r.ReadDouble()

		if code > 0 {
			f.Nutrients[code] = value
		}
	}
}

// deserializeMeasure reads a Measure from the stream.
func deserializeMeasure(r *GWTReader) (GWTMeasure, error) {
	m := GWTMeasure{}

	typeSig := r.ReadObject()
	if typeSig == "" {
		return m, fmt.Errorf("null measure")
	}

	// Check if it's a DerivedMeasure (has extra fields)
	isDerived := strings.Contains(typeSig, "DerivedMeasure")

	if isDerived {
		// DerivedMeasure extends Measure with conversion fields
		// Read the base double value
		r.ReadDouble() // conversion factor
		// Then read as regular measure
	}

	// Measure fields: type enum, ID, foodID, ?, amount, name?
	measType := r.ReadObject() // Measure$Type enum
	if measType != "" && !strings.HasPrefix(measType, "__backref") {
		r.ReadInt() // enum ordinal
	}

	m.ID = r.ReadLong()
	m.FoodID = r.ReadLong()

	// Read remaining fields — amount and possibly a flag
	r.ReadInt() // unknown flag

	m.Amount = r.ReadDouble()

	return m, nil
}

// deserializeTranslation reads a Translation object.
// Returns language code and translated name.
func deserializeTranslation(r *GWTReader) (string, string) {
	typeSig := r.ReadObject()
	if typeSig == "" || strings.HasPrefix(typeSig, "__backref") {
		return "", ""
	}

	// Translation fields: Language enum, then strings
	langType := r.ReadObject() // Language enum
	if langType == "" {
		return "", ""
	}
	if !strings.HasPrefix(langType, "__backref") {
		r.ReadInt() // enum ordinal
	}

	langCode := r.ReadString()     // "en", "de", etc.
	langDisplay := r.ReadString()  // "English", "German", etc.
	flagURL := r.ReadString()      // flag image URL
	translatedName := r.ReadString() // the actual translated food name

	_ = langDisplay
	_ = flagURL

	return langCode, translatedName
}

// readLongWrapper reads a wrapped Long object (java.lang.Long or similar).
// Returns 0 for null.
func readLongWrapper(r *GWTReader) int64 {
	typeSig := r.ReadObject()
	if typeSig == "" {
		return 0
	}
	if strings.HasPrefix(typeSig, "__backref") {
		return 0 // back-reference, can't resolve
	}
	return r.ReadLong()
}

// skipObject reads and discards an object token.
// For null objects (token 0), does nothing extra.
// For real objects, we can't skip the fields without knowing the type,
// so this only handles null and back-references.
func skipObject(r *GWTReader) {
	typeSig := r.ReadObject()
	if typeSig == "" || strings.HasPrefix(typeSig, "__backref") {
		return
	}

	// For known simple types, read their value
	if strings.Contains(typeSig, "NutritionLabelType") {
		r.ReadInt() // enum ordinal
	}
	// Other unknown types — we can't skip without knowing field count
}
