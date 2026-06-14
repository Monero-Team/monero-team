# Security policy

This is a privacy-focused project. We take security and confidentiality
seriously and appreciate responsible disclosure.

## Reporting a vulnerability

**Do not open a public issue for security vulnerabilities.**

Report privately through one of:

- GitHub's private vulnerability reporting (Security → Report a vulnerability),
  if enabled for this repository.
- Encrypted email to `<SECURITY_CONTACT_EMAIL>` using the PGP key published at
  `<PGP_KEY_URL_OR_FINGERPRINT>`.

> Maintainers: replace the placeholders above before publishing the repository.

Please include: a description of the issue, steps to reproduce, affected
component or version, and the potential impact.

## What to expect

- We aim to acknowledge a report within a few days.
- We will work with you on a fix and a coordinated disclosure timeline.
- Please give us reasonable time to release a fix before any public disclosure.

## Scope

In scope: the application code in this repository (server, templates, security
middleware, asset handling). Anything that could leak visitor data, weaken the
Content-Security-Policy, introduce tracking, set cookies, or make unexpected
external requests is especially relevant to this project's threat model.
