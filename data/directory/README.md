# Directory dataset

One JSON file per resource, named `<slug>.json` (the file name must equal the
`slug` field). The schema, enums, and validation rules live in
[`internal/directory`](../../internal/directory); invalid data fails the build
(CI runs `TestRealDataValid`) and prevents the server from starting.

Released under **CC0 1.0** — see [`../LICENSE`](../LICENSE).

## Moderation guardrail (read before adding entries)

These seed entries are **provisional**. Statuses are subject to real
moderation, which is not built yet.

- **Real, named entities are entered at `admitted` only** — a neutral baseline
  meaning "listed, not yet reviewed". Never assign a real service `verified`,
  `questionable`, or `scam` without a substantiated review: an unfounded status
  is a reputational claim and is against this project's values.
- The `example-scam-service` and `example-questionable-service` entries are
  **explicitly fictional placeholders** (domain `example.invalid`, which is
  reserved and never resolves). They exist solely so later stages can render the
  `questionable` and `scam` states. Do not model them on any real service.

## Status meanings

| status         | meaning                                               |
| -------------- | ----------------------------------------------------- |
| `verified`     | reviewed and confirmed (reserved; not auto-assigned)  |
| `admitted`     | listed, neutral baseline, not yet reviewed            |
| `questionable` | substantiated concerns                                |
| `scam`         | substantiated as fraudulent                           |
