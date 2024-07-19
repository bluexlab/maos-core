-- name: ApiTokenFindByID :one
SELECT t.id, a.id as agent_id, a.queue_id, t.permissions, t.expire_at
FROM api_tokens t
JOIN agents a ON t.agent_id = a.id
WHERE t.id = @id
LIMIT 1;
