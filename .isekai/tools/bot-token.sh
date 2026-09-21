#!/usr/bin/env bash
# Forge-agnostic entrypoint for this world's bot identity: prints a usable
# token to stdout and nothing else, regardless of which forge the calling
# repo lives on. Downstream usage is identical either way:
#   TOKEN=$(.isekai/tools/bot-token.sh)
#   GH_TOKEN="$TOKEN" gh pr create ...          # GitHub
#   git -c http.extraHeader="PRIVATE-TOKEN: $TOKEN" push ...   # GitLab
#
# The two forges' bot-identity mechanisms are genuinely different, not just
# differently-named the same thing, so this script doesn't paper over that -
# it just picks the right one based on which credentials are actually set,
# so a caller never needs to know or care which forge it's talking to.
#
# GitHub — GitHub App installation token, short-lived (~1h), re-minted every
# call via github-app-token.sh (JWT-signed with the App's private key).
# Needs: GITHUB_APP_ID, GITHUB_APP_PRIVATE_KEY_PATH, GITHUB_APP_INSTALLATION_ID
# (set up via .isekai/tools/setup-github-app.sh).
#
# GitLab — a Project or Group Access Token IS the credential directly, no
# minting step: GitLab creates a distinct bot user (e.g. project_123_bot_...)
# the moment the token is created in its UI/API, and the token stays valid
# until its own expiry (GitLab caps these at 1 year, renew before it lapses).
# Needs: GITLAB_BOT_TOKEN (the token value itself, saved wherever this
# world's credentials live for the machine in question — see
# .isekai/elf/core/primer-comms.md's "Bot identity" section).
set -euo pipefail

if [[ -n "${GITLAB_BOT_TOKEN:-}" ]]; then
  printf '%s\n' "$GITLAB_BOT_TOKEN"
  exit 0
fi

if [[ -n "${GITHUB_APP_ID:-}" && -n "${GITHUB_APP_PRIVATE_KEY_PATH:-}" && -n "${GITHUB_APP_INSTALLATION_ID:-}" ]]; then
  exec "$(dirname "${BASH_SOURCE[0]}")/github-app-token.sh"
fi

echo "error: no bot identity credentials found in the environment." >&2
echo "  GitHub: set GITHUB_APP_ID, GITHUB_APP_PRIVATE_KEY_PATH, GITHUB_APP_INSTALLATION_ID" >&2
echo "          (run .isekai/tools/setup-github-app.sh, then source its .env file)" >&2
echo "  GitLab: set GITLAB_BOT_TOKEN to a Project or Group Access Token's value" >&2
exit 1
