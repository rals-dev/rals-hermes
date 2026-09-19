// Command bff is the backend-for-frontend of the Hermes monitoring dashboard.
//
// It fans out to the API server of every configured Hermes profile, aggregates
// the results, serves a read-only JSON API plus the embedded web UI, and never
// starts an agent run. See docs/prd.md for the full contract.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/rals-dev/rals-hermes/internal/activity"
	"github.com/rals-dev/rals-hermes/internal/api"
	"github.com/rals-dev/rals-hermes/internal/config"
	"github.com/rals-dev/rals-hermes/internal/hermes"
	"github.com/rals-dev/rals-hermes/internal/observ"
	"github.com/rals-dev/rals-hermes/internal/server"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(); err != nil {
		// Config errors are the common case here; they never contain secret
		// values (see config.Secret and MissingEnvError).
		fmt.Fprintln(os.Stderr, "hermes-bff:", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "config.yaml", "path to the YAML configuration file")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return nil
	}

	logger := newLogger(os.Getenv("LOG_LEVEL"))
	slog.SetDefault(logger)

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	logger.Info("configuration loaded",
		"config", *configPath,
		"profiles", profileNames(cfg),
		"upstream_timeout", cfg.Upstream.Timeout.String(),
		"cache_ttl", cfg.Cache.TTL.String(),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ln, err := net.Listen("tcp", cfg.Server.Addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", cfg.Server.Addr, err)
	}

	metrics := observ.New()
	clients := make([]*hermes.Client, 0, len(cfg.Upstream.Profiles))
	for _, p := range cfg.Upstream.Profiles {
		clients = append(clients, hermes.NewClient(p, hermes.Options{
			Timeout:  cfg.Upstream.Timeout,
			Logger:   logger,
			Observer: metrics,
		}))
	}
	handler := api.NewHandler(api.Deps{
		Logger:     logger,
		Version:    version,
		Profiles:   clients,
		CacheTTL:   cfg.Cache.TTL,
		AuthKey:    cfg.Auth.Key,
		SessionTTL: cfg.Auth.SessionTTL,
		Metrics:    metrics,
		Activity: activity.Config{
			PollInterval:  cfg.Activity.PollInterval,
			Window:        cfg.Activity.Window,
			MaxSessions:   cfg.Activity.MaxSessionsPerProfile,
			IdleStopAfter: cfg.Activity.IdleStopAfter,
		},
	})
	err = server.Run(ctx, ln, handler, server.Options{
		Logger:          logger,
		ReadTimeout:     cfg.Server.ReadTimeout,
		ShutdownTimeout: cfg.Server.ShutdownTimeout,
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	logger.Info("bye")
	return nil
}

// newLogger builds the JSON logger every line goes through. Loki/Alloy
// ingest stdout as-is, so the format is fixed here and nowhere else.
func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

func profileNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Upstream.Profiles))
	for _, p := range cfg.Upstream.Profiles {
		names = append(names, p.Name)
	}
	return names
}
