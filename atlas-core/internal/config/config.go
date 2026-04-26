package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

type Config struct {
	Host                string
	Port                int
	DatabaseURL         string
	ObjectStorageRoot   string
	CommandCatalogPath  string
	SightingCatalogPath string
	AllowedOrigins      []string
	LogLevel            string
	Version             string
	MaxUploadBytes      int64
	DataFusionStack     string
}

func Load(rootDir string) (Config, error) {
	cfg := Config{
		Host:                envOrDefault("ATLAS_CORE_HOST", "0.0.0.0"),
		Port:                envIntOrDefault("ATLAS_CORE_PORT", 8080),
		DatabaseURL:         envOrDefault("ATLAS_CORE_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/atlas_core?sslmode=disable"),
		ObjectStorageRoot:   envOrDefault("ATLAS_CORE_OBJECT_STORAGE_ROOT", filepath.Join(rootDir, "var", "object-storage")),
		CommandCatalogPath:  envOrDefault("ATLAS_CORE_COMMAND_CATALOG_PATH", filepath.Join(rootDir, "command-catalog", "catalog.json")),
		SightingCatalogPath: envOrDefault("ATLAS_CORE_SIGHTING_CATALOG_PATH", filepath.Join(rootDir, "sighting-catalog", "catalog.json")),
		AllowedOrigins:      model.CleanOriginList(envOrDefault("ATLAS_CORE_ALLOWED_ORIGINS", "http://localhost:5173")),
		LogLevel:            strings.ToLower(envOrDefault("ATLAS_CORE_LOG_LEVEL", "info")),
		Version:             envOrDefault("ATLAS_CORE_VERSION", "dev"),
		MaxUploadBytes:      envInt64OrDefault("ATLAS_CORE_MAX_UPLOAD_BYTES", 16*1024*1024),
		DataFusionStack:     envOrDefault("ATLAS_DATA_FUSION_STACK", ""),
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return Config{}, fmt.Errorf("ATLAS_CORE_PORT must be between 1 and 65535")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("ATLAS_CORE_DATABASE_URL is required")
	}
	if err := os.MkdirAll(cfg.ObjectStorageRoot, 0o755); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envIntOrDefault(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envInt64OrDefault(name string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return value
}

func (c Config) GetAllowedOrigins() []string {
	return append([]string(nil), c.AllowedOrigins...)
}
