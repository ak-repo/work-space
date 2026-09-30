**Technical Design & Implementation Specification**

Go + PostgreSQL + Jev System One + optional LLM fallback + HTML
dashboard

| **Document** | **Value**                                                                      |
|--------------|--------------------------------------------------------------------------------|
| Status       | Implementation-ready design                                                    |
| Version      | 1.0                                                                            |
| Date         | 29 September 2026                                                              |
| Primary goal | Demonstrate System-One decision AI in a real backend control-flow architecture |
| Demo domain  | IT / SaaS incident and support-event triage                                    |

# 1. Goal and Project Definition

Build a production-shaped demonstration system where unstructured
operational events are converted into typed decisions by Jev, evaluated
by deterministic Go policy rules, persisted for audit, and streamed to a
live dashboard. Low-confidence or high-risk cases can be handed to an
LLM or a human review queue.

- Demonstrate that Jev is a decision component, not a chatbot.

- Show Choice, Score, and Noul judgments against the same application
  state.

- Keep business control flow, thresholds, authorization, retries, and
  side effects in Go.

- Measure latency, decision distributions, fallback rate, and agreement
  with labelled expected outcomes.

- Provide a clean path for comparing Jev-only, LLM-only, and Jev→LLM
  fallback modes.

# 2. Scope

| **In scope**                                        | **Out of scope for v1**                       |
|-----------------------------------------------------|-----------------------------------------------|
| Synthetic and manually submitted operational events | Training or fine-tuning Jev                   |
| Jev decision integration                            | Autonomous destructive infrastructure actions |
| Deterministic policy engine                         | Complex multi-agent orchestration             |
| PostgreSQL audit trail                              | Kubernetes deployment                         |
| SSE live dashboard                                  | Native mobile app                             |
| Optional LLM fallback adapter                       | Computer vision/audio input                   |
| Benchmark/evaluation runner                         | Replacing incident-management products        |

# 3. Jev Integration Model

The current public Jev interface is described as a System-One endpoint
accepting state plus named typed questions. The design below
deliberately isolates the provider contract because Jev is new and the
exact model identifiers, limits, pricing, and SDK surface can change.

> Application event
>
> \|
>
> v
>
> DecisionService
>
> \|
>
> v
>
> JevProvider.Decide(ctx, DecisionRequest)
>
> \|
>
> +--\> Choice: category / route / next action
>
> +--\> Score: severity
>
> +--\> Noul: escalate? / needs human?
>
> \|
>
> v
>
> Normalized DecisionResult
>
> \|
>
> v
>
> PolicyEngine

| **Primitive** | **Project use**                               | **Application handling**                           |
|---------------|-----------------------------------------------|----------------------------------------------------|
| Choice        | Classify event category and recommended route | Use winner + probability distribution + confidence |
| Score         | Severity/impact rubric                        | Map ordered score to internal severity band        |
| Noul          | Boolean-style escalation judgment             | Compare probability against configured threshold   |

# 4. Functional Requirements

- FR-01: Accept an event through REST and through a synthetic event
  generator.

- FR-02: Validate and normalize event state before sending it to any AI
  provider.

- FR-03: Ask Jev multiple independent questions for category, severity,
  escalation, and next action.

- FR-04: Persist the raw normalized event, decision answers,
  probabilities, model version, latency, and policy outcome.

- FR-05: Execute only safe demo actions: route, ignore, queue review,
  create alert, or request LLM analysis.

- FR-06: Stream event/decision/action updates to the browser via SSE.

- FR-07: Allow thresholds and provider mode to be configured without
  changing decision code.

- FR-08: Provide a review endpoint to record the expected/correct
  outcome for evaluation.

- FR-09: Provide metrics for throughput, p50/p95 latency, auto-decision
  rate, fallback rate, and labelled accuracy.

- FR-10: Never expose Jev or LLM API keys to the frontend.

# 5. Non-Functional Requirements

| **Area**      | **Requirement**                                                                                              |
|---------------|--------------------------------------------------------------------------------------------------------------|
| Reliability   | Timeout all provider calls; bounded retry only for transient failures; no retry on validation/auth failures. |
| Auditability  | Every model decision must be traceable to event, model version, question-set version, and policy version.    |
| Security      | Secrets server-side only; redact configured sensitive fields before external model calls.                    |
| Performance   | Backend overhead excluding provider call should remain small; measure provider latency separately.           |
| Idempotency   | POST /events accepts an idempotency key to prevent duplicate processing.                                     |
| Extensibility | Jev and LLM integrations implement provider interfaces; business services depend on interfaces.              |
| Observability | Structured logs, request IDs, metrics, decision latency, fallback reason.                                    |

