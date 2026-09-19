// Package server runs an http.Server with timeouts and graceful shutdown tied
// to a context.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Options tunes the server. Zero durations fall back to safe defaults.
type Options struct {
	Logger          *slog.Logger
	ReadTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// Run serves h on ln until ctx is cancelled, then drains in-flight requests
// for at most ShutdownTimeout. It returns nil after a clean shutdown and the
// listener error otherwise.
func Run(ctx context.Context, ln net.Listener, h http.Handler, o Options) error {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.ReadTimeout <= 0 {
		o.ReadTimeout = 10 * time.Second
	}
	if o.ShutdownTimeout <= 0 {
		o.ShutdownTimeout = 10 * time.Second
	}

	srv := &http.Server{
		Handler:           h,
		ReadTimeout:       o.ReadTimeout,
		ReadHeaderTimeout: o.ReadTimeout,
		// No WriteTimeout: SSE responses are long-lived by design. Per-request
		// deadlines are enforced by handlers through the request context.
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	o.Logger.Info("server listening", "addr", ln.Addr().String())

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	o.Logger.Info("server shutting down", "timeout", o.ShutdownTimeout.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), o.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		o.Logger.Warn("graceful shutdown incomplete, closing", "err", err)
		_ = srv.Close()
		return err
	}
	<-errc // Serve has returned ErrServerClosed
	return nil
}
