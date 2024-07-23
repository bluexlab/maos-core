CREATE TABLE agents(
  id bigserial PRIMARY KEY,
  name text NOT NULL,
  queue_id bigint NOT NULL REFERENCES queues(id),
  created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  metadata jsonb NOT NULL DEFAULT '{}' ::jsonb,
  updated_at bigint
);

CREATE UNIQUE INDEX agents_name ON agents USING btree(name);
