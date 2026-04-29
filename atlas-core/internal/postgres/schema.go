package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// schemaStatements is DDL applied in order on startup (one Exec per statement).
var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS entities (
  entity_id text PRIMARY KEY CHECK (length(entity_id) BETWEEN 1 AND 50),
  type text NOT NULL CHECK (type IN ('asset', 'track', 'geofeature')),
  subtype text,
  alias text,
  json jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
)`,
	`CREATE INDEX IF NOT EXISTS entities_type_idx ON entities(type)`,
	`CREATE INDEX IF NOT EXISTS entities_updated_at_idx ON entities(updated_at DESC, entity_id ASC)`,
	`CREATE TABLE IF NOT EXISTS observations (
  observation_id text PRIMARY KEY CHECK (length(observation_id) BETWEEN 1 AND 50),
  source_asset_id text NOT NULL REFERENCES entities(entity_id),
  json jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
)`,
	`CREATE INDEX IF NOT EXISTS observations_source_asset_idx ON observations(source_asset_id)`,
	`CREATE INDEX IF NOT EXISTS observations_updated_at_idx ON observations(updated_at DESC, observation_id ASC)`,
	`CREATE TABLE IF NOT EXISTS objects (
  object_id text PRIMARY KEY CHECK (length(object_id) BETWEEN 1 AND 50),
  type text NOT NULL,
  owner_type text NOT NULL CHECK (owner_type IN ('entity', 'observation', 'task', 'system')),
  owner_id text NOT NULL,
  json jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
)`,
	`CREATE INDEX IF NOT EXISTS objects_owner_idx ON objects(owner_type, owner_id)`,
	`CREATE INDEX IF NOT EXISTS objects_type_idx ON objects(type)`,
	`CREATE INDEX IF NOT EXISTS objects_updated_at_idx ON objects(updated_at DESC, object_id ASC)`,
	`CREATE TABLE IF NOT EXISTS tasks (
  task_id text PRIMARY KEY CHECK (length(task_id) BETWEEN 1 AND 50),
  status text NOT NULL CHECK (status IN ('pending', 'acknowledged', 'completed', 'failed')),
  asset_id text NOT NULL REFERENCES entities(entity_id),
  command_catalog_object_id text NOT NULL REFERENCES objects(object_id),
  json jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
)`,
	`CREATE INDEX IF NOT EXISTS tasks_asset_idx ON tasks(asset_id)`,
	`CREATE INDEX IF NOT EXISTS tasks_status_idx ON tasks(status)`,
	`CREATE INDEX IF NOT EXISTS tasks_asset_status_updated_idx ON tasks(asset_id, status, updated_at DESC, task_id ASC)`,
	`CREATE INDEX IF NOT EXISTS tasks_updated_at_idx ON tasks(updated_at DESC, task_id ASC)`,
	`CREATE TABLE IF NOT EXISTS object_files (
  file_id text NOT NULL CHECK (length(file_id) BETWEEN 1 AND 50),
  object_id text NOT NULL REFERENCES objects(object_id) ON DELETE CASCADE,
  path text NOT NULL UNIQUE,
  content_type text NOT NULL,
  size_bytes bigint NOT NULL CHECK (size_bytes >= 0),
  usage_hint text,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
)`,
	`ALTER TABLE object_files DROP CONSTRAINT IF EXISTS object_files_pkey`,
	`DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'object_files_pkey' AND conrelid = 'object_files'::regclass) THEN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'object_files_pkey' AND conrelid = 'object_files'::regclass AND conkey = (SELECT array_agg(attnum) FROM pg_attribute WHERE attrelid = 'object_files'::regclass AND attname IN ('object_id', 'file_id'))) THEN
      ALTER TABLE object_files DROP CONSTRAINT object_files_pkey;
      ALTER TABLE object_files ADD CONSTRAINT object_files_pkey PRIMARY KEY (object_id, file_id);
    END IF;
  ELSE
    ALTER TABLE object_files ADD CONSTRAINT object_files_pkey PRIMARY KEY (object_id, file_id);
  END IF;
END $$`,
	`CREATE INDEX IF NOT EXISTS object_files_object_idx ON object_files(object_id)`,
	`CREATE INDEX IF NOT EXISTS object_files_updated_at_idx ON object_files(updated_at DESC, object_id ASC, file_id ASC)`,
}

func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for i, stmt := range schemaStatements {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("schema statement %d of %d: %w", i+1, len(schemaStatements), err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("schema commit: %w", err)
	}
	return nil
}
