-- Remove the 'role' column from the 'actors' table
ALTER TABLE actors DROP COLUMN role;

-- Drop the 'actor_role' enum type
DROP TYPE IF EXISTS actor_role;