# 6. High-Level Architecture

> Browser / Demo Generator
>
> \|
>
> v
>
> HTTP API (Go)
>
> \|
>
> v
>
> EventService
>
> \| \|
>
> \| +---------------------\> PostgreSQL
>
> v
>
> DecisionService
>
> \|
>
> v
>
> JevProvider
>
> \|
>
> v
>
> Normalized Decision
>
> \|
>
> v
>
> PolicyEngine
>
> / \| \\
>
> / \| \\
>
> AUTO REVIEW LLM FALLBACK
>
> \| \| \|
>
> +-------+----------+
>
> \|
>
> v
>
> ActionService
>
> \|
>
> +----\> audit/event tables
>
> +----\> SSE Broker ----\> Dashboard

# 7. Recommended Repository Structure

> cmd/
>
> server/
>
> main.go
>
> simulator/
>
> main.go
>
> evaluator/
>
> main.go
>
> internal/
>
> config/
>
> http/
>
> handler/
>
> middleware/
>
> response/
>
> event/
>
> model.go
>
> service.go
>
> repository.go
>
> decision/
>
> model.go
>
> service.go
>
> provider.go
>
> provider/
>
> jev/
>
> client.go
>
> dto.go
>
> mapper.go
>
> llm/
>
> client.go
>
> mapper.go
>
> policy/
>
> engine.go
>
> rules.go
>
> action/
>
> service.go
>
> review/
>
> service.go
>
> repository.go
>
> metrics/
>
> stream/
>
> sse.go
>
> storage/
>
> postgres/
>
> migrations/
>
> web/
>
> index.html
>
> app.js
>
> styles.css
>
> configs/
>
> config.example.yaml
>
> docker-compose.yml
>
> Makefile
>
> README.md

Exact names may be adjusted to match the repository conventions when
implementation begins; the key boundary is event → decision provider →
policy → action.

# 8. Domain Model

| **Entity**      | **Purpose**                        | **Important fields**                                                         |
|-----------------|------------------------------------|------------------------------------------------------------------------------|
| Event           | Incoming operational state         | id, source, type, title, body/state, occurred_at, received_at                |
| DecisionRun     | One provider evaluation            | id, event_id, provider, model, question_set_version, latency_ms, status      |
| DecisionAnswer  | Normalized answer                  | question_key, type, selected_value, numeric_value, confidence, probabilities |
| PolicyOutcome   | Deterministic application decision | decision_run_id, action, reason_code, policy_version                         |
| ActionExecution | Safe demo action result            | action, status, payload, executed_at                                         |
| HumanReview     | Ground truth/correction            | event_id, expected_category, expected_severity, expected_action, notes       |

# 9. Jev Question Set

Use stable question keys and version the entire set. Initial question
set: incident_triage/v1.

> {
>
> "category": {
>
> "type": "choice",
>
> "instructions": "Classify the primary operational issue.",
>
> "criteria": {
>
> "billing": "Payment, invoice, subscription, refund, or charge issue.",
>
> "technical": "Application bug, integration failure, or software
> malfunction.",
>
> "infrastructure": "Service, server, database, network, or availability
> issue.",
>
> "security": "Authentication abuse, suspicious access, data exposure,
> or security event.",
>
> "general": "Does not fit the other operational categories."
>
> }
>
> },
>
> "severity": {
>
> "type": "score",
>
> "instructions": "Rate the operational severity.",
>
> "criteria": \[
>
> "Low: little or no user impact",
>
> "Medium: limited degradation or individual user impact",
>
> "High: major degradation or many users affected",
>
> "Critical: outage, severe security risk, or major business impact"
>
> \]
>
> },
>
> "needs_escalation": {
>
> "type": "noul",
>
> "instructions": "This event requires immediate human escalation."
>
> },
>
> "next_action": {
>
> "type": "choice",
>
> "instructions": "Choose the safest immediate handling action.",
>
> "criteria": {
>
> "observe": "Record and monitor; no immediate intervention.",
>
> "route_support": "Send to the normal support workflow.",
>
> "page_engineer": "Escalate immediately to an on-call engineer.",
>
> "security_review": "Send immediately to the security review
> workflow.",
>
> "human_review": "The correct action is uncertain and needs a human."
>
> }
>
> }
>
> }

