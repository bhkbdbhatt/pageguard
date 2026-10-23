package plugin

import "github.com/pageguard/pageguard/pkg/pluginsdk"

// BuiltinValidators returns the validators that ship with Pageguard.
func BuiltinValidators() []pluginsdk.Validator {
	return []pluginsdk.Validator{
		&DeclarativeValidator{PluginID: "core", Rules: CoreRules()},
	}
}

// CoreRules is the default declarative rule set applied to every scan.
func CoreRules() []pluginsdk.RuleSpec {
	return []pluginsdk.RuleSpec{
		{
			ID: "core/http-error-status", Severity: "high", Category: "availability",
			Title:       "Page returned an error status",
			Description: "The page responded with an HTTP status of 400 or greater.",
			Remediation: "Fix the server response or remove links leading to this URL.",
			Condition: pluginsdk.Condition{
				All: []pluginsdk.Predicate{{Field: "statusCode", Op: "gte", Value: 400}},
			},
		},
		{
			ID: "core/title-missing", Severity: "medium", Category: "seo",
			Title:       "Document has no <title>",
			Description: "A missing title harms accessibility and search indexing.",
			Remediation: "Add a unique, descriptive <title> element.",
			Condition: pluginsdk.Condition{
				None: []pluginsdk.Predicate{{Field: "title", Op: "exists"}},
			},
		},
		{
			ID: "core/https-required", Severity: "medium", Category: "security",
			Title:       "Page is not served over HTTPS",
			Description: "The final URL does not use the https scheme.",
			Remediation: "Serve the page over HTTPS and redirect HTTP requests.",
			Condition: pluginsdk.Condition{
				None: []pluginsdk.Predicate{{Field: "finalUrl", Op: "contains", Value: "https://"}},
			},
		},
		{
			ID: "core/meta-description-missing", Severity: "low", Category: "seo",
			Title:       "Document has no meta description",
			Description: "No <meta name=\"description\"> tag was found.",
			Remediation: "Add a concise meta description.",
			Condition: pluginsdk.Condition{
				None: []pluginsdk.Predicate{
					{Field: "html", Op: "contains", Value: `name="description"`},
					{Field: "html", Op: "contains", Value: `name='description'`},
				},
			},
		},
		{
			ID: "core/large-html-payload", Severity: "low", Category: "performance",
			Title:       "HTML document is very large",
			Description: "The HTML payload exceeds 500 KiB and may slow rendering.",
			Remediation: "Reduce inline data, split bundles, or stream the document.",
			Condition: pluginsdk.Condition{
				All: []pluginsdk.Predicate{{Field: "htmlLength", Op: "gt", Value: 512000}},
			},
		},
	}
}
