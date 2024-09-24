ALTER TABLE configs
ALTER COLUMN min_agent_version TYPE text USING array_to_string(min_agent_version, '.');