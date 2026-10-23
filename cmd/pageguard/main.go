// Command pageguard runs the scan orchestrator and HTTP API.
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pageguard/pageguard/internal/api"
	"github.com/pageguard/pageguard/internal/domain"
	"github.com/pageguard/pageguard/internal/events"
	"github.com/pageguard/pageguard/internal/orchestrator"
	"github.com/pageguard/pageguard/internal/plugin"
	"github.com/pageguard/pageguard/internal/proxy"
	"github.com/pageguard/pageguard/internal/scanner"
	"github.com/pageguard/pageguard/internal/service"
	"github.com/pageguard/pageguard/internal/store/memory"
	"github.com/pageguard/pageguard/internal/store/postgres"
)

func main() {
	addr := flag.String("addr", env("PAGEGUARD_ADDR", ":8080"), "HTTP listen address")
	pluginsDir := flag.String("plugins", env("PAGEGUARD_PLUGINS_DIR", "plugins"), "external plugin directory")
	browserCmd := flag.String("browser-cmd", env("PAGEGUARD_BROWSER_CMD", ""), "headless browser worker command")
	nodeBin := flag.String("node", env("PAGEGUARD_NODE_BIN", "node"), "node interpreter for plugins")
	logLevel := flag.String("log", env("PAGEGUARD_LOG", "info"), "log level: debug|info|warn|error")
	databaseURL := flag.String("database-url", env("DATABASE_URL", ""), "PostgreSQL DSN (empty uses in-memory store)")
	migrationsDir := flag.String("migrations", env("PAGEGUARD_MIGRATIONS_DIR", "migrations"), "migrations directory (applied when --database-url is set)")
	migrate := flag.Bool("migrate", env("PAGEGUARD_MIGRATE", "true") == "true", "apply migrations on startup")
	proxies := flag.String("proxies", env("PAGEGUARD_PROXIES", ""), "comma-separated proxy specs, e.g. us=http://u:p@h:1,eu=http://h:2")
	captureHAR := flag.Bool("capture-har", env("PAGEGUARD_CAPTURE_HAR", "false") == "true", "capture a HAR archive for browser-rendered pages")
	captureDataLayer := flag.Bool("capture-datalayer", env("PAGEGUARD_CAPTURE_DATALAYER", "false") == "true", "capture client-side data layer payloads")
	flag.Parse()

	logger := newLogger(*logLevel)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, closeStore := openStore(ctx, *databaseURL, *migrationsDir, *migrate, logger)
	defer closeStore()
	hub := events.NewHub()

	registry := plugin.NewRegistry()
	for _, v := range plugin.BuiltinValidators() {
		registry.Register(v)
	}
	if loaded, err := plugin.LoadDir(registry, *pluginsDir, *nodeBin); err != nil {
		logger.Error("failed to load plugins", "dir", *pluginsDir, "err", err)
	} else if len(loaded) > 0 {
		logger.Info("loaded plugins", "count", len(loaded))
	}

	fetcher := scanner.NewHTTPFetcher()
	var renderer scanner.Renderer
	if strings.TrimSpace(*browserCmd) != "" {
		parts := strings.Fields(*browserCmd)
		renderer = scanner.NewBrowserRenderer(parts...)
		logger.Info("browser renderer enabled", "command", parts)
	}

	var mgrOpts []orchestrator.Option
	if specs := splitList(*proxies); len(specs) > 0 {
		rotator := proxy.New(proxy.ParseProxies(specs))
		fetcher.Transport = &proxy.RoundTripper{Rotator: rotator, MaxAttempts: 2}
		mgrOpts = append(mgrOpts, orchestrator.WithProxyRotator(rotator))
		logger.Info("proxy rotation enabled", "regions", rotator.Regions(), "count", len(specs))
	}
	if *captureHAR || *captureDataLayer {
		mgrOpts = append(mgrOpts, orchestrator.WithCapture(*captureHAR, *captureDataLayer))
	}

	scn := scanner.New(fetcher, renderer)
	host := plugin.NewHost(registry)
	manager := orchestrator.NewManager(store, hub, scn, host, mgrOpts...)
	svc := service.New(store, hub, manager)

	server := &http.Server{
		Addr:              *addr,
		Handler:           api.NewServer(svc, registry, logger).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("pageguard listening", "addr", *addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
	}
	if renderer != nil {
		_ = renderer.Close()
	}
}

// openStore returns the configured store and a cleanup function. With no
// DATABASE_URL the process uses the in-memory store (local/dev). With one it
// connects to PostgreSQL and, unless disabled, applies pending migrations.
func openStore(ctx context.Context, databaseURL, migrationsDir string, migrate bool, logger *slog.Logger) (domain.Store, func()) {
	if strings.TrimSpace(databaseURL) == "" {
		logger.Info("using in-memory store")
		return memory.New(), func() {}
	}
	st, err := postgres.New(ctx, databaseURL)
	if err != nil {
		logger.Error("postgres store unavailable", "err", err)
		os.Exit(1)
	}
	if migrate {
		if err := st.Migrate(ctx, migrationsDir); err != nil {
			logger.Error("migrations failed", "dir", migrationsDir, "err", err)
			st.Close()
			os.Exit(1)
		}
		logger.Info("migrations applied", "dir", migrationsDir)
	}
	logger.Info("using postgres store")
	return st, st.Close
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

// splitList splits a comma-separated list, trimming empty entries.
func splitList(s string) []string {
	parts := strings.Split(s, ",")
	out := parts[:0:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func newLogger(level string) *slog.Logger {
	var lv slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv}))
}
