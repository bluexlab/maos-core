CREATE TABLE api_tokens(
  id text PRIMARY KEY NOT NULL,
  agent_id bigint NOT NULL REFERENCES agents(id),
  expire_at bigint NOT NULL,
  created_by text NOT NULL,
  created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  permissions varchar(255)[] NOT NULL DEFAULT '{}' ::varchar(255)[]
);
