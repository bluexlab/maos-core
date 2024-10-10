ALTER TABLE deployments DROP COLUMN IF EXISTS migration_logs;
ALTER TABLE deployments DROP COLUMN IF EXISTS deploying_at;
ALTER TABLE deployments DROP COLUMN IF EXISTS deployed_at;
ALTER TABLE deployments DROP COLUMN IF EXISTS last_error;
ALTER TABLE actors DROP COLUMN IF EXISTS migratable;
