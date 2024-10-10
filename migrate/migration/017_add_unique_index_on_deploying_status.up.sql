CREATE UNIQUE INDEX deployments_unique_deploying ON deployments (status) WHERE status = 'deploying';
CREATE UNIQUE INDEX deployments_unique_deployed ON deployments (status) WHERE status = 'deployed';
CREATE UNIQUE INDEX config_suites_unique_active ON config_suites (active) WHERE active = true;
