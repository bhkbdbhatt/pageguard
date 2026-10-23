// Command privacy evaluates a Pageguard privacy policy against captured page
// observations and prints a per-regulation compliance report.
//
// Observations are supplied as a JSON file mapping each consent state to the
// page state observed under it (see examples/privacy/observations.json). The
// command exits 0 when every regulation passes and 1 otherwise.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pageguard/pageguard/internal/privacy"
)

func main() {
	policy := flag.String("policy", "", "path to a privacy policy (YAML or JSON)")
	observations := flag.String("observations", "", "path to a JSON file mapping consent states to observations")
	format := flag.String("format", "text", "report format: text|json")
	out := flag.String("out", "", "write the report to this file instead of stdout")
	flag.Parse()

	report, err := run(*policy, *observations, *format, *out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "privacy:", err)
		os.Exit(1)
	}
	if !report.Passed {
		os.Exit(1)
	}
}

func run(policyPath, obsPath, format, out string) (*privacy.Report, error) {
	if strings.TrimSpace(policyPath) == "" {
		return nil, fmt.Errorf("--policy is required")
	}
	cfg, err := privacy.LoadFile(policyPath)
	if err != nil {
		return nil, err
	}
	rescanner, err := loadRescanner(obsPath)
	if err != nil {
		return nil, err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	report, err := privacy.Evaluate(ctx, cfg, rescanner)
	if err != nil {
		return nil, err
	}

	var formatter privacy.Formatter
	switch strings.ToLower(format) {
	case "json":
		formatter = privacy.JSONFormatter{}
	case "text", "":
		formatter = privacy.TextFormatter{}
	default:
		return nil, fmt.Errorf("unknown format %q", format)
	}

	w := os.Stdout
	if out != "" {
		f, err := os.Create(out)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		w = f
	}
	if err := formatter.Format(w, report); err != nil {
		return nil, err
	}
	return report, nil
}

func loadRescanner(path string) (privacy.Rescanner, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("--observations is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]privacy.Observation
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse observations: %w", err)
	}
	byState := make(map[privacy.ConsentState]*privacy.Observation, len(raw))
	for k, v := range raw {
		obs := v
		byState[privacy.ConsentState(k)] = &obs
	}
	return &privacy.FakeRescanner{ByState: byState}, nil
}
