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

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/app"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/config"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/logging"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

func main() {
	rootDir := os.Getenv("ATLAS_CORE_ROOT_DIR")
	if rootDir == "" {
		rootDir, _ = os.Getwd()
	}
	cfg, err := config.Load(rootDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	runID := fmt.Sprintf("run-%d", time.Now().UTC().UnixNano())
	logger := logging.New(cfg.LogLevel, model.ServiceName, runID)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	atlas, err := app.Start(ctx, rootDir, logger)
	if err != nil {
		logger.Component("startup").ErrorContext(ctx, "startup failed", slog.String("error", err.Error()))
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer atlas.Close()
	server := &http.Server{Addr: fmt.Sprintf("%s:%d", atlas.Config.Host, atlas.Config.Port), Handler: atlas.Router, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	logger.Component("startup").InfoContext(ctx, "atlas core listening")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		atlas.Close()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
