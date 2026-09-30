# Technical Design Summary

## Goal

Demonstrate Jev as a System-One decision component inside a production-shaped Go application. Incoming operational state is evaluated with typed questions; deterministic application policy decides what is allowed to happen next.

## Core flow

```text
Event -> REST API -> durable store
                    |
                    v
             DecisionProvider
              /          \
         Mock Jev       Live Jev
                    |
                    v
          normalized answers
                    |
                    v
              Policy Engine
                    |
        +-----------+-----------+
        |           |           |
      route       review      escalate
        |
        v
 durable store + SSE -> Dashboard
```

## Decision questions

- `category`: Choice — billing, technical, infrastructure, security, general.
- `severity`: Score — Low through Critical.
- `needs_escalation`: Noul — probability the event needs immediate human escalation.
- `next_action`: Choice — observe, route support, page engineer, security review, human review.

## Policy

The current demonstration gates on category confidence and the probability margin between the two strongest category choices. Security gets a dedicated route. High Noul escalation probability pages the engineer. Otherwise the event follows normal routing.

Thresholds are demonstration defaults and must be calibrated using labelled evaluation data before any high-impact use.

## Persistence

The runnable build uses a standard-library JSON snapshot store (`data/state.json`) so the demo has zero third-party runtime dependencies and can be verified offline. `migrations/001_init.sql` contains the PostgreSQL production schema for events, decision runs, typed answers, policy outcomes, and action executions. Replace the store implementation behind the same service boundary when moving to PostgreSQL.

## Provider isolation

Only `internal/provider/jev` understands TypeSafe's HTTP DTO. Business code depends on the `decision.Provider` interface. This is intentional because the Jev API and model versions are new and may evolve.
