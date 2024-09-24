CREATE TYPE deployment_status AS ENUM (
    'draft',
    'reviewing',
    'approved',
    'deployed',
    'rejected',
    'retired',
    'cancelled'
);

CREATE TABLE deployments(
  id bigserial PRIMARY KEY,
  name text NOT NULL,
  status deployment_status NOT NULL DEFAULT 'draft',
  reviewers text[] NOT NULL DEFAULT '{}',
  config_suite_id bigint REFERENCES config_suites(id) ON DELETE RESTRICT ON UPDATE CASCADE,
  notes jsonb,
  created_by text NOT NULL,
  created_at bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW()),
  approved_by text,
  approved_at bigint,
  finished_by text,
  finished_at bigint
);