# 10. Internal Go Contracts

> type DecisionProvider interface {
>
> Decide(ctx context.Context, req DecisionRequest) (DecisionResult,
> error)
>
> }
>
> type DecisionRequest struct {
>
> State any
>
> QuestionSet QuestionSet
>
> Model string
>
> }
>
> type DecisionResult struct {
>
> Provider string
>
> Model string
>
> Answers map\[string\]Answer
>
> Usage Usage
>
> Latency time.Duration
>
> }
>
> type Answer struct {
>
> Type AnswerType
>
> Choice string
>
> Score \*float64
>
> Probability \*float64
>
> Confidence \*float64
>
> Probabilities map\[string\]float64
>
> }

Do not pass Jev-specific DTOs beyond provider/jev. The mapper converts
the external response into the internal DecisionResult so policy and
persistence are provider-independent.

# 11. Policy Engine

The policy engine owns the final automated decision. AI output is
evidence, not authority.

> Example policy v1:
>
> if provider_error:
>
> action = HUMAN_REVIEW
>
> else if category == "security"
>
> and category_confidence \>= 0.85:
>
> action = SECURITY_REVIEW
>
> else if needs_escalation \>= 0.90
>
> and category_confidence \>= 0.80:
>
> action = PAGE_ENGINEER
>
> else if category_confidence \< 0.75:
>
> action = LLM_FALLBACK
>
> else if top_choice_margin \< 0.15:
>
> action = HUMAN_REVIEW
>
> else:
>
> action = ROUTE_NORMAL

Thresholds above are initial demonstration values, not claims about Jev
calibration. They must be tuned against a labelled evaluation set.
High-impact actions should require stronger gates than low-risk routing.

# 12. LLM Fallback

- Disabled in the first implementation milestone.

- Triggered by policy, never directly by Jev.

- Use only when confidence/margin is insufficient or the event is
  explicitly configured as complex.

- Return a normalized fallback result; do not let free-form model text
  directly execute an action.

- Persist fallback reason, model, latency, token usage, and resulting
  action.

- For the demo, compare Jev-only, LLM-only, and hybrid modes on the same
  labelled dataset.

# 13. Database Design

> events
>
> id uuid PK
>
> source varchar
>
> event_type varchar
>
> title text
>
> state jsonb
>
> idempotency_key varchar NULL UNIQUE
>
> occurred_at timestamptz
>
> created_at timestamptz
>
> decision_runs
>
> id uuid PK
>
> event_id uuid FK -\> events(id)
>
> provider varchar
>
> model varchar
>
> question_set_version varchar
>
> status varchar
>
> latency_ms integer
>
> input_usage integer NULL
>
> output_usage integer NULL
>
> error_code varchar NULL
>
> created_at timestamptz
>
> decision_answers
>
> id uuid PK
>
> decision_run_id uuid FK -\> decision_runs(id)
>
> question_key varchar
>
> answer_type varchar
>
> selected_value varchar NULL
>
> numeric_value double precision NULL
>
> confidence double precision NULL
>
> probabilities jsonb
>
> created_at timestamptz
>
> UNIQUE(decision_run_id, question_key)
>
> policy_outcomes
>
> id uuid PK
>
> decision_run_id uuid FK -\> decision_runs(id)
>
> policy_version varchar
>
> action varchar
>
> reason_code varchar
>
> details jsonb
>
> created_at timestamptz
>
> action_executions
>
> id uuid PK
>
> event_id uuid FK -\> events(id)
>
> policy_outcome_id uuid FK -\> policy_outcomes(id)
>
> action varchar
>
> status varchar
>
> result jsonb
>
> created_at timestamptz
>
> human_reviews
>
> id uuid PK
>
> event_id uuid FK -\> events(id)
>
> expected_category varchar NULL
>
> expected_severity integer NULL
>
> expected_action varchar NULL
>
> notes text NULL
>
> created_at timestamptz

| **Index**                                | **Reason**              |
|------------------------------------------|-------------------------|
| events(created_at DESC)                  | Recent-event dashboard  |
| decision_runs(event_id, created_at DESC) | Event audit lookup      |
| decision_runs(provider, created_at DESC) | Provider metrics        |
| policy_outcomes(action, created_at DESC) | Action/fallback metrics |
| human_reviews(event_id)                  | Evaluation joins        |

# 14. Transaction Boundaries

- Insert Event first and commit before the external AI call; provider
  latency must not hold a DB transaction open.

- After Jev returns, persist DecisionRun + DecisionAnswers +
  PolicyOutcome atomically.

