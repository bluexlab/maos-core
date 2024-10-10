-- Add "deploying" status to deployment_status enum
ALTER TYPE deployment_status ADD VALUE 'deploying' AFTER 'approved';
ALTER TYPE deployment_status ADD VALUE 'failed' AFTER 'deployed';
