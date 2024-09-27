-- Rename columns in related tables back to original names
ALTER TABLE configs RENAME COLUMN actor_id TO agent_id;
ALTER TABLE configs RENAME COLUMN min_actor_version TO min_agent_version;
ALTER TABLE api_tokens RENAME COLUMN actor_id TO agent_id;

-- Update the type of related columns if necessary (reverting any type changes)
ALTER TABLE configs ALTER COLUMN agent_id TYPE bigint;

-- Rename indexes back to original names
ALTER INDEX actors_pkey RENAME TO agents_pkey;
ALTER INDEX actors_name RENAME TO agents_name;

-- Rename foreign key constraints back to original names
ALTER TABLE configs RENAME CONSTRAINT configs_actor_id_fkey TO configs_agent_id_fkey;
ALTER TABLE api_tokens RENAME CONSTRAINT api_tokens_actor_id_fkey TO api_tokens_agent_id_fkey;

-- Rename the sequence back to original name
ALTER SEQUENCE actors_id_seq RENAME TO agents_id_seq;

-- Rename the table back to original name
ALTER TABLE actors RENAME TO agents;