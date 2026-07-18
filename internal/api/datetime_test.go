package api

import "testing"

func TestValidators(t *testing.T) {
	cases := []struct {
		name    string
		fn      func(string) (string, bool)
		in      string
		want    string
		wantOK  bool
	}{
		{"date ok", validDate, "2026-07-15", "2026-07-15", true},
		{"date requires 2-digit parts", validDate, "2026-7-5", "", false},
		{"date bad month", validDate, "2026-13-40", "", false},
		{"date junk", validDate, "banana", "", false},
		{"clock ok", validClock, "08:00", "08:00", true},
		{"clock normalizes", validClock, "8:00", "08:00", true},
		{"clock bad", validClock, "25:99", "", false},
		{"naive minute", validNaiveTime, "2026-07-15T14:00", "2026-07-15T14:00", true},
		{"naive seconds", validNaiveTime, "2026-07-15T14:00:30", "2026-07-15T14:00", true},
		{"naive rejects offset", validNaiveTime, "2026-07-15T14:00:00+09:00", "", false},
		{"naive rejects Z", validNaiveTime, "2026-07-15T14:00:00Z", "", false},
		{"zone ok", validZone, "Asia/Seoul", "Asia/Seoul", true},
		{"zone bad", validZone, "Mars/Phobos", "", false},
	}
	for _, tc := range cases {
		got, ok := tc.fn(tc.in)
		if ok != tc.wantOK || got != tc.want {
			t.Errorf("%s: got (%q,%v), want (%q,%v)", tc.name, got, ok, tc.want, tc.wantOK)
		}
	}
}
