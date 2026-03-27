package gocronometer

import (
	"fmt"
	"strconv"
	"strings"
)

// GWTReader deserializes GWT RPC responses.
// Ported from com.gdevelop.gwt.syncrpc.SyncClientSerializationStreamReader.
//
// Response format: //OK[token1,token2,...,["stringTable"],flags,version]
// Tokens are consumed from the END via --index (stack-like).
type GWTReader struct {
	tokens      []string // all tokens from the response payload
	stringTable []string // the string table extracted from the payload
	index       int      // current read position (decrements)
	version     int
	flags       int
}

// NewGWTReader parses a GWT RPC response string and prepares for reading.
func NewGWTReader(encoded string) (*GWTReader, error) {
	// Strip //OK[ prefix or //EX[ prefix
	if strings.HasPrefix(encoded, "//OK") {
		encoded = encoded[4:]
	} else if strings.HasPrefix(encoded, "//EX") {
		return nil, fmt.Errorf("GWT RPC exception response")
	}

	// Handle ].concat([ patterns (large responses split across arrays)
	encoded = deconcatGWT(encoded)

	// Strip outer brackets
	if strings.HasPrefix(encoded, "[") && strings.HasSuffix(encoded, "]") {
		encoded = encoded[1 : len(encoded)-1]
	}

	// Parse into tokens, extracting the string table
	tokens, stringTableRaw, err := parseGWTTokens(encoded)
	if err != nil {
		return nil, fmt.Errorf("parsing tokens: %w", err)
	}

	// Parse string table
	stringTable, err := parseGWTStringTableRaw(stringTableRaw)
	if err != nil {
		return nil, fmt.Errorf("parsing string table: %w", err)
	}

	// Remove the string table placeholder and any empty tokens around it
	var cleanTokens []string
	for _, t := range tokens {
		if t == "__stringtable__" || t == "" {
			continue
		}
		cleanTokens = append(cleanTokens, t)
	}

	r := &GWTReader{
		tokens:      cleanTokens,
		stringTable: stringTable,
		index:       len(cleanTokens),
	}

	// Read version and flags (last two tokens)
	r.version = r.ReadInt()
	r.flags = r.ReadInt()

	return r, nil
}

// ReadInt reads the next integer token (decrements index).
func (r *GWTReader) ReadInt() int {
	if r.index <= 0 {
		return 0
	}
	r.index--
	v, _ := strconv.Atoi(r.tokens[r.index])
	return v
}

// ReadDouble reads the next double token.
func (r *GWTReader) ReadDouble() float64 {
	if r.index <= 0 {
		return 0
	}
	r.index--
	v, _ := strconv.ParseFloat(r.tokens[r.index], 64)
	return v
}

// ReadString reads the next string value (reads an int index, looks up in string table).
// Returns "" for null (index 0).
func (r *GWTReader) ReadString() string {
	idx := r.ReadInt()
	return r.GetString(idx)
}

// ReadLong reads a long value. In GWT v7+, longs are base64 encoded strings.
// In older versions, they're two doubles added together.
func (r *GWTReader) ReadLong() int64 {
	if r.version >= 7 {
		if r.index <= 0 {
			return 0
		}
		r.index--
		s := r.tokens[r.index]
		// Remove quotes if present
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			s = s[1 : len(s)-1]
		}
		v, _ := longFromBase64(s)
		return v
	}
	// Old format: two doubles
	return int64(r.ReadDouble()) + int64(r.ReadDouble())
}

// ReadBoolean reads a boolean token.
func (r *GWTReader) ReadBoolean() bool {
	return r.ReadInt() != 0
}

// ReadObject reads a type token and returns the type signature string.
// Returns "" for null objects (token 0).
// Negative tokens are back-references to previously seen objects (not yet supported).
func (r *GWTReader) ReadObject() string {
	token := r.ReadInt()
	if token == 0 {
		return "" // null
	}
	if token < 0 {
		// Back-reference to previously deserialized object
		// Not fully supported — return a marker
		return fmt.Sprintf("__backref_%d", -(token + 1))
	}
	// Positive token = 1-based string table index for type signature
	return r.GetString(token)
}

// GetString returns the string at the given 1-based index in the string table.
// Returns "" for index 0 (null).
func (r *GWTReader) GetString(index int) string {
	if index == 0 {
		return ""
	}
	if index < 0 || index > len(r.stringTable) {
		return ""
	}
	return r.stringTable[index-1]
}

// Remaining returns how many tokens are left to read.
func (r *GWTReader) Remaining() int {
	return r.index
}

// Version returns the GWT stream version.
func (r *GWTReader) Version() int {
	return r.version
}

// Tokens returns all tokens for debugging.
func (r *GWTReader) Tokens() []string {
	return r.tokens
}

// Peek returns the next token without consuming it.
func (r *GWTReader) Peek() string {
	if r.index <= 0 {
		return ""
	}
	return r.tokens[r.index-1]
}

