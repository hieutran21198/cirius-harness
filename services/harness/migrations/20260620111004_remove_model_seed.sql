-- +goose Up
-- Models are client-reported now (ADR-0011): a client syncs its enabled models into the
-- catalog at session start, so the harness no longer ships a hardcoded list. Remove the rows
-- seeded by 20260619091805_seed_system_agents.sql. Safe: no session_agents reference them yet.
-- (Append-only: the seed migration is left intact; this reverses just its model rows.)
DELETE FROM models WHERE (provider, slug) IN (
    ('anthropic', 'claude-opus-4-7'),
    ('anthropic', 'claude-opus-4-8'),
    ('anthropic', 'claude-sonnet-4-6'),
    ('openai',    'gpt-5.5'),
    ('openai',    'gpt-5.4'),
    ('minimax',   'minimax-m3'),
    ('moonshot',  'kimi-k2.7'),
    ('google',    'gemini-3-pro'),
    ('deepseek',  'deepseek-v3')
);

-- +goose Down
-- Restore the original seed rows (mirrors 20260619091805) so a rollback is faithful.
INSERT INTO models (id, provider, slug) VALUES
    ('019edf6f-a3e2-7a8d-b51a-16c8e93205a8', 'anthropic', 'claude-opus-4-7'),
    ('019edf6f-a3e2-7aca-88d7-6136b67c590f', 'anthropic', 'claude-opus-4-8'),
    ('019edf6f-a3e2-7add-bd5d-1969c2d58b75', 'anthropic', 'claude-sonnet-4-6'),
    ('019edf6f-a3e2-7aec-ab4e-e3daf5f76b6a', 'openai',    'gpt-5.5'),
    ('019edf6f-a3e2-7afb-998b-0d701d2fb1ba', 'openai',    'gpt-5.4'),
    ('019edf6f-a3e2-7b0a-b5f9-fa8ba8e3503f', 'minimax',   'minimax-m3'),
    ('019edf6f-a3e2-7b1a-b59c-a3e63892be30', 'moonshot',  'kimi-k2.7'),
    ('019edf6f-a3e2-7b29-90b8-7fe59739b249', 'google',    'gemini-3-pro'),
    ('019edf6f-a3e2-7b38-b62f-46c272418582', 'deepseek',  'deepseek-v3')
ON CONFLICT(provider, slug) DO UPDATE SET enabled = excluded.enabled;
