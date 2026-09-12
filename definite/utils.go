package definite

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	dateRe   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(\.\d{1,3})?$`)
	emailRe  = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]{2,}$`)
	kraPinRe = regexp.MustCompile(`(?i)^[A-Z]\d{9,10}[A-Z]$`)
	nonDigit = regexp.MustCompile(`\D`)
)

// FormatDate formats t in the API's YYYY-MM-DD HH:MM:SS.mmm shape, in t's own location.
func FormatDate(t time.Time) string { return t.Format(DateLayout) }

// ParseDate parses a YYYY-MM-DD HH:MM:SS[.mmm] timestamp, as returned by the
// API, in the local time zone. A "T" separator is also accepted.
//
// Impossible dates such as 2026-02-30 are rejected rather than rolled over.
func ParseDate(value string) (time.Time, error) {
	s := strings.TrimSpace(value)
	if !dateRe.MatchString(s) {
		return time.Time{}, &ClientError{
			Type: InternalError, Code: ErrParseTime, Operation: "ParseDate",
			Message: fmt.Sprintf("unrecognised date format: %q", value),
		}
	}
	// A fractional second after the seconds field is accepted even though the
	// layout does not name one.
	t, err := time.ParseInLocation("2006-01-02 15:04:05", strings.Replace(s, "T", " ", 1), time.Local)
	if err != nil {
		return time.Time{}, &ClientError{
			Type: InternalError, Code: ErrParseTime, Operation: "ParseDate",
			Message: fmt.Sprintf("invalid date %q: %v", value, err), Err: err,
		}
	}
	return t, nil
}

// AddPolicyYear adds whole years to a cover start date and returns the day
// before the anniversary, matching the API's inclusive cover period.
//
//	AddPolicyYear(2026-06-02, 1) // 2027-06-01
func AddPolicyYear(start time.Time, years int) time.Time {
	return start.AddDate(years, 0, -1)
}

// NormalizeKenyanPhone converts a Kenyan phone number to the 2547XXXXXXXX form
// the payments endpoint expects. It accepts 07XXXXXXXX, +2547XXXXXXXX,
// 2547XXXXXXXX and 7XXXXXXXX, with any spaces, dashes or parentheses.
func NormalizeKenyanPhone(phone string) (string, error) {
	digits := nonDigit.ReplaceAllString(phone, "")
	if digits == "" {
		return "", &ClientError{Type: InternalError, Code: ErrInvalidInput, Operation: "NormalizeKenyanPhone",
			Message: "phone number is required"}
	}
	var normalized string
	switch {
	case strings.HasPrefix(digits, "254"):
		normalized = digits
	case strings.HasPrefix(digits, "0"):
		normalized = "254" + digits[1:]
	case len(digits) == 9:
		normalized = "254" + digits
	}
	if len(normalized) != 12 {
		return "", &ClientError{Type: InternalError, Code: ErrInvalidInput, Operation: "NormalizeKenyanPhone",
			Message: fmt.Sprintf("unrecognised Kenyan phone number: %q", phone)}
	}
	return normalized, nil
}

// IsValidEmail reports whether s looks like an email address.
func IsValidEmail(s string) bool { return emailRe.MatchString(strings.TrimSpace(s)) }

// IsValidKraPin reports whether s looks like a KRA PIN: a letter, nine or ten
// digits, then a check letter (e.g. A0101401976H).
//
// Advisory only — the validators require a PIN but do not enforce this
// pattern, so an unusual but legitimate PIN is never rejected client-side.
func IsValidKraPin(s string) bool { return kraPinRe.MatchString(strings.TrimSpace(s)) }

// PaymentIntervalDescription describes a PaymentInterval constant.
func PaymentIntervalDescription(interval string) string {
	switch interval {
	case PaymentIntervalAnnually:
		return "Single annual premium"
	case PaymentIntervalMonthly:
		return "Twelve monthly instalments"
	case PaymentIntervalQuarterly:
		return "Four quarterly instalments"
	case PaymentIntervalTwoInstallments:
		return "Two instalments over the cover period"
	default:
		return fmt.Sprintf("Unknown payment interval: %s", interval)
	}
}

// PaymentMethodDescription describes a PaymentMethod constant.
func PaymentMethodDescription(method int) string {
	switch method {
	case PaymentMethodMpesa:
		return "M-Pesa"
	default:
		return fmt.Sprintf("Unknown payment method: %d", method)
	}
}

// TransactionTypeDescription describes a TransactionType constant.
func TransactionTypeDescription(transactionType int) string {
	switch transactionType {
	case TransactionTypeNewBusiness:
		return "New business"
	default:
		return fmt.Sprintf("Unknown transaction type: %d", transactionType)
	}
}
