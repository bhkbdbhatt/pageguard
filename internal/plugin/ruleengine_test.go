package plugin

import (
	"context"
	"testing"

	"github.com/pageguard/pageguard/pkg/pluginsdk"
)

func TestDeclarativeValidatorPredicates(t *testing.T) {
	v := &DeclarativeValidator{
		PluginID: "test",
		Rules: []pluginsdk.RuleSpec{
			{ID: "status", Severity: "high", Title: "bad status",
				Condition: pluginsdk.Condition{All: []pluginsdk.Predicate{{Field: "statusCode", Op: "gte", Value: 400}}}},
			{ID: "no-title", Severity: "medium", Title: "no title",
				Condition: pluginsdk.Condition{None: []pluginsdk.Predicate{{Field: "title", Op: "exists"}}}},
			{ID: "no-csp", Severity: "low", Title: "no CSP",
				Condition: pluginsdk.Condition{None: []pluginsdk.Predicate{{Field: "header", Name: "Content-Security-Policy", Op: "exists"}}}},
			{ID: "insecure-cookie", Severity: "high", Title: "insecure cookie",
				Condition: pluginsdk.Condition{All: []pluginsdk.Predicate{{Field: "cookieSecure", Name: "sid", Op: "eq", Value: false}}}},
			{ID: "has-gtm", Severity: "info", Title: "gtm present",
				Condition: pluginsdk.Condition{All: []pluginsdk.Predicate{{Field: "tag", Name: "google-tag-manager", Op: "exists"}}}},
		},
	}

	obs := &pluginsdk.Observation{
		URL: "https://x/", FinalURL: "https://x/", StatusCode: 503,
		Headers: map[string]string{"Server": "nginx"},
		Cookies: []pluginsdk.Cookie{{Name: "sid", Secure: false}},
		Tags:    []pluginsdk.Tag{{ID: "google-tag-manager"}},
	}

	findings, err := v.Validate(context.Background(), obs, pluginsdk.Scope{})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	got := map[string]bool{}
	for _, f := range findings {
		got[f.RuleID] = true
	}
	for _, want := range []string{"status", "no-title", "no-csp", "insecure-cookie", "has-gtm"} {
		if !got[want] {
			t.Errorf("expected rule %q to fire; got %v", want, got)
		}
	}
}

func TestDeclarativeValidatorSecureCookieDoesNotFire(t *testing.T) {
	v := &DeclarativeValidator{PluginID: "test", Rules: []pluginsdk.RuleSpec{
		{ID: "insecure-cookie", Severity: "high",
			Condition: pluginsdk.Condition{All: []pluginsdk.Predicate{{Field: "cookieSecure", Name: "sid", Op: "eq", Value: false}}}},
	}}
	obs := &pluginsdk.Observation{Cookies: []pluginsdk.Cookie{{Name: "sid", Secure: true}}}
	findings, _ := v.Validate(context.Background(), obs, pluginsdk.Scope{})
	if len(findings) != 0 {
		t.Errorf("secure cookie should not fire, got %v", findings)
	}
}

func TestConditionAnyAndIn(t *testing.T) {
	v := &DeclarativeValidator{PluginID: "t", Rules: []pluginsdk.RuleSpec{
		{ID: "any", Severity: "low", Condition: pluginsdk.Condition{
			Any: []pluginsdk.Predicate{
				{Field: "title", Op: "contains", Value: "shop"},
				{Field: "statusCode", Op: "eq", Value: 200},
			},
		}},
		{ID: "in", Severity: "low", Condition: pluginsdk.Condition{
			All: []pluginsdk.Predicate{{Field: "title", Op: "in", Value: []any{"Home", "Shop"}}},
		}},
	}}
	obs := &pluginsdk.Observation{Title: "Home", StatusCode: 200}
	findings, _ := v.Validate(context.Background(), obs, pluginsdk.Scope{})
	got := map[string]bool{}
	for _, f := range findings {
		got[f.RuleID] = true
	}
	if !got["any"] || !got["in"] {
		t.Errorf("expected any+in, got %v", got)
	}
}
