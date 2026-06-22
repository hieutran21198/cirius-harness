-- +goose Up
-- The audit log: an append-only record of what the harness did, turning the ephemeral
-- stderr logs into queryable state (ADR-0016 spirit; distinct from scribe's distilled
-- lessons). One row per recorded happening; never updated or deleted.
CREATE TABLE events (
    id          TEXT PRIMARY KEY NOT NULL,
    occurred_at DATETIME NOT NULL,
    kind        TEXT NOT NULL,
    actor       TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL,
    message     TEXT NOT NULL DEFAULT '',
    detail      TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_events_occurred_at ON events (occurred_at);
CREATE INDEX idx_events_kind ON events (kind);

-- +goose Down
DROP TABLE events;