// Skip skips n tokens.
func (r *GWTReader) Skip(n int) {
	r.index -= n
	if r.index < 0 {
		r.index = 0
	}
}

// StringTable returns the full string table.
func (r *GWTReader) StringTable() []string {
	return r.stringTable
}

// parseGWTTokens splits the GWT response payload into tokens and extracts the string table.
// The string table is the [...] array embedded in the token stream.
func parseGWTTokens(encoded string) (tokens []string, stringTableRaw string, err error) {
	var current strings.Builder
	inString := false
	escaped := false

	for i := 0; i < len(encoded); i++ {
		ch := encoded[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			current.WriteByte(ch)
			escaped = true
			continue
		}

		if ch == '"' {
			current.WriteByte(ch)
			inString = !inString
			continue
		}

		if inString {
			current.WriteByte(ch)
			continue
		}

		if ch == ',' {
			tokens = append(tokens, current.String())
			current.Reset()
			continue
		}

		if ch == '[' {
			// Find matching ] — this is the string table
			depth := 1
			start := i
			inStr := false
			esc := false
			for j := i + 1; j < len(encoded); j++ {
				if esc {
					esc = false
					continue
				}
				if encoded[j] == '\\' && inStr {
					esc = true
					continue
				}
				if encoded[j] == '"' {
					inStr = !inStr
					continue
				}
				if !inStr {
					if encoded[j] == '[' {
						depth++
					} else if encoded[j] == ']' {
						depth--
						if depth == 0 {
							stringTableRaw = encoded[start+1 : j]
							// Add a placeholder token for the string table position
							tokens = append(tokens, "__stringtable__")
							i = j
							break
						}
					}
				}
			}
			continue
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens, stringTableRaw, nil
}

// parseGWTStringTableRaw parses the raw string table content (inside the [...] brackets).
// Handles escaped characters including \uNNNN, \xNN, \0, \n, \t, etc.
func parseGWTStringTableRaw(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}

	var result []string
	var buf strings.Builder
	inString := false

	for i := 0; i < len(raw); i++ {
		ch := raw[i]

		if !inString {
			if ch == '"' {
				inString = true
				buf.Reset()
			}
			// Skip commas and whitespace between strings
			continue
		}

		// Inside a quoted string
		if ch == '"' {
			// End of string
			result = append(result, buf.String())
			inString = false
			continue
		}

		if ch == '\\' && i+1 < len(raw) {
			i++
			next := raw[i]
			switch next {
			case '0':
				buf.WriteByte(0)
			case 'b':
				buf.WriteByte('\b')
			case 't':
				buf.WriteByte('\t')
			case 'n':
				buf.WriteByte('\n')
			case 'f':
				buf.WriteByte('\f')
			case 'r':
				buf.WriteByte('\r')
			case '"':
				buf.WriteByte('"')
			case '\\':
				buf.WriteByte('\\')
			case 'x':
				// \xNN
				if i+2 < len(raw) {
					b, err := strconv.ParseUint(raw[i+1:i+3], 16, 8)
					if err == nil {
						buf.WriteByte(byte(b))
						i += 2
					}
				}
			case 'u':
				// \uNNNN
				if i+4 < len(raw) {
					b, err := strconv.ParseUint(raw[i+1:i+5], 16, 16)
					if err == nil {
						buf.WriteRune(rune(b))
						i += 4
					}
				}
			default:
				buf.WriteByte(next)
			}
			continue
		}

		buf.WriteByte(ch)
	}

	return result, nil
}

// deconcatGWT handles GWT responses that split large arrays using .concat().
// Format: [data1].concat([data2],[data3])
func deconcatGWT(encoded string) string {
	const prelude = "].concat(["
	idx := strings.Index(encoded, prelude)
	if idx < 0 {
		return encoded
	}

	var result strings.Builder
	result.WriteString(encoded[:idx])

	rest := encoded[idx+len(prelude):]
	for {
		// Find next ],[ or ])
		endBracket := strings.Index(rest, "],[")
		endFinal := strings.Index(rest, "])")

		if endBracket >= 0 && (endFinal < 0 || endBracket < endFinal) {
			result.WriteByte(',')
			result.WriteString(rest[:endBracket])
			rest = rest[endBracket+3:]
		} else if endFinal >= 0 {
			result.WriteByte(',')
			result.WriteString(rest[:endFinal])
			result.WriteByte(']')
			break
		} else {
			result.WriteByte(',')
			result.WriteString(rest)
			break
		}
	}

	return result.String()
}

// longFromBase64 decodes a GWT base64-encoded long value.
// GWT uses a custom base64 encoding for longs.
func longFromBase64(s string) (int64, error) {
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_$"

	var value int64
	for _, ch := range s {
		idx := strings.IndexRune(base64Chars, ch)
		if idx < 0 {
			return 0, fmt.Errorf("invalid base64 char: %c", ch)
		}
		value = (value << 6) | int64(idx)
	}
	return value, nil
}
