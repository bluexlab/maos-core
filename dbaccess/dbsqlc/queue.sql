-- name: QueueInsert :one
INSERT INTO queues(
    name,
    metadata
) VALUES (
    @name::text,
    coalesce(@metadata::jsonb, '{}')
) RETURNING *;

-- name: QueueFindById :one
SELECT *
FROM queues
WHERE id = @id;
