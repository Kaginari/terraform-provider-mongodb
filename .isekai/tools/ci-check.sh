#!/usr/bin/env bash
# CI/tooling drift instrument (Nature 9 — "build/CI status" is explicitly one of the raw
# signals isekai.md promises instruments capture). Never hand-authored: every run re-reads
# the live workflow files and, where `gh` is available, live run status. Output is a window,
# not a record — nothing here is meant to be committed; what it reveals that matters gets
# written into slime-ci's doc and log.md by whoever ran it.
#
# What it catches that a plain `go build`/`go vet`/`go test` pass never will: the actual
# lint/action tooling pinned in .github/workflows drifting out of the runner's supported range
# (see log.md 2026-09-21 — golangci-lint v1.30 + skip-go-installation against a runner whose
# default Go had drifted to 1.24.x, failing every PR with no code-level cause).
set -uo pipefail
cd "$(git rev-parse --show-toplevel 2>/dev/null || echo .)"

echo "== workflow files =="
workflow_dir=".github/workflows"
if [ ! -d "$workflow_dir" ]; then
  echo "  (none — no $workflow_dir directory)"
else
  for f in "$workflow_dir"/*.yml "$workflow_dir"/*.yaml; do
    [ -f "$f" ] || continue
    echo "-- $f"
    grep -nE "uses:|version:" "$f" | sed 's/^/     /'
  done
fi

echo
echo "== pin staleness heuristic (eyeball, not gospel) =="
echo "  A pin like @v2 on an action, or a linter 'version: vX.Y' more than ~2 years old, is"
echo "  the exact shape that bit this repo once already — flag it for a human/session look,"
echo "  don't assume it's fine just because CI was green the day it was written."
if [ -d "$workflow_dir" ]; then
  grep -hoE "uses: [a-zA-Z0-9_.\/-]+@v[0-9]+" "$workflow_dir"/*.yml 2>/dev/null | sort -u | sed 's/^/  /'
fi

echo
echo "== local golangci-lint vs pinned config =="
if command -v golangci-lint >/dev/null 2>&1; then
  echo "  local golangci-lint: $(golangci-lint --version 2>&1 | head -1)"
  if [ -f .golangci.yml ]; then
    if golangci-lint run ./... >/tmp/ci-check-lint.$$ 2>&1; then
      echo "  .golangci.yml loads and run is clean against the local toolchain"
    else
      echo "  .golangci.yml FAILED to load or run — see below"
      sed 's/^/    /' /tmp/ci-check-lint.$$
    fi
    rm -f /tmp/ci-check-lint.$$
  fi
else
  echo "  golangci-lint not on PATH — can't cross-check .golangci.yml locally."
  echo "  (install: https://golangci-lint.run/welcome/install/#local-installation)"
fi

echo
echo "== live CI status (needs gh auth) =="
if command -v gh >/dev/null 2>&1 && gh auth status >/dev/null 2>&1; then
  gh run list --limit 5 2>&1 | sed 's/^/  /'
else
  echo "  gh not available/authenticated — skipping live run status"
fi
