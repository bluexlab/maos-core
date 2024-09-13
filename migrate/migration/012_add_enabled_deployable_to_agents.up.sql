ALTER TABLE agents
    ADD COLUMN enabled bool NOT NULL DEFAULT true,
    ADD COLUMN deployable bool NOT NULL DEFAULT false,
    ADD COLUMN configurable bool NOT NULL DEFAULT false;
