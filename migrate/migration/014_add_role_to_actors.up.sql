CREATE TYPE actor_role AS ENUM ('agent', 'service', 'portal', 'user', 'other');
ALTER TABLE actors ADD COLUMN role actor_role NOT NULL DEFAULT 'agent';
