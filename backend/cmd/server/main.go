package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"livestream/config"
	"livestream/internal/app"
	"livestream/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	log := config.NewLogger(cfg)
	slog.SetDefault(log)
	gin.SetMode(gin.ReleaseMode)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg, log); err != nil {
		log.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	if err := config.WaitForDatabase(ctx, cfg, log); err != nil {
		return err
	}
	if err := config.RunMigrations(ctx, cfg, migrations.Files, log); err != nil {
		return err
	}
	db, err := config.OpenDatabase(cfg)
	if err != nil {
		return err
	}

	a, err := app.New(ctx, cfg, db, log)
	if err != nil {
		return err
	}
	if err := a.EnsureAdmin(ctx); err != nil {
		return err
	}
	a.StartBackground(ctx)

	servers := []*http.Server{
		{Addr: cfg.ServerAddr, Handler: a.Router, ReadHeaderTimeout: 10 * time.Second},
		{Addr: cfg.InternalAddr, Handler: a.Internal, ReadHeaderTimeout: 10 * time.Second},
	}
	errs := make(chan error, len(servers))
	for _, srv := range servers {
		go func() {
			log.Info("listening", "addr", srv.Addr)
			errs <- srv.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
	case err := <-errs:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, srv := range servers {
		srv.Shutdown(shutdownCtx)
	}
	return nil
}