- Action execution occurs after the decision transaction commits.

- Persist ActionExecution separately so a failed action does not erase
  the model/policy audit trail.

- Idempotency conflicts return the existing event rather than creating a
  second decision pipeline unless explicit reprocess is requested.

# 15. HTTP API

| **Method** | **Path**                      | **Purpose**                                                    |
|------------|-------------------------------|----------------------------------------------------------------|
| POST       | /api/v1/events                | Submit and process one event                                   |
| GET        | /api/v1/events                | Paginated recent events                                        |
| GET        | /api/v1/events/{id}           | Event + decisions + policy + actions                           |
| POST       | /api/v1/events/{id}/reprocess | Run a new decision using current configured model/question set |
| POST       | /api/v1/events/{id}/review    | Record expected/correct result                                 |
| GET        | /api/v1/metrics/summary       | Dashboard KPI summary                                          |
| GET        | /api/v1/metrics/latency       | Latency series/distribution                                    |
| GET        | /api/v1/stream                | SSE event stream                                               |
| GET        | /health/live                  | Process liveness                                               |
| GET        | /health/ready                 | DB/config readiness                                            |

## 16. POST /events Contract

> Request
>
> {
>
> "source": "demo-simulator",
>
> "event_type": "service_alert",
>
> "title": "Payment API failures",
>
> "state": {
>
> "message": "Payment API failed 14 times in two minutes",
>
> "service": "payments",
>
> "error_rate": 0.42,
>
> "affected_regions": \["ap-south-1"\]
>
> },
>
> "occurred_at": "2026-09-29T10:00:00+05:30"
>
> }
>
> Response
>
> {
>
> "event_id": "...",
>
> "status": "processed",
>
> "decision": {
>
> "category": "...",
>
> "severity": 0,
>
> "needs_escalation": 0.0,
>
> "next_action": "..."
>
> },
>
> "policy": {
>
> "action": "...",
>
> "reason_code": "..."
>
> }
>
> }

# 17. Processing Sequence

> 1\. Handler validates request.
>
> 2\. EventService normalizes and persists event.
>
> 3\. DecisionService loads incident_triage/v1.
>
> 4\. JevProvider sends state + questions with timeout.
>
> 5\. Provider response is mapped to internal answers.
>
> 6\. PolicyEngine evaluates confidence, margins, Noul threshold and
> risk.
>
> 7\. Decision + policy outcome are persisted atomically.
>
> 8\. ActionService executes safe action.
>
> 9\. Action result is persisted.
>
> 10\. Stream broker publishes update to connected dashboards.
>
> 11\. Metrics are updated.

# 18. Failure Handling

| **Failure**                            | **Behavior**                                                                                   |
|----------------------------------------|------------------------------------------------------------------------------------------------|
| Jev timeout                            | Persist failed DecisionRun; route to review/fallback according to policy; never lose event.    |
| 401/403 provider auth                  | No retry; mark provider unavailable/config error.                                              |
| 429                                    | Retry with bounded exponential backoff if request deadline permits; otherwise fallback/review. |
| 5xx/network                            | Small bounded retry with jitter; then fallback/review.                                         |
| Malformed/unexpected provider response | Treat as provider failure; retain sanitized raw metadata for debugging if allowed.             |
| DB unavailable before event insert     | Return 503; do not call provider.                                                              |
| DB failure after provider call         | Return processing failure and log request ID; reprocessing must be safe.                       |
| SSE client disconnect                  | No impact on processing; client reconnects and refreshes recent events.                        |

# 19. Security and Privacy

- Keep TYPESAFE/JEV and LLM credentials only in backend
  environment/config secret storage.

- Never log Authorization headers.

- Define an outbound-state sanitizer that removes configured secrets,
  tokens, passwords, and unnecessary PII.

- Limit request body size before decoding.

- Use allowlisted event fields when constructing external AI state for
  sensitive integrations.

- Protect review/reprocess/admin endpoints with authentication before
  any non-local deployment.

- Use TLS outside local development.

- Do not automatically execute destructive infrastructure commands in
  this demonstration.

# 20. Configuration

> server:
>
> address: ":8080"
>
> database:
>
> url: "\${DATABASE_URL}"
>
> jev:
>
> base_url: "https://api.typesafe.ai"
>
> api_key: "\${JEV_API_KEY}"
>
> model: "jev-latest"
>
> timeout: "2s"
>
> policy:
>
> version: "incident-policy/v1"
>
> category_confidence_min: 0.75
>
> escalation_probability: 0.90
>
> high_risk_confidence_min: 0.85
>
> choice_margin_min: 0.15
>
> low_confidence_action: "llm_fallback"
>
> llm:
>
> enabled: false
>
> provider: ""
>
> model: ""
>
> timeout: "8s"

