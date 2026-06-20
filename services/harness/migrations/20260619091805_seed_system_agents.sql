-- +goose Up
-- Seed the default system team (Source = system) from .cirius-harness/00-system.yaml,
-- normalized per ADR-0007: a `models` catalog, a `tools` catalog, identity-only `agents`
-- (roles), and their `agent_tools` grants. IDs are fixed UUID v7 literals (SQLite generates
-- none). Idempotency keys on each table's natural/unique key so re-running reconciles edits.
--
-- NOT seeded (by design, ADR-0007): the per-agent model (model is bound per session on
-- session_agents.model_id), fallbacks (deferred), and permissions (authorization lives in
-- casbin_rule, owned by the Casbin gorm-adapter — ADR-0003).

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
ON CONFLICT(provider, slug) DO UPDATE SET
    enabled = excluded.enabled;

INSERT INTO tools (id, name) VALUES
    ('019ee267-b7a5-7a46-b4b6-977f3782ea3c', 'read'),
    ('019ee267-b7a5-7a91-93e3-de240b16a2fb', 'grep'),
    ('019ee267-b7a5-7aa3-a336-60853cdc24d0', 'glob'),
    ('019ee267-b7a5-7ab3-b2d1-03f889947b88', 'list'),
    ('019ee267-b7a5-7ac1-91b2-d1f9420fbdcc', 'edit'),
    ('019ee267-b7a5-7ad0-99f0-785da1f663ca', 'bash'),
    ('019ee267-b7a5-7adf-beb1-5195e7b5f701', 'webfetch'),
    ('019ee267-b7a5-7aee-8777-ba7c46b66e52', 'websearch')
ON CONFLICT(name) DO NOTHING;

INSERT INTO agents (id, name, archetype, responsibility, description, source, enabled) VALUES
    ('019edf40-0015-7b53-aa7e-47bbfb74dc7f', 'prayer',      'none',             'pray',     'Intentionally model-less no-op agent: does no work, just hopes everything succeeds.', 'system', 1),
    ('019edf40-0015-7bab-8b68-fe62baaa7bf7', 'council',     'communicator',     'route',    'Strategic orchestrator and communicator; routes work and delegates to other agents.', 'system', 1),
    ('019edf40-0015-7bcf-be2b-fc4426ee5c41', 'planner',     'communicator',     'design',   'Authors the detailed, structured implementation plan.', 'system', 1),
    ('019edf40-0015-7be0-aec8-7e28c22680c7', 'implementer', 'principle-driven', 'build',    'Deep technical coding; executes the plan. The only agent that may edit source.', 'system', 1),
    ('019edf40-0015-7bef-8aa1-543438826ead', 'researcher',  'principle-driven', 'gather',   'Investigation and research across web and codebase; the only agent with web access.', 'system', 1),
    ('019edf40-0015-7bfe-8b35-6947607b71f3', 'explorer',    'utility-runner',   'scan',     'Fast, cheap, high-volume read-only codebase scanning.', 'system', 1),
    ('019edf40-0015-7c0e-8306-b637a58dbab3', 'reviewer',    'communicator',     'critique', 'Code review and plan-gap analysis; advisory, never edits.', 'system', 1),
    ('019edf40-0015-7c1e-876d-fbfe6198ce58', 'scribe',      'communicator',     'retain',   'Captures technical debt, summaries, and lessons into durable team knowledge.', 'system', 1)
ON CONFLICT(name) DO UPDATE SET
    archetype      = excluded.archetype,
    responsibility = excluded.responsibility,
    description    = excluded.description,
    source         = excluded.source,
    enabled        = excluded.enabled;

