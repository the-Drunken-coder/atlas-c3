package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/app"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/config"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/logging"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

// resolveRootDir returns the deployment root: the directory used to resolve
// default relative paths in config (see config.Load in internal/config): var/object-storage,
// command-catalog/catalog.json, sighting-catalog/catalog.json, unless those are
// overridden with ATLAS_CORE_OBJECT_STORAGE_ROOT, ATLAS_CORE_COMMAND_CATALOG_PATH,
// and ATLAS_CORE_SIGHTING_CATALOG_PATH.
//
// Resolution order:
//  1. If ATLAS_CORE_ROOT_DIR is set, it is used (trimmed, then made absolute). Use
//     this for go run (the binary lives in a temp dir, not your checkout),
//     custom install layouts, or when the binary is not colocated with data.
//  2. Otherwise the parent directory of the running executable is used, after
//     EvalSymlinks. That matches: go build -o ./atlas-core ./cmd/atlas-core and
//     running ./atlas-core from the built tree; or a Docker image with the
//     binary at e.g. /app/atlas-core (root becomes /app). Production images
//     often set absolute catalog/storage env vars; root still anchors any
//     remaining defaults.
func resolveRootDir() (string, error) {
	if d := strings.TrimSpace(os.Getenv("ATLAS_CORE_ROOT_DIR")); d != "" {
		abs, err := filepath.Abs(d)
		if err != nil {
			return "", fmt.Errorf("ATLAS_CORE_ROOT_DIR: %w", err)
		}
		return abs, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("ATLAS_CORE_ROOT_DIR is not set and could not read executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("ATLAS_CORE_ROOT_DIR is not set and could not resolve executable path: %w", err)
	}
	return filepath.Clean(filepath.Dir(exe)), nil
}

func main() {
	rootDir, err := resolveRootDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", atlas.Config.Host, atlas.Config.Port),
		Handler:           atlas.Router,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
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
