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

  queue_id bigint NOT NULL REFERENCES queues(id),
  attempted_at bigint,
  created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  finalized_at bigint,

  priority smallint NOT NULL DEFAULT 1,

  -- types stored out-of-band
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  errors jsonb,
  result jsonb,
  metadata jsonb NOT NULL DEFAULT '{}' ::jsonb,
  tags varchar(255)[] NOT NULL DEFAULT '{}' ::varchar(255)[],
  attempted_by bigint[],

  CONSTRAINT finalized_or_finalized_at_null CHECK (
        (finalized_at IS NULL AND state NOT IN ('cancelled', 'completed', 'discarded')) OR
        (finalized_at IS NOT NULL AND state IN ('cancelled', 'completed', 'discarded'))
    ),
  CONSTRAINT priority_in_range CHECK (priority >= 1 AND priority <= 8)
);

CREATE INDEX invocations_state_and_finalized_at_index ON invocations USING btree(state, finalized_at) WHERE finalized_at IS NOT NULL;

CREATE INDEX invocations_prioritized_fetching_index ON invocations USING btree(state, queue_id, priority, id);

CREATE INDEX invocations_metadata_index ON invocations USING GIN(metadata);
