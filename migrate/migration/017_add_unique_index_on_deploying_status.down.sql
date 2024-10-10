-- Remove the unique index on deployments table for 'deploying' status
DROP INDEX IF EXISTS deployments_unique_deploying;
DROP INDEX IF EXISTS deployments_unique_deployed;
DROP INDEX IF EXISTS config_suites_unique_active;
