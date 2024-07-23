-- name: ApiTokenListByPage :many
SELECT
  t.id,
  a.id as agent_id,
  a.queue_id,
  t.permissions,
  t.created_at,
  t.expire_at,
  t.created_by,
  COUNT(*) OVER() AS total_count
FROM api_tokens t
JOIN agents a ON t.agent_id = a.id
ORDER BY t.created_at DESC, t.id
LIMIT sqlc.arg(page_size)::bigint
OFFSET sqlc.arg(page_size) * (sqlc.arg(page)::bigint - 1);

-- name: ApiTokenFindByID :one
SELECT t.id, a.id as agent_id, a.queue_id, t.permissions, t.expire_at, t.created_by
FROM api_tokens t
JOIN agents a ON t.agent_id = a.id
WHERE t.id = @id
LIMIT 1;

-- name: ApiTokenInsert :one
INSERT INTO api_tokens(
    id,
    agent_id,
    expire_at,
    created_by,
    permissions,
    created_at
) VALUES (
    @id::text,
    @agent_id::bigint,
    @expire_at::bigint,
    @created_by::text,
    @permissions::varchar(255)[],
    coalesce(@created_at::bigint, EXTRACT(EPOCH FROM NOW()))
) RETURNING *;
