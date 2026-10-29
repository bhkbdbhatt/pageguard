// Package cron implements the small subset of 5-field cron expressions used by
// Pageguard schedules: minute, hour, day-of-month, month and day-of-week.
//
// Supported syntax: "*", numbers, names (months jan-dec, weekdays sun-sat),
// ranges ("1-5"), steps ("*/15", "0-30/10") and comma-separated lists. As in
// Vixie cron, when both day-of-month and day-of-week are restricted a day
// matches if either field matches.
package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// schedule is a parsed cron expression.
type schedule struct {
	minute, hour, dom, month, dow set

	domRestricted bool
	dowRestricted bool
}

// set is a bitset of matching field values.
type set uint64

func (s set) has(v int) bool { return s&(1<<uint(v)) != 0 }

func (s *set) add(v int) { *s |= 1 << uint(v) }

func (s set) values(min, max int) []int {
	out := make([]int, 0, max-min+1)
	for v := min; v <= max; v++ {
		if s.has(v) {
			out = append(out, v)
		}
	}
	return out
}

var monthNames = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
	"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

var dayNames = map[string]int{
	"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6,
}

// Parse parses a standard 5-field cron expression.
func Parse(expr string) (*schedule, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron: expected 5 fields, got %d", len(fields))
	}
	var s schedule
	var err error
	if s.minute, err = parseField(fields[0], 0, 59, nil); err != nil {
		return nil, fmt.Errorf("cron: minute: %w", err)
	}
	if s.hour, err = parseField(fields[1], 0, 23, nil); err != nil {
		return nil, fmt.Errorf("cron: hour: %w", err)
	}
	if s.dom, err = parseField(fields[2], 1, 31, nil); err != nil {
		return nil, fmt.Errorf("cron: day-of-month: %w", err)
	}
	if s.month, err = parseField(fields[3], 1, 12, monthNames); err != nil {
		return nil, fmt.Errorf("cron: month: %w", err)
	}
	if s.dow, err = parseField(fields[4], 0, 7, dayNames); err != nil {
		return nil, fmt.Errorf("cron: day-of-week: %w", err)
	}
	if s.dow.has(7) {
		s.dow.add(0)
		s.dow &^= 1 << 7
	}
	s.domRestricted = fields[2] != "*"
	s.dowRestricted = fields[4] != "*"
	if len(s.minute.values(0, 59)) == 0 || len(s.hour.values(0, 23)) == 0 ||
		len(s.month.values(1, 12)) == 0 {
		return nil, fmt.Errorf("cron: expression never matches")
	}
	return &s, nil
}

func parseField(raw string, min, max int, names map[string]int) (set, error) {
	var out set
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return 0, fmt.Errorf("empty list element")
		}
		step := 1
		if i := strings.Index(part, "/"); i >= 0 {
			n, err := strconv.Atoi(part[i+1:])
			if err != nil || n <= 0 {
				return 0, fmt.Errorf("invalid step %q", part[i+1:])
			}
			step = n
			part = part[:i]
		}
		lo, hi := min, max
		switch {
		case part == "*":
			// full range
		case strings.Contains(part, "-"):
			bounds := strings.SplitN(part, "-", 2)
			var err error
			if lo, err = parseValue(bounds[0], names); err != nil {
				return 0, err
			}
			if hi, err = parseValue(bounds[1], names); err != nil {
				return 0, err
			}
			if lo > hi {
				return 0, fmt.Errorf("invalid range %q", part)
			}
		default:
			v, err := parseValue(part, names)
			if err != nil {
				return 0, err
			}
			lo, hi = v, v
			if step > 1 {
				hi = max
			}
		}
		if lo < min || hi > max {
			return 0, fmt.Errorf("value out of range [%d,%d]", min, max)
		}
		for v := lo; v <= hi; v += step {
			out.add(v)
		}
	}
	if out == 0 {
		return 0, fmt.Errorf("expression never matches")
	}
	return out, nil
}

func parseValue(raw string, names map[string]int) (int, error) {
	raw = strings.TrimSpace(raw)
	if names != nil {
		if v, ok := names[strings.ToLower(raw)]; ok {
			return v, nil
		}
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid value %q", raw)
	}
	return n, nil
}

// Next returns the first time strictly after `after` that matches the
// expression in loc. It returns the zero time when no match exists within five
// years (for example 30 February).
func (s *schedule) Next(after time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	start := after.In(loc)
	day := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	limit := day.AddDate(5, 0, 0)

	hours := s.hour.values(0, 23)
	minutes := s.minute.values(0, 59)
	first := true

	for d := day; d.Before(limit); d = d.AddDate(0, 0, 1) {
		if !s.dayMatches(d) {
			first = false
			continue
		}
		for _, h := range hours {
			for _, m := range minutes {
				cand := time.Date(d.Year(), d.Month(), d.Day(), h, m, 0, 0, loc)
				if first && !cand.After(start) {
					continue
				}
				return cand
			}
		}
		first = false
	}
	return time.Time{}
}

func (s *schedule) dayMatches(d time.Time) bool {
	if !s.month.has(int(d.Month())) {
		return false
	}
	domOK := s.dom.has(d.Day())
	dowOK := s.dow.has(int(d.Weekday()))
	if s.domRestricted && s.dowRestricted {
		return domOK || dowOK
	}
	if s.domRestricted {
		return domOK
	}
	if s.dowRestricted {
		return dowOK
	}
	return true
}

// Next parses expr and returns the next activation after `after` in loc.
func Next(expr string, after time.Time, loc *time.Location) (time.Time, error) {
	s, err := Parse(expr)
	if err != nil {
		return time.Time{}, err
	}
	next := s.Next(after, loc)
	if next.IsZero() {
		return time.Time{}, fmt.Errorf("cron: no activation found")
	}
	return next, nil
}
