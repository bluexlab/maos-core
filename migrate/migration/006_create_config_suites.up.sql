CREATE TABLE config_suites(
  id bigserial PRIMARY KEY,
  active boolean NOT NULL DEFAULT false,
  created_by text NOT NULL,
  created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_by text,
  updated_at bigint,
  deployed_at bigint
);

CREATE TABLE configs(
  id bigserial PRIMARY KEY,
  agent_id bigint NOT NULL REFERENCES agents(id),
  config_suite_id bigint REFERENCES config_suites(id),
  content jsonb NOT NULL,
  min_agent_version text,
  created_by text NOT NULL,
  created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  updated_by text,
  updated_at bigint
);
