ALTER TABLE deployments
  ADD COLUMN IF NOT EXISTS migration_logs jsonb,
  ADD COLUMN IF NOT EXISTS last_error text,
  ADD COLUMN IF NOT EXISTS deploying_at bigint,
  ADD COLUMN IF NOT EXISTS deployed_at bigint;
ALTER TABLE actors
  ADD COLUMN IF NOT EXISTS migratable boolean NOT NULL DEFAULT false;
