CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source TEXT NOT NULL,
    event_type TEXT NOT NULL,
    title TEXT NOT NULL,
    state JSONB NOT NULL,
    idempotency_key TEXT NULL UNIQUE,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS decision_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    question_set_version TEXT NOT NULL,
    status TEXT NOT NULL,
    latency_ms INTEGER NOT NULL DEFAULT 0,
    input_tokens INTEGER NULL,
    output_tokens INTEGER NULL,
    error_code TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS decision_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_run_id UUID NOT NULL REFERENCES decision_runs(id) ON DELETE CASCADE,
    question_key TEXT NOT NULL,
    answer_type TEXT NOT NULL,
    selected_value TEXT NULL,
    numeric_value DOUBLE PRECISION NULL,
    confidence DOUBLE PRECISION NULL,
    probabilities JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(decision_run_id, question_key)
);

CREATE TABLE IF NOT EXISTS policy_outcomes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_run_id UUID NOT NULL REFERENCES decision_runs(id) ON DELETE CASCADE,
    policy_version TEXT NOT NULL,
    action TEXT NOT NULL,
    reason_code TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS action_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    policy_outcome_id UUID NOT NULL REFERENCES policy_outcomes(id) ON DELETE CASCADE,
    action TEXT NOT NULL,
    status TEXT NOT NULL,
    result JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_decision_runs_event ON decision_runs(event_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_decision_runs_provider ON decision_runs(provider, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_policy_outcomes_action ON policy_outcomes(action, created_at DESC);