Pin a Jev model version for benchmark/reproducibility runs; aliases are
useful for experimentation but can change behavior.

# 21. Dashboard

| **Widget**                  | **Purpose**                                                                           |
|-----------------------------|---------------------------------------------------------------------------------------|
| Live event feed             | Show event → Jev decision → policy action in real time                                |
| Decision inspector          | Display Choice probabilities, Score distribution, Noul probability, model and latency |
| Automation rate             | Percentage auto-routed vs LLM fallback vs human review                                |
| Latency                     | Jev p50/p95 and end-to-end processing latency                                         |
| Category distribution       | Operational event mix                                                                 |
| Severity distribution       | Current workload risk mix                                                             |
| Confidence/margin histogram | Understand threshold behavior                                                         |
| Evaluation accuracy         | Agreement with human-labelled expected outcomes                                       |
| Provider comparison         | Jev vs LLM vs hybrid latency/cost/accuracy where measured                             |

Avoid duplicate KPI cards. The dashboard should emphasize decision flow
and model behavior rather than generic infrastructure monitoring.

# 22. SSE Event Shape

> event: decision.completed
>
> data: {
>
> "event_id": "...",
>
> "title": "Payment API failures",
>
> "category": "infrastructure",
>
> "category_confidence": 0.93,
>
> "severity": 2.7,
>
> "escalation_probability": 0.94,
>
> "policy_action": "page_engineer",
>
> "latency_ms": 118,
>
> "created_at": "..."
>
> }

# 23. Demo Dataset

Create 50–100 labelled events covering obvious, ambiguous, and
adversarial/boundary cases.

| **Class**         | **Examples**                                                                  |
|-------------------|-------------------------------------------------------------------------------|
| Billing           | Duplicate charge, refund request, invoice mismatch, subscription cancellation |
| Technical         | SDK error, broken integration, UI bug, failed import                          |
| Infrastructure    | Database saturation, service outage, latency spike, queue backlog             |
| Security          | Suspicious login, leaked token report, privilege anomaly, phishing report     |
| General/ambiguous | Incomplete complaint, mixed billing+technical request, vague user message     |

Each fixture should include expected category, severity band, escalation
expectation, and expected safe action.

# 24. Evaluation Method

- Run the same immutable labelled dataset against a pinned Jev model and
  question-set version.

- Measure category accuracy and confusion matrix.

- Measure severity agreement using exact-band and adjacent-band
  agreement.

- Measure escalation precision/recall at several probability thresholds.

- Measure coverage: percentage of cases auto-handled at each confidence
  threshold.

- Measure p50/p95 provider latency and total pipeline latency.

- For hybrid mode, measure how many events reach the LLM and whether
  fallback improves labelled correctness.

- Do not claim Jev is faster/cheaper/more accurate from marketing
  figures; benchmark the actual integration.

# 25. Observability

| **Signal** | **Examples**                                                                                |
|------------|---------------------------------------------------------------------------------------------|
| Logs       | request_id, event_id, decision_run_id, provider, model, latency_ms, action, fallback_reason |
| Counters   | events_total, decisions_total, provider_errors_total, fallbacks_total, human_reviews_total  |
| Histograms | jev_latency_ms, pipeline_latency_ms, confidence, choice_margin                              |
| Gauges     | SSE clients, pending review count                                                           |

Never use event body/state as a metric label; it creates cardinality and
privacy problems.

# 26. Testing Strategy

| **Test type**     | **Coverage**                                                                   |
|-------------------|--------------------------------------------------------------------------------|
| Unit              | Policy thresholds, choice margin, severity mapping, sanitizer, DTO mapper      |
| Provider contract | Recorded/fixture Jev responses including Choice/Score/Noul and error responses |
| Repository        | Insert/read, uniqueness, transaction rollback, JSONB probability persistence   |
| HTTP              | Validation, idempotency, status codes, pagination                              |
| Integration       | PostgreSQL + fake decision provider through complete event pipeline            |
| SSE               | Publish, reconnect behavior, client removal                                    |
| Evaluation        | Golden labelled fixture set with reproducible report                           |
| Failure           | Timeout, 429, 5xx, malformed provider result, DB failure                       |

