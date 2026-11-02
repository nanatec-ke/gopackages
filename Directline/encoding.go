package directline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// The Hayabusa gateway is loosely typed, and encoding/json fails an entire
// decode on a single mismatch. /api/token/generate has been seen to answer with
// expires_in as a bare number (3600), as a quoted number ("3600") and as
// English prose ("60 Seconds"), each of which took down every authenticated
// call in turn. The gateway's scalar response fields are therefore decoded
// through the lenient types below, for the same reason the definite package
// decodes leniently — a call the gateway accepted must never fail to decode
// here.
//
// Requests keep plain Go types; only decoding is lenient.

// scalarText returns the JSON scalar in b as text with any surrounding quotes
// removed, and reports whether it was null.
func scalarText(b []byte) (text string, isNull bool) {
	s := string(bytes.TrimSpace(b))
	if s == "" || s == "null" {
		return "", true
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		var unquoted string
		if err := json.Unmarshal(b, &unquoted); err == nil {
			return strings.TrimSpace(unquoted), false
		}
		return strings.TrimSpace(s[1 : len(s)-1]), false
	}
	return s, false
}

// flexInt is an integer that also decodes from a numeric string, an integral
// float such as 3600.0, or null (as 0).
type flexInt int

// UnmarshalJSON implements json.Unmarshaler.
func (f *flexInt) UnmarshalJSON(b []byte) error {
	s, isNull := scalarText(b)
	if isNull || s == "" {
		*f = 0
		return nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		*f = flexInt(n)
		return nil
	}
	x, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(x, 0) || math.IsNaN(x) || x != math.Trunc(x) {
		return fmt.Errorf("directline: cannot decode %s as an integer", b)
	}
	*f = flexInt(int64(x))
	return nil
}

// flexFloat is a number that also decodes from a numeric string or null (as 0).
type flexFloat float64

// UnmarshalJSON implements json.Unmarshaler.
func (f *flexFloat) UnmarshalJSON(b []byte) error {
	s, isNull := scalarText(b)
	if isNull || s == "" {
		*f = 0
		return nil
	}
	x, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(x, 0) || math.IsNaN(x) {
		return fmt.Errorf("directline: cannot decode %s as a number", b)
	}
	*f = flexFloat(x)
	return nil
}

// flexBool is a boolean that also decodes from "true"/"false", 1/0 or null
// (as false).
type flexBool bool

// UnmarshalJSON implements json.Unmarshaler.
func (f *flexBool) UnmarshalJSON(b []byte) error {
	s, isNull := scalarText(b)
	if isNull || s == "" {
		*f = false
		return nil
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return fmt.Errorf("directline: cannot decode %s as a boolean", b)
	}
	*f = flexBool(v)
	return nil
}

// parseTTLSeconds reads a token lifetime and returns it in seconds. It accepts
// a bare count of seconds ("3600"), a decimal ("3600.0") and a count carried by
// an English unit ("60 Seconds", "1 hour", "30 min", "90s"). A bare number is
// read as seconds, which is what the documented shape means.
func parseTTLSeconds(s string) (int, bool) {
	s = strings.TrimSpace(s)
	i := 0
	for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.' || s[i] == '+' || s[i] == '-') {
		i++
	}
	count, err := strconv.ParseFloat(strings.TrimSpace(s[:i]), 64)
	if err != nil || math.IsNaN(count) || math.IsInf(count, 0) || count <= 0 {
		return 0, false
	}

	var perUnit float64
	switch unit := strings.ToLower(strings.TrimSpace(s[i:])); {
	case unit == "", strings.HasPrefix(unit, "s"):
		perUnit = 1
	case strings.HasPrefix(unit, "ms"), strings.HasPrefix(unit, "milli"):
		perUnit = 0.001
	case strings.HasPrefix(unit, "m"):
		perUnit = 60
	case strings.HasPrefix(unit, "h"):
		perUnit = 3600
	case strings.HasPrefix(unit, "d"):
		perUnit = 86400
	default:
		// An unrecognised unit makes the number meaningless, so report nothing
		// rather than guess at seconds.
		return 0, false
	}

	switch secs := count * perUnit; {
	case secs < 1:
		return 0, false
	case secs > math.MaxInt32:
		return math.MaxInt32, true
	default:
		return int(secs), true
	}
}

// expiresInSeconds is a token lifetime in seconds.
//
// It never fails to decode. expires_in only decides how long a token is cached,
// and Config.TokenTTL already stands in when the gateway does not supply one —
// so a token the gateway issued must not be thrown away because its lifetime
// was written in a shape this package does not recognise. That is exactly what
// happened to "60 Seconds". Anything unreadable decodes as 0, which GetToken
// treats as "not supplied".
type expiresInSeconds int

// UnmarshalJSON implements json.Unmarshaler.
func (e *expiresInSeconds) UnmarshalJSON(b []byte) error {
	*e = 0
	if s, isNull := scalarText(b); !isNull && s != "" {
		if secs, ok := parseTTLSeconds(s); ok {
			*e = expiresInSeconds(secs)
		}
	}
	return nil
}
