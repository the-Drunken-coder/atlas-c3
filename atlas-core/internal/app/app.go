package app

import (
	"context"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/catalog"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/config"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/events"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/httpapi"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/logging"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/objectfiles"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/postgres"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/service"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/sightingcatalog"
)

type App struct {
	Config          config.Config
	Logger          *logging.Logger
	Pool            *pgxpool.Pool
	Files           *objectfiles.Store
	Events          *events.Hub
	Services        *service.Services
	CommandCatalog  *catalog.Active
	SightingCatalog *sightingcatalog.Active
	StartedAt       time.Time
	Router          http.Handler
}

func Start(ctx context.Context, rootDir string, logger *logging.Logger) (*App, error) {
	if logger == nil {
		logger = logging.New("info", "atlas-core", "local")
	}
	cfg, err := config.Load(rootDir)
	if err != nil {
		return nil, err
	}
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	closePoolOnError := true
	defer func() {
		if closePoolOnError {
			pool.Close()
		}
	}()
	if err := postgres.EnsureSchema(ctx, pool); err != nil {
		return nil, err
	}
	files, err := objectfiles.New(cfg.ObjectStorageRoot)
	if err != nil {
		return nil, err
	}
	if err := files.Verify(); err != nil {
		return nil, err
	}
	commandCatalog := &catalog.Active{}
	sightingCat := &sightingcatalog.Active{}
	stores := postgres.NewStore(pool, files, cfg.MaxUploadBytes, logger.Component("postgres"))
	if loaded, err := catalog.Load(cfg.CommandCatalogPath); err != nil {
		logger.Component("app").WarnContext(ctx, "failed to load command catalog",
			slog.String("path", cfg.CommandCatalogPath),
			slog.String("error", err.Error()),
		)
	} else if materialized, err := catalog.Materialize(ctx, stores, loaded); err != nil {
		logger.Component("app").WarnContext(ctx, "failed to materialize command catalog",
			slog.String("path", cfg.CommandCatalogPath),
			slog.String("error", err.Error()),
		)
	} else {
		commandCatalog.Set(materialized)
	}
	if loaded, err := sightingcatalog.Load(cfg.SightingCatalogPath); err != nil {
		logger.Component("app").WarnContext(ctx, "failed to load sighting catalog",
			slog.String("path", cfg.SightingCatalogPath),
			slog.String("error", err.Error()),
		)
	} else {
		sightingCat.Set(loaded)
	}
	hub := events.NewHub()
	services := service.New(stores, commandCatalog, sightingCat, hub)
	startedAt := time.Now().UTC()
	atlas := &App{Config: cfg, Logger: logger, Pool: pool, Files: files, Events: hub, Services: services, CommandCatalog: commandCatalog, SightingCatalog: sightingCat, StartedAt: startedAt}
	atlas.Router = httpapi.NewRouter(httpapi.Dependencies{
		AllowedOrigins: cfg.AllowedOrigins,
		StartedAt:      startedAt,
		MaxUploadBytes: cfg.MaxUploadBytes,
		Services:       services,
		Events:         hub,
		CommandCatalog: commandCatalog,
		ObjectStore:    stores,
		Files:          files,
		Logger:         logger,
		Readiness:      func(ctx context.Context) (model.ReadinessResponse, int) { return atlas.Readiness(ctx) },
		Descriptor:     atlas.ServiceDescriptor,
	})
	closePoolOnError = false
	return atlas, nil
}

func RootDir() string {
	return filepath.Clean(".")
}

func (a *App) Close() {
	if a.Pool != nil {
		a.Pool.Close()
	}
}

func (a *App) ServiceDescriptor() (model.ServiceDescriptor, error) {
	active, ok := a.CommandCatalog.Get()
	if !ok {
		return model.ServiceDescriptor{}, model.CatalogUnavailable("active command catalog is unavailable", nil)
	}
	return model.ServiceDescriptor{Service: model.ServiceName, Status: "ok", Version: a.Config.Version, StartedAt: a.StartedAt, ActiveCommandCatalogObjectID: active.ObjectID, Links: map[string]string{"health": "/health", "readiness": "/readiness", "full_query": "/queries/full", "stream": "/stream/changes"}}, nil
}

func (a *App) Health() model.HealthResponse {
	return model.HealthResponse{Status: "ok", Service: model.ServiceName, Timestamp: time.Now().UTC()}
}

func (a *App) Readiness(ctx context.Context) (model.ReadinessResponse, int) {
	dependencies := map[string]model.DependencyStatus{"object_storage": a.Files.Status()}
	if err := a.Pool.Ping(ctx); err != nil {
		dependencies["postgres"] = model.DependencyStatus{Status: "error", Message: err.Error()}
	} else {
		dependencies["postgres"] = model.DependencyStatus{Status: "ready"}
	}
	if _, ok := a.CommandCatalog.Get(); ok {
		dependencies["command_catalog"] = model.DependencyStatus{Status: "ready"}
	} else {
		dependencies["command_catalog"] = model.DependencyStatus{Status: "error", Message: "catalog unavailable"}
	}
	statusCode := http.StatusOK
	overall := "ready"
	if dependencies["postgres"].Status != "ready" || dependencies["object_storage"].Status != "ready" {
		statusCode = http.StatusServiceUnavailable
		overall = "not_ready"
	}
	if dependencies["command_catalog"].Status != "ready" {
		statusCode = http.StatusServiceUnavailable
		overall = "not_ready"
	}
	return model.ReadinessResponse{Status: overall, Timestamp: time.Now().UTC(), Dependencies: dependencies}, statusCode
}
