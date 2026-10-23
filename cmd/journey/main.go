// Command journey runs a Pageguard journey definition in a headless browser and
// prints a step-by-step report. It exits 0 when every step passes and 1 when a
// step fails, errors, or the run cannot start.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pageguard/pageguard/internal/journey"
)

func main() {
	path := flag.String("journey", "", "path to a journey definition (YAML or JSON)")
	format := flag.String("format", "text", "report format: text|json")
	screenshotDir := flag.String("screenshots", "", "directory for failure screenshots (default: OS temp)")
	browserCmd := flag.String("browser-cmd",
		env("PAGEGUARD_BROWSER_CMD", "node tools/browser-renderer/index.mjs"),
		"headless browser worker command")
	out := flag.String("out", "", "write the report to this file instead of stdout")
	flag.Parse()

	report, err := run(*path, *format, *screenshotDir, *browserCmd, *out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "journey:", err)
		os.Exit(1)
	}
	if !report.Passed {
		os.Exit(1)
	}
}

func run(path, format, screenshotDir, browserCmd, out string) (*journey.Report, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("--journey is required")
	}
	j, err := journey.LoadFile(path)
	if err != nil {
		return nil, err
	}

	engine := journey.NewEngine(journey.NewPlaywrightSessionFactory(strings.Fields(browserCmd)...))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	report, err := engine.Run(ctx, j, journey.RunOptions{ScreenshotDir: screenshotDir})
	if err != nil {
		return nil, err
	}

	var formatter journey.Formatter
	switch strings.ToLower(format) {
	case "json":
		formatter = journey.JSONFormatter{}
	case "text", "":
		formatter = journey.TextFormatter{}
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

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}
