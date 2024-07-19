CREATE TABLE queues(
  name text PRIMARY KEY NOT NULL,
  created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  metadata jsonb NOT NULL DEFAULT '{}' ::jsonb,
  paused_at bigint,
  updated_at bigint NOT NULL,

  CONSTRAINT name_length CHECK (char_length(name) > 0 AND char_length(name) < 128)
);
