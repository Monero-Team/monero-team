# Contributing

Thank you for helping build a calm, trustworthy, privacy-respecting platform.
Contributions are reviewed in the open — every change lands through a visible
pull request.

## Ground rules (these are not negotiable)

These follow directly from the project's values. A pull request that breaks any
of them will be asked to change before review continues:

- **No external runtime dependencies** without explicit maintainer agreement.
  The base product makes no third-party API calls and bundles no CDN assets.
- **No JavaScript in the base reading path.** JavaScript may only be added as
  progressive enhancement on top of a fully working no-JS version.
- **No trackers, no analytics, no cookies** for core functionality.
- **Design tokens are the single source of truth.** Use the CSS variables from
  `tokens.css`; do not hard-code colours, type sizes, or spacing. Do not add new
  design tokens to the locked token file. No shadows, gradients, glow, or neon.
- **Sentence case** in all UI copy. Never CAPS, never Title Case.

## Code contributions

1. Fork and create a branch from `main`.
2. Make your change. Keep it small and focused.
3. Run the checks locally before opening a PR:
   ```sh
   make check        # gofmt -l, go vet, go test, no-external-origins guard
   ```
   or, without make:
   ```sh
   gofmt -l .
   go vet ./...
   go test ./...
   ./scripts/check-no-external-origins.sh
   ```
4. Open a pull request. CI must pass and one maintainer must approve before
   merge. Describe *why*, not just *what*.

Keep commits readable; prefer present-tense, descriptive messages.

## Submitting a resource to the directory

The directory is community-curated and stored as version-controlled data, so
submissions are reviewed as pull requests in the open. **This flow is not built
yet** — it arrives in a later stage, together with an anonymous (no-account)
intake path so that contributing does not require a platform or hosting-provider
identity. Until then, please do not open directory-data PRs against this
repository.

## Reporting problems

- **Security vulnerabilities:** follow [SECURITY.md](SECURITY.md) — do not open a
  public issue.
- **Bugs and proposals:** open an issue describing the behaviour and how to
  reproduce it.
