-- name: AgentInsert :one
INSERT INTO agents(
    name,
    queue_id,
    metadata
) VALUES (
    @name::text,
    @queue_id::bigint,
    coalesce(@metadata::jsonb, '{}')
) RETURNING *;
