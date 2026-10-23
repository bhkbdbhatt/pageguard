package orchestrator

import (
	"regexp"
	"strings"
	"sync"
)

var globCache sync.Map // pattern string -> *regexp.Regexp

// globMatch reports whether target matches a shell-style glob (* and ?).
// Matching is case-insensitive and runs against the full URL string, so
// patterns may target the scheme, host, path or a substring.
func globMatch(pattern, target string) bool {
	return globToRegex(pattern).MatchString(target)
}

// matchAny reports whether target matches any pattern.
func matchAny(patterns []string, target string) bool {
	for _, p := range patterns {
		if globMatch(p, target) {
			return true
		}
	}
	return false
}

func globToRegex(pattern string) *regexp.Regexp {
	if v, ok := globCache.Load(pattern); ok {
		return v.(*regexp.Regexp)
	}
	var b strings.Builder
	b.WriteString("(?i)^")
	for _, r := range pattern {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteByte('.')
		case '.', '+', '(', ')', '[', ']', '{', '}', '^', '$', '|', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteString("$")
	re := regexp.MustCompile(b.String())
	globCache.Store(pattern, re)
	return re
}
