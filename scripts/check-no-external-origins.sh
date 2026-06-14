#!/usr/bin/env sh
# Privacy guard: fail if any template or stylesheet loads an external resource.
#
# This enforces the project's core promise — the base reading path makes no
# external requests. It checks *loaded* resources (stylesheets, fonts, scripts,
# images, CSS url()), NOT plain navigation links (<a href>), which are allowed
# to point anywhere (e.g. the GitHub link in the footer).
#
# Exit non-zero on the first violation.

set -eu

# Directories that contain rendered markup and styles.
SEARCH_DIRS="internal/web/templates internal/web/assets"

fail=0
note() { printf '  %s\n' "$1"; }

# 1) Known CDN / hosted-font providers — banned anywhere in markup/CSS.
banned='fonts\.googleapis\.com|fonts\.gstatic\.com|cdnjs|jsdelivr|unpkg\.com|cdn\.|googleapis\.com|googletagmanager|google-analytics'

# 2) Loaded resources pointing at an absolute http(s) origin.
loaders='<link[^>]+href="https?://|<script[^>]+src="https?://|<img[^>]+src="https?://|@font-face|url\(\s*["'"'"']?https?://'

for dir in $SEARCH_DIRS; do
  [ -d "$dir" ] || continue

  if grep -RniE "$banned" "$dir" >/dev/null 2>&1; then
    echo "FAIL: external CDN / hosted-font / analytics reference found:"
    grep -RniE "$banned" "$dir" | while IFS= read -r line; do note "$line"; done
    fail=1
  fi

  if grep -RniE "$loaders" "$dir" 2>/dev/null \
      | grep -iE 'https?://' >/dev/null 2>&1; then
    echo "FAIL: a stylesheet/font/script/image is loaded from an external origin:"
    grep -RniE "$loaders" "$dir" 2>/dev/null | grep -iE 'https?://' \
      | while IFS= read -r line; do note "$line"; done
    fail=1
  fi
done

if [ "$fail" -ne 0 ]; then
  echo
  echo "Base reading path must load no external resources. Self-host instead."
  exit 1
fi

echo "OK: no external origins in markup or styles."
