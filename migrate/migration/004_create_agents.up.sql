CREATE TABLE agents(
  id bigserial PRIMARY KEY,
  name text NOT NULL,
  created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  metadata jsonb NOT NULL DEFAULT '{}' ::jsonb,
  updated_at bigint
);
