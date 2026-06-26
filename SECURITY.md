# Security policy

This is a privacy-focused project. We take security and confidentiality
seriously and appreciate responsible disclosure.

## Reporting a vulnerability

**Do not open a public issue for security vulnerabilities.**

Report privately through one of:

- **GitHub private vulnerability reporting** (Security → Report a vulnerability) —
  preferred, and encrypted in transit by GitHub.
- **Email `mail@monero.team`.** This inbox is hosted on Proton Mail, so messages
  are stored with zero-access encryption at rest. Note this is not end-to-end
  encryption unless you send from a Proton account or otherwise encrypt to us, so
  avoid putting exploit details in the body when a proof of concept can wait —
  send a brief first contact and we will arrange a secure channel.

Please include: a description of the issue, steps to reproduce, the affected
component, and the potential impact.

## What to expect

- We aim to acknowledge a report within a few days.
- We will work with you on a fix and a coordinated disclosure timeline.
- Please give us reasonable time to release a fix before any public disclosure.

## Scope

In scope: the application code in this repository (server, templates, security
middleware, asset handling). Anything that could leak visitor data, weaken the
Content-Security-Policy, introduce tracking, set cookies, or make unexpected
external requests is especially relevant to this project's threat model.