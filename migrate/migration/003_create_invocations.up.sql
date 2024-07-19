CREATE TYPE invocation_state AS ENUM(
  'available',
  'cancelled',
  'completed',
  'discarded',
  'running'
);

CREATE TABLE invocations(
  id bigserial PRIMARY KEY,

  state invocation_state NOT NULL DEFAULT 'available' ::invocation_state,

  created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  finalized_at bigint,

  priority smallint NOT NULL DEFAULT 1,

  -- types stored out-of-band
  name text NOT NULL,
  args jsonb NOT NULL DEFAULT '{}'::jsonb,
  errors jsonb[],
  metadata jsonb NOT NULL DEFAULT '{}' ::jsonb,
  queue_id bigint NOT NULL REFERENCES queues(id),
  tags varchar(255)[] NOT NULL DEFAULT '{}' ::varchar(255)[],

  CONSTRAINT finalized_or_finalized_at_null CHECK (
        (finalized_at IS NULL AND state NOT IN ('cancelled', 'completed', 'discarded')) OR
        (finalized_at IS NOT NULL AND state IN ('cancelled', 'completed', 'discarded'))
    ),
  CONSTRAINT priority_in_range CHECK (priority >= 1 AND priority <= 8),
  CONSTRAINT name_length CHECK (char_length(name) > 0 AND char_length(name) < 128)
);

CREATE INDEX invocations_name ON invocations USING btree(name);

CREATE INDEX invocations_state_and_finalized_at_index ON invocations USING btree(state, finalized_at) WHERE finalized_at IS NOT NULL;

CREATE INDEX invocations_prioritized_fetching_index ON invocations USING btree(state, queue_id, priority, id);

CREATE INDEX invocations_args_index ON invocations USING GIN(args);

CREATE INDEX invocations_metadata_index ON invocations USING GIN(metadata);
