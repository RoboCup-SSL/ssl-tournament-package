// Format validators for the model's date/time text fields. Each parses its
// layout, rejecting malformed input, and returns the canonical form. The
// backend stays timezone-naive: scheduling instants carry no offset.
package api

import (
	"time"

	_ "time/tzdata" // embed the IANA zone database so validZone works on any host
)

// validDate accepts an ISO calendar date and returns it canonicalized.
func validDate(value string) (string, bool) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return "", false
	}
	return parsed.Format("2006-01-02"), true
}

// validClock accepts a 24-hour wall-clock time (HH:MM) and canonicalizes it.
func validClock(value string) (string, bool) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return "", false
	}
	return parsed.Format("15:04"), true
}

// validNaiveTime accepts a naive local datetime (optional seconds, no offset)
// and returns canonical minute-precision form. An offset or Z is rejected.
func validNaiveTime(value string) (string, bool) {
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format("2006-01-02T15:04"), true
		}
	}
	return "", false
}

// validZone accepts an IANA timezone name (e.g. Asia/Seoul).
func validZone(value string) (string, bool) {
	if _, err := time.LoadLocation(value); err != nil {
		return "", false
	}
	return value, true
}
