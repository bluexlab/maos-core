-- name: ApiTokenListByPage :many
SELECT
  t.id,
  a.id as agent_id,
  a.name as agent_name,
  a.queue_id,
  t.permissions,
  t.created_at,
  t.expire_at,
  t.created_by,
  COUNT(*) OVER() AS total_count
FROM api_tokens t
JOIN agents a ON t.agent_id = a.id
WHERE (sqlc.narg('agent_id')::bigint IS NULL OR a.id = sqlc.narg('agent_id')::bigint)
ORDER BY t.created_at DESC, t.id
LIMIT sqlc.arg(page_size)::bigint
OFFSET sqlc.arg(page_size) * (sqlc.arg(page)::bigint - 1);

-- name: ApiTokenFindByID :one
SELECT t.id, a.id as agent_id, a.queue_id, t.permissions, t.expire_at, t.created_by
FROM api_tokens t
JOIN agents a ON t.agent_id = a.id
WHERE t.id = @id
LIMIT 1;

-- name: ApiTokenCount :one
SELECT COUNT(*) as count
FROM api_tokens;

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
    EXTRACT(EPOCH FROM NOW())
) RETURNING *;

-- name: ApiTokenDelete :exec
DELETE FROM api_tokens WHERE id = @id;

-- name: ApiTokenRotate :one
WITH new_token AS (
  INSERT INTO api_tokens (
    id,
    agent_id,
    expire_at,
    created_by,
    permissions,
    created_at
  ) VALUES (
    @id::text,
    @agent_id::bigint,
    @new_expire_at::bigint,
    @created_by::text,
    @permissions::varchar(255)[],
    EXTRACT(EPOCH FROM NOW())
  )
  RETURNING id
), update_existing AS (
  UPDATE api_tokens
  SET expire_at = EXTRACT(EPOCH FROM NOW() + INTERVAL '15 minutes')
  WHERE agent_id = @agent_id
    AND id != (SELECT id FROM new_token)
)
SELECT id FROM new_token;
