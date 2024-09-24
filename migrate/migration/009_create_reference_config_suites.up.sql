CREATE TABLE reference_config_suites(
  id bigserial PRIMARY KEY,
  name text NOT NULL,
  config_suites jsonb NOT NULL DEFAULT '[]',
  created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_at bigint
);

CREATE UNIQUE INDEX ON reference_config_suites (name);