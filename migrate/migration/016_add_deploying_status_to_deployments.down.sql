-- Remove "deploying" status from deployment_status enum
DELETE FROM pg_enum
WHERE (enumlabel = 'deploying' OR enumlabel = 'failed')
  AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'deployment_status' LIMIT 1);