-- Tool grants (pure junction; a conflicting row is already identical). prayer gets none.
-- council/planner/explorer/reviewer: read,grep,glob,list. implementer: +edit,+bash.
-- researcher: +webfetch,+websearch. scribe: +edit.
INSERT INTO agent_tools (agent_id, tool_id) VALUES
    -- council
    ('019edf40-0015-7bab-8b68-fe62baaa7bf7', '019ee267-b7a5-7a46-b4b6-977f3782ea3c'),
    ('019edf40-0015-7bab-8b68-fe62baaa7bf7', '019ee267-b7a5-7a91-93e3-de240b16a2fb'),
    ('019edf40-0015-7bab-8b68-fe62baaa7bf7', '019ee267-b7a5-7aa3-a336-60853cdc24d0'),
    ('019edf40-0015-7bab-8b68-fe62baaa7bf7', '019ee267-b7a5-7ab3-b2d1-03f889947b88'),
    -- planner
    ('019edf40-0015-7bcf-be2b-fc4426ee5c41', '019ee267-b7a5-7a46-b4b6-977f3782ea3c'),
    ('019edf40-0015-7bcf-be2b-fc4426ee5c41', '019ee267-b7a5-7a91-93e3-de240b16a2fb'),
    ('019edf40-0015-7bcf-be2b-fc4426ee5c41', '019ee267-b7a5-7aa3-a336-60853cdc24d0'),
    ('019edf40-0015-7bcf-be2b-fc4426ee5c41', '019ee267-b7a5-7ab3-b2d1-03f889947b88'),
    -- implementer
    ('019edf40-0015-7be0-aec8-7e28c22680c7', '019ee267-b7a5-7a46-b4b6-977f3782ea3c'),
    ('019edf40-0015-7be0-aec8-7e28c22680c7', '019ee267-b7a5-7a91-93e3-de240b16a2fb'),
    ('019edf40-0015-7be0-aec8-7e28c22680c7', '019ee267-b7a5-7aa3-a336-60853cdc24d0'),
    ('019edf40-0015-7be0-aec8-7e28c22680c7', '019ee267-b7a5-7ab3-b2d1-03f889947b88'),
    ('019edf40-0015-7be0-aec8-7e28c22680c7', '019ee267-b7a5-7ac1-91b2-d1f9420fbdcc'),
    ('019edf40-0015-7be0-aec8-7e28c22680c7', '019ee267-b7a5-7ad0-99f0-785da1f663ca'),
    -- researcher
    ('019edf40-0015-7bef-8aa1-543438826ead', '019ee267-b7a5-7a46-b4b6-977f3782ea3c'),
    ('019edf40-0015-7bef-8aa1-543438826ead', '019ee267-b7a5-7a91-93e3-de240b16a2fb'),
    ('019edf40-0015-7bef-8aa1-543438826ead', '019ee267-b7a5-7aa3-a336-60853cdc24d0'),
    ('019edf40-0015-7bef-8aa1-543438826ead', '019ee267-b7a5-7ab3-b2d1-03f889947b88'),
    ('019edf40-0015-7bef-8aa1-543438826ead', '019ee267-b7a5-7adf-beb1-5195e7b5f701'),
    ('019edf40-0015-7bef-8aa1-543438826ead', '019ee267-b7a5-7aee-8777-ba7c46b66e52'),
    -- explorer
    ('019edf40-0015-7bfe-8b35-6947607b71f3', '019ee267-b7a5-7a46-b4b6-977f3782ea3c'),
    ('019edf40-0015-7bfe-8b35-6947607b71f3', '019ee267-b7a5-7a91-93e3-de240b16a2fb'),
    ('019edf40-0015-7bfe-8b35-6947607b71f3', '019ee267-b7a5-7aa3-a336-60853cdc24d0'),
    ('019edf40-0015-7bfe-8b35-6947607b71f3', '019ee267-b7a5-7ab3-b2d1-03f889947b88'),
    -- reviewer
    ('019edf40-0015-7c0e-8306-b637a58dbab3', '019ee267-b7a5-7a46-b4b6-977f3782ea3c'),
    ('019edf40-0015-7c0e-8306-b637a58dbab3', '019ee267-b7a5-7a91-93e3-de240b16a2fb'),
    ('019edf40-0015-7c0e-8306-b637a58dbab3', '019ee267-b7a5-7aa3-a336-60853cdc24d0'),
    ('019edf40-0015-7c0e-8306-b637a58dbab3', '019ee267-b7a5-7ab3-b2d1-03f889947b88'),
    -- scribe
    ('019edf40-0015-7c1e-876d-fbfe6198ce58', '019ee267-b7a5-7a46-b4b6-977f3782ea3c'),
    ('019edf40-0015-7c1e-876d-fbfe6198ce58', '019ee267-b7a5-7ac1-91b2-d1f9420fbdcc'),
    ('019edf40-0015-7c1e-876d-fbfe6198ce58', '019ee267-b7a5-7a91-93e3-de240b16a2fb'),
    ('019edf40-0015-7c1e-876d-fbfe6198ce58', '019ee267-b7a5-7aa3-a336-60853cdc24d0'),
    ('019edf40-0015-7c1e-876d-fbfe6198ce58', '019ee267-b7a5-7ab3-b2d1-03f889947b88')
ON CONFLICT(agent_id, tool_id) DO NOTHING;

-- +goose Down
DELETE FROM agent_tools WHERE agent_id IN (
    '019edf40-0015-7bab-8b68-fe62baaa7bf7', '019edf40-0015-7bcf-be2b-fc4426ee5c41',
    '019edf40-0015-7be0-aec8-7e28c22680c7', '019edf40-0015-7bef-8aa1-543438826ead',
    '019edf40-0015-7bfe-8b35-6947607b71f3', '019edf40-0015-7c0e-8306-b637a58dbab3',
    '019edf40-0015-7c1e-876d-fbfe6198ce58');
DELETE FROM agents WHERE name IN
    ('prayer','council','planner','implementer','researcher','explorer','reviewer','scribe');
DELETE FROM tools WHERE id IN (
    '019ee267-b7a5-7a46-b4b6-977f3782ea3c', '019ee267-b7a5-7a91-93e3-de240b16a2fb',
    '019ee267-b7a5-7aa3-a336-60853cdc24d0', '019ee267-b7a5-7ab3-b2d1-03f889947b88',
    '019ee267-b7a5-7ac1-91b2-d1f9420fbdcc', '019ee267-b7a5-7ad0-99f0-785da1f663ca',
    '019ee267-b7a5-7adf-beb1-5195e7b5f701', '019ee267-b7a5-7aee-8777-ba7c46b66e52');
DELETE FROM models WHERE id IN (
    '019edf6f-a3e2-7a8d-b51a-16c8e93205a8', '019edf6f-a3e2-7aca-88d7-6136b67c590f',
    '019edf6f-a3e2-7add-bd5d-1969c2d58b75', '019edf6f-a3e2-7aec-ab4e-e3daf5f76b6a',
    '019edf6f-a3e2-7afb-998b-0d701d2fb1ba', '019edf6f-a3e2-7b0a-b5f9-fa8ba8e3503f',
    '019edf6f-a3e2-7b1a-b59c-a3e63892be30', '019edf6f-a3e2-7b29-90b8-7fe59739b249',
    '019edf6f-a3e2-7b38-b62f-46c272418582');