CI should use a fake Jev provider. Live Jev tests should be opt-in so
normal tests do not depend on network, quota, or model drift.

# 27. Migration and Rollback Rules

- Use additive migrations for v1 tables and indexes.

- Do not store only a flattened final action; retain the normalized
  answer distribution required for audit/evaluation.

- Rollback should drop dependent tables in reverse FK order only for
  local/demo migrations.

- In a production evolution, prefer forward fixes over destructive
  rollback after decision history exists.

- Question-set and policy changes are versioned data/config changes, not
  destructive rewrites of historical rows.

# 28. Implementation Order

| **Phase** | **Deliverable**                                                                                    |
|-----------|----------------------------------------------------------------------------------------------------|
| 0         | Obtain Jev API access; run one manual Choice/Score/Noul request; freeze verified contract fixture. |
| 1         | Go skeleton, config, PostgreSQL, migrations, health endpoints.                                     |
| 2         | Event model/repository/service and POST/GET event APIs.                                            |
| 3         | DecisionProvider interface + fake provider + unit/integration tests.                               |
| 4         | Real Jev HTTP client, DTO mapping, timeouts/errors, contract tests.                                |
| 5         | Policy engine + persisted DecisionRun/Answers/PolicyOutcome.                                       |
| 6         | Action service with safe demo actions.                                                             |
| 7         | SSE broker + live HTML dashboard.                                                                  |
| 8         | Simulator and labelled fixture dataset.                                                            |
| 9         | Evaluation CLI + metrics/dashboard benchmark views.                                                |
| 10        | Optional LLM provider and low-confidence fallback.                                                 |
| 11        | Docker Compose, README, demo script, final end-to-end verification.                                |

# 29. Acceptance Criteria

- A submitted event produces a persisted Jev decision and deterministic
  policy outcome.

- Choice, Score, and Noul results are visible in the event detail view.

- The browser receives completed decisions without polling.

- A low-confidence fixture is routed to review/fallback rather than
  blindly executed.

- Provider failure does not lose the original event.

- Reprocessing creates a new DecisionRun and preserves history.

- Evaluation can run against labelled fixtures and output reproducible
  metrics.

- No external AI credential is present in browser assets, logs, or
  committed configuration.

- All normal automated tests pass without requiring live Jev access.

# 30. Main Risks

| **Risk**                         | **Mitigation**                                                                              |
|----------------------------------|---------------------------------------------------------------------------------------------|
| Jev API/model changes            | Provider isolation, pinned benchmark model, contract fixtures, configurable base URL/model. |
| Over-trusting confidence         | Empirical thresholds, choice-margin check, human/LLM fallback, high-risk gates.             |
| Prompt/question drift            | Version question sets and benchmark every change.                                           |
| Model drift through latest alias | Pin versions for evaluation and controlled production changes.                              |
| Sensitive data leakage           | Outbound sanitizer, allowlists, minimal state, secret redaction.                            |
| Demo becomes an LLM app          | Keep LLM disabled until Jev-only pipeline and evaluation are complete.                      |

# 31. Environment Prerequisites

- Go toolchain (use the repository/team-supported current version).

- PostgreSQL 16+ recommended for local development.

- Docker + Docker Compose for repeatable local setup.

- Jev/TypeSafe API key and access to current API documentation.

- Browser with EventSource/SSE support.

- Optional LLM API key only after the Jev-only milestones are complete.

# 32. Definition of Done

The project is done when it can be cloned, configured with a database
and Jev key, started with documented commands, process a deterministic
demo dataset end-to-end, show live decisions on the dashboard, preserve
a complete audit trail, and generate an evaluation report demonstrating
where Jev auto-decides versus where the policy requires fallback or
human review.

# 33. Source Notes

Jev is a newly released system. Before implementation, verify the
provider contract against TypeSafe's current official documentation.
Public developer references checked for this design consistently
describe a System-One request containing state, model, and named
questions; Choice, Score, and Noul typed decisions; and the TypeSafe
/v1/systemone endpoint. Performance/cost claims should be independently
benchmarked rather than copied into product requirements.

References checked 29 September 2026:

- TypeSafe Jev API reference mirror:
  https://typesafe-jev.com/en/guides/api/

- Jev manual API reference (official-contract cross-check):
  https://jevmanual.com/manual/api/

- Financial Times coverage of TypeSafe/Jev launch:
  https://www.ft.com/content/456884ea-2558-4648-8036-a77b73733430
