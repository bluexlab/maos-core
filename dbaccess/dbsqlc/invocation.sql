-- name: InvocationFindById :one
SELECT *
FROM invocations
WHERE id = @id::bigint;

-- name: InvocationInsert :one
WITH agent_queue AS (
	SELECT queue_id
	FROM agents
	WHERE name = @agent_name::text
)
INSERT INTO invocations(
	state,
	queue_id,
	created_at,
	finalized_at,
	priority,
	payload,
	metadata,
	tags
)
SELECT
	@state::invocation_state,
	agent_queue.queue_id,
	coalesce(@created_at::bigint, EXTRACT(EPOCH FROM NOW())),
	@finalized_at,
	@priority::smallint,
	@payload::jsonb,
	coalesce(@metadata::jsonb, '{}'),
	coalesce(@tags::varchar(255)[], '{}')
FROM agent_queue
RETURNING id, queue_id;

-- name: InvocationGetAvailable :many
WITH locked_invocations AS (
	SELECT
		*
	FROM
		invocations
	WHERE
		state = 'available'::invocation_state
		AND queue_id = @queue_id::bigint
	ORDER BY
		priority ASC,
		id ASC
	LIMIT @max::integer
	FOR UPDATE
	SKIP LOCKED
)
UPDATE
	invocations
SET
	state = 'running'::invocation_state,
	attempted_at = EXTRACT(EPOCH FROM NOW()),
	attempted_by = array_append(invocations.attempted_by, @attempted_by::bigint)
FROM
	locked_invocations
WHERE
	invocations.id = locked_invocations.id
RETURNING
	invocations.*;


-- name: InvocationSetCompleteIfRunning :one
WITH invocation_to_update AS (
	SELECT invocations.id
	FROM invocations
	WHERE invocations.id = @id::bigint
		AND invocations.state = 'running'::invocation_state
		AND (
            array_length(attempted_by, 1) > 0
            AND attempted_by[array_length(attempted_by, 1)] = @finalizer_id::bigint
        )
	FOR UPDATE
),
updated_invocation AS (
	UPDATE invocations
	SET
		finalized_at = @finalized_at::bigint,
		result = @result::jsonb,
		state = 'completed'
	FROM invocation_to_update
	WHERE invocations.id = invocation_to_update.id
	RETURNING invocations.*
)
SELECT id, state, finalized_at
FROM updated_invocation;
