package definite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// The Definite API's response shapes are only partly documented, and two
// endpoints have already turned out to differ from their documentation.
// encoding/json fails an entire decode on a single type mismatch — a number
// where a string was expected — so every response field uses one of the
// lenient types below. A call that succeeded at Definite must never fail to
// decode here: for newProposal that would invite a retry, and a duplicate
// proposal at the underwriter.
//
// Requests use plain Go types; only decoding is lenient.

// FlexInt is an integer that also decodes from a numeric string, an integral
// float such as 12.0, or null (as 0).
type FlexInt int64

// UnmarshalJSON implements json.Unmarshaler.
func (f *FlexInt) UnmarshalJSON(b []byte) error {
	s, isNull := unquoteScalar(b)
	if isNull || s == "" {
		*f = 0
		return nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		*f = FlexInt(n)
		return nil
	}
	x, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(x, 0) || x != math.Trunc(x) {
		return fmt.Errorf("definite: cannot decode %s as an integer", b)
	}
	*f = FlexInt(int64(x))
	return nil
}

// FlexFloat is a number that also decodes from a numeric string or null (as 0).
type FlexFloat float64

// UnmarshalJSON implements json.Unmarshaler.
func (f *FlexFloat) UnmarshalJSON(b []byte) error {
	s, isNull := unquoteScalar(b)
	if isNull || s == "" {
		*f = 0
		return nil
	}
	x, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("definite: cannot decode %s as a number", b)
	}
	*f = FlexFloat(x)
	return nil
}

// FlexString is a string that also decodes from any other JSON value. Numbers,
// booleans, objects and arrays keep their JSON text; null decodes as "".
type FlexString string

// UnmarshalJSON implements json.Unmarshaler.
func (f *FlexString) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	switch {
	case len(b) == 0 || string(b) == "null":
		*f = ""
	case b[0] == '"':
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = FlexString(s)
	default:
		*f = FlexString(b)
	}
	return nil
}

// FlexBool is a boolean that also decodes from "true"/"false", 1/0 and
// "1"/"0". Response structs hold it as *FlexBool so an absent field (nil) can
// be told apart from an explicit false.
type FlexBool bool

// UnmarshalJSON implements json.Unmarshaler.
func (f *FlexBool) UnmarshalJSON(b []byte) error {
	s, isNull := unquoteScalar(b)
	if isNull {
		*f = false
		return nil
	}
	switch strings.ToLower(s) {
	case "true", "1":
		*f = true
	case "false", "0", "":
		*f = false
	default:
		return fmt.Errorf("definite: cannot decode %s as a boolean", b)
	}
	return nil
}

// unquoteScalar returns the text of a JSON scalar with any string quoting
// removed, and whether the value was null.
func unquoteScalar(b []byte) (string, bool) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return "", true
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err == nil {
			return strings.TrimSpace(s), false
		}
	}
	return string(b), false
}

// Timestamp is a request date, serialised in the API's YYYY-MM-DD HH:MM:SS.mmm
// shape. It is formatted in the time's own location — no zone conversion is
// applied, so convert with In() first if the API needs a particular zone.
//
// The zero Timestamp marshals as null; the request validators reject it for
// every required date.
type Timestamp struct {
	time.Time
}

// NewTimestamp wraps t as a Timestamp.
func NewTimestamp(t time.Time) Timestamp { return Timestamp{Time: t} }

// String returns the timestamp in DateLayout.
func (t Timestamp) String() string { return FormatDate(t.Time) }

// MarshalJSON implements json.Marshaler.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(FormatDate(t.Time))
}

// UnmarshalJSON implements json.Unmarshaler, accepting anything ParseDate does.
func (t *Timestamp) UnmarshalJSON(b []byte) error {
	s, isNull := unquoteScalar(b)
	if isNull || s == "" {
		*t = Timestamp{}
		return nil
	}
	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}
	*t = Timestamp{Time: parsed}
	return nil
}
