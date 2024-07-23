-- name: InvocationFindById :one
SELECT *
FROM invocations
WHERE id = @id;

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
RETURNING id;

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