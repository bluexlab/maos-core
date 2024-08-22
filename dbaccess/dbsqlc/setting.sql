-- name: SettingGetSystem :one
SELECT key, value FROM settings WHERE key = 'system' LIMIT 1;

-- name: SettingUpdateSystem :one
WITH existing_settings AS (
  SELECT value AS existing_value
  FROM settings
  WHERE key = 'system'
),
merged_settings AS (
  SELECT COALESCE(
    jsonb_object_agg(
      COALESCE(k, key),
      CASE WHEN v IS NOT NULL THEN v ELSE value END
    ),
    '{}'::jsonb
  ) AS merged_value
  FROM jsonb_each(COALESCE((SELECT existing_value FROM existing_settings), '{}'::jsonb)) AS e(key, value)
  FULL OUTER JOIN jsonb_each(@value::jsonb) AS n(k, v) ON e.key = n.k
)
INSERT INTO settings (key, value)
VALUES ('system', (SELECT merged_value FROM merged_settings))
ON CONFLICT (key)
DO UPDATE SET value = EXCLUDED.value
RETURNING *;
