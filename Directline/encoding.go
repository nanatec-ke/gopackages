package directline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// The Hayabusa gateway is loosely typed. /api/token/generate answers with
// expires_in quoted as a string ("3600") although the documentation shows a
// number, and encoding/json fails an entire decode on a single such mismatch:
// every authenticated call lost its token to
// "cannot unmarshal string into Go struct field TokenResponse.expires_in of
// type int". The gateway's scalar response fields are therefore decoded through
// the lenient types below, for the same reason the definite package decodes
// leniently — a call the gateway accepted must never fail to decode here.
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
