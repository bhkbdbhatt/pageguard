package cron

import (
	"testing"
	"time"
)

func mustNext(t *testing.T, expr, from, want string) {
	t.Helper()
	loc := time.UTC
	after, err := time.Parse(time.RFC3339, from)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Next(expr, after, loc)
	if err != nil {
		t.Fatalf("Next(%q): %v", expr, err)
	}
	wantTime, err := time.Parse(time.RFC3339, want)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(wantTime) {
		t.Errorf("Next(%q, %s) = %s, want %s", expr, from, got, wantTime)
	}
}

func TestNext(t *testing.T) {
	mustNext(t, "*/15 * * * *", "2026-10-23T14:31:00Z", "2026-10-23T14:45:00Z")
	mustNext(t, "*/15 * * * *", "2026-10-23T14:45:00Z", "2026-10-23T15:00:00Z")
	mustNext(t, "0 6 * * *", "2026-10-23T14:31:00Z", "2026-10-24T06:00:00Z")
	mustNext(t, "0 6 * * *", "2026-10-23T05:59:00Z", "2026-10-23T06:00:00Z")
	mustNext(t, "30 8 * * 1-5", "2026-10-23T14:31:00Z", "2026-10-26T08:30:00Z")
	mustNext(t, "0 2 1 * *", "2026-10-23T14:31:00Z", "2026-11-01T02:00:00Z")
	mustNext(t, "0 0 * * 0", "2026-10-23T14:31:00Z", "2026-10-25T00:00:00Z")
	mustNext(t, "0 0 * * 7", "2026-10-23T14:31:00Z", "2026-10-25T00:00:00Z")
	mustNext(t, "0 0 29 2 *", "2026-10-23T14:31:00Z", "2028-02-29T00:00:00Z")
	mustNext(t, "0 12 1-7 * mon", "2026-10-23T14:31:00Z", "2026-10-26T12:00:00Z")
}

func TestNextTimezone(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("timezone database unavailable")
	}
	after := time.Date(2026, 10, 23, 14, 31, 0, 0, time.UTC)
	got, err := Next("0 6 * * *", after, loc)
	if err != nil {
		t.Fatal(err)
	}
	if got.In(loc).Hour() != 6 {
		t.Errorf("hour = %d, want 6 (%s)", got.In(loc).Hour(), got)
	}
}

func TestParseErrors(t *testing.T) {
	for _, expr := range []string{
		"", "* * * *", "* * * * * *", "60 * * * *", "* 24 * * *",
		"* * 0 * *", "* * * 13 *", "* * * * 8", "*/0 * * * *",
		"5-1 * * * *", "a * * * *", "1,,2 * * * *",
	} {
		if _, err := Parse(expr); err == nil {
			t.Errorf("Parse(%q) expected error", expr)
		}
	}
}

func TestParseNamesAndSteps(t *testing.T) {
	s, err := Parse("0-30/10 8 * jan-mar mon-fri")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []int{0, 10, 20, 30} {
		if !s.minute.has(v) {
			t.Errorf("minute %d missing", v)
		}
	}
	if s.minute.has(40) || !s.hour.has(8) || !s.month.has(2) || !s.dow.has(3) {
		t.Errorf("unexpected parse result")
	}
}
