-- name: QueueInsert :one
INSERT INTO queues(
    name,
    metadata
) VALUES (
    @name::text,
    coalesce(@metadata::jsonb, '{}')
) RETURNING *;
