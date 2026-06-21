-- +goose Up
-- Client joins the catalog's natural key (ADR-0015): model names are client-specific
-- (Pi and opencode report the same model under different provider/slug names), so the
-- catalog is keyed on (client, provider, slug), not (provider, slug). SQLite cannot drop
-- the old UNIQUE(provider, slug), so rebuild the table — preserving `id` so the
-- session_agents.model_id -> models(id) FK stays valid (session_agents is empty today).
-- Existing rows are attributed to 'pi', the only client that has ever written.
CREATE TABLE models_new (
    id       TEXT PRIMARY KEY NOT NULL,
    client   TEXT    NOT NULL,
    provider TEXT    NOT NULL,
    slug     TEXT    NOT NULL,
    enabled  INTEGER NOT NULL DEFAULT 1,
    UNIQUE (client, provider, slug)
);
INSERT INTO models_new (id, client, provider, slug, enabled)
    SELECT id, 'pi', provider, slug, enabled FROM models;
DROP TABLE models;
ALTER TABLE models_new RENAME TO models;

-- +goose Down
-- Reverse to the (provider, slug) key, dropping client. This FAILS if two clients share a
-- (provider, slug) — acceptable for a dev rollback; the forward key is the supported state.
CREATE TABLE models_old (
    id       TEXT PRIMARY KEY NOT NULL,
    provider TEXT    NOT NULL,
    slug     TEXT    NOT NULL,
    enabled  INTEGER NOT NULL DEFAULT 1,
    UNIQUE (provider, slug)
);
INSERT INTO models_old (id, provider, slug, enabled)
    SELECT id, provider, slug, enabled FROM models;
DROP TABLE models;
ALTER TABLE models_old RENAME TO models;
