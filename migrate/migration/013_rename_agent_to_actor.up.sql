-- Rename the table
ALTER TABLE agents RENAME TO actors;

-- Rename the sequence
ALTER SEQUENCE agents_id_seq RENAME TO actors_id_seq;

-- Rename foreign key constraints
ALTER TABLE configs RENAME CONSTRAINT configs_agent_id_fkey TO configs_actor_id_fkey;
ALTER TABLE api_tokens RENAME CONSTRAINT api_tokens_agent_id_fkey TO api_tokens_actor_id_fkey;

-- Rename indexes
ALTER INDEX agents_pkey RENAME TO actors_pkey;
ALTER INDEX agents_name RENAME TO actors_name;

-- Rename columns in related tables
ALTER TABLE configs RENAME COLUMN agent_id TO actor_id;
ALTER TABLE configs RENAME COLUMN min_agent_version TO min_actor_version;
ALTER TABLE api_tokens RENAME COLUMN agent_id TO actor_id;
