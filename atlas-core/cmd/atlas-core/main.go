package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/app"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/config"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/logging"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

func main() {
	rootDir, err := filepath.Abs(filepath.Join(filepath.Dir(os.Args[0]), "..", ".."))
	if err != nil {
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
		logger.Component("startup").ErrorContext(ctx, "startup failed")
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
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
