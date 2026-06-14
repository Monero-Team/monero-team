#!/usr/bin/env sh
# Design-token regression guard.
#
# Fails the build if app.css drifts from the locked design fundament
# (docs/design-tokens.md). It guards the few invariants that previously
# regressed — the single canonical accent and the border width — so a wrong
# palette cannot land again, in CI or locally.
#
# Checks (all against internal/web/assets/app.css):
#   1. --c-orange is defined as #FF6B1A (case-insensitive).
#   2. --accent is bound to var(--c-orange) — not a literal, not something else.
#   3. --accent-2 does not exist (there is exactly one accent).
#   4. --accent is defined exactly once (no shadow redefinition).
#   5. the retired accent #f60000 does not appear anywhere.
#   6. --border-width is 0.5px.

set -eu

CSS="internal/web/assets/app.css"

fail=0
err() { printf 'FAIL: %s\n' "$1"; fail=1; }

[ -f "$CSS" ] || { echo "FAIL: $CSS not found"; exit 1; }

# 1. Canonical accent value.
orange_line=$(grep -iE -- '--c-orange[[:space:]]*:' "$CSS" || true)
if ! printf '%s' "$orange_line" | grep -iqE '#FF6B1A'; then
  err "--c-orange must be #FF6B1A (got: ${orange_line:-<missing>})"
fi

# 2. --accent must alias the canonical orange.
accent_line=$(grep -E -- '--accent[[:space:]]*:' "$CSS" || true)
if ! printf '%s' "$accent_line" | grep -qE 'var\([[:space:]]*--c-orange[[:space:]]*\)'; then
  err "--accent must be bound to var(--c-orange) (got: ${accent_line:-<missing>})"
fi

# 3. No second accent.
if grep -qE -- '--accent-2' "$CSS"; then
  err "--accent-2 must not exist (single canonical accent only)"
fi

# 4. Exactly one --accent definition.
accent_defs=$(grep -cE -- '--accent[[:space:]]*:' "$CSS" || true)
if [ "$accent_defs" -ne 1 ]; then
  err "expected exactly one --accent definition, found $accent_defs"
fi

# 5. Retired accent must be gone.
if grep -iqE '#f60000' "$CSS"; then
  err "retired accent #f60000 must not appear"
fi

# 6. Border width locked at 0.5px.
bw_line=$(grep -E -- '--border-width[[:space:]]*:' "$CSS" || true)
if ! printf '%s' "$bw_line" | grep -qE '(^|[^0-9.])0\.5px'; then
  err "--border-width must be 0.5px (got: ${bw_line:-<missing>})"
fi

if [ "$fail" -ne 0 ]; then
  echo
  echo "Design tokens drifted from the locked fundament (docs/design-tokens.md)."
  echo "Fix app.css to match the canonical values; do not edit the guard."
  exit 1
fi

echo "OK: design tokens match the locked canonical values."
