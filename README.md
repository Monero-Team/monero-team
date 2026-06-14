# Monero team

A privacy-first platform for the Monero ecosystem: a trusted directory of
services, a news aggregator, a weekly digest, and a self-hosted Monero
blockchain explorer — in one place, without surveillance, tracking, or KYC.

This project values verification over claims, transparency over marketing, and
long-term community value over growth metrics. It should feel like a technical
tool or a well-designed publication, not a typical cryptocurrency website.

## Principles

- **Privacy first** — no trackers, no behavioural analytics, no cookies. Core
  reading works with JavaScript disabled.
- **Open source** — all code is public and auditable. Anyone can run their own
  instance or fork.
- **No KYC** — no feature on the platform requires identity verification.
- **Verifiable trust** — trust is earned through verification, not promises.
  Moderation is public and append-only.
- **Minimalism** — typography carries the hierarchy; one accent colour; no
  shadows, gradients, or visual noise.

## Status

Early development. Current stage: **Stage 1 — project shell** (sticky navbar,
dark theme, no-JS base reading path, privacy-hardened defaults). Most sections
render a calm "coming soon" state until their stage lands. See the roadmap.

## Stack

- **Go (standard library)** — server-rendered HTML, single self-contained
  binary, minimal operational footprint.
- **No JavaScript framework, no Node, no frontend build.** JavaScript is only
  ever progressive enhancement layered on a fully working no-JS base.
- **Self-hosted fonts** (Inter Tight, JetBrains Mono — SIL OFL). No CDN, no
  Google Fonts — a CDN sees every visitor's IP.
- **No external runtime dependencies.** No third-party APIs in the base product.

## Run

```sh
go build ./cmd/server
./server            # serves on the configured port (default :8080)
# or, for development:
go run ./cmd/server
```

## Project layout

```
cmd/server            entrypoint, config, graceful shutdown
internal/web          router, security middleware, handlers
internal/web/templates  base layout + navbar/footer partials + page skeletons
internal/web/assets   embedded CSS, self-hosted fonts
internal/content/strings  centralized UI copy (English, locale `en`)
```

## Privacy posture

This repository ships software that sets **no cookies**, makes **no external
requests** in its base reading path, and serves a **strict Content-Security-
Policy**. These properties are enforced in CI, not merely promised — see
`scripts/check-no-external-origins.sh`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security issues:
[SECURITY.md](SECURITY.md).

## Licence

Application code is licensed under the **GNU AGPL-3.0** — anyone hosting a
modified version as a network service must publish their source. Bundled fonts
are under their own **SIL Open Font License**; see `third_party/fonts/`.
