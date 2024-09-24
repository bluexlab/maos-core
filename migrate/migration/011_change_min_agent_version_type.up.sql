ALTER TABLE configs
ALTER COLUMN min_agent_version TYPE integer[] USING string_to_array(min_agent_version, '.')::integer[];
