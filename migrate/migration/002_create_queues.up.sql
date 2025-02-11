CREATE TABLE queues(
  id bigserial PRIMARY KEY NOT NULL,
  name text NOT NULL,
  created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  metadata jsonb NOT NULL DEFAULT '{}' ::jsonb,
  paused_at bigint,
  updated_at bigint,

  CONSTRAINT name_length CHECK (char_length(name) > 0 AND char_length(name) < 128)
);

CREATE UNIQUE INDEX queues_name ON queues USING btree(name);
