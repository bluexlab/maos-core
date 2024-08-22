CREATE TABLE settings(
  key text PRIMARY KEY,
  value jsonb NOT NULL DEFAULT '{}'
);
