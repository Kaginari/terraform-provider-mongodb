#!/usr/bin/env bash
# Mints a short-lived (1h) GitHub App installation access token and prints it
# to stdout — nothing else, so it's safe to capture directly:
#   GH_TOKEN=$(.isekai/tools/github-app-token.sh)
#   gh pr create ...          # now authenticates as the App's bot identity
#   git commit --author="kaginari-mongodb-bot <...>" ...
#
# Portable on purpose: plain bash + openssl + curl + jq only. No Claude Code
# or OpenCode specific tooling — any harness that can run a shell command can
# run this, which is the actual point (see .isekai/elf/core/primer-comms.md,
# "Bot identity" section).
#
# Requires three values, either as env vars or the three positional args
# below (env vars win if both are set). None of these are committed anywhere
# in this repo — see the primer for where they actually live on this machine.
#   GITHUB_APP_ID              — the App's numeric ID
#   GITHUB_APP_PRIVATE_KEY_PATH — path to the App's downloaded .pem key
#   GITHUB_APP_INSTALLATION_ID — the App's installation ID on this org/repo
set -euo pipefail

APP_ID="${GITHUB_APP_ID:-${1:-}}"
PRIVATE_KEY_PATH="${GITHUB_APP_PRIVATE_KEY_PATH:-${2:-}}"
INSTALLATION_ID="${GITHUB_APP_INSTALLATION_ID:-${3:-}}"

if [[ -z "$APP_ID" || -z "$PRIVATE_KEY_PATH" || -z "$INSTALLATION_ID" ]]; then
  echo "usage: GITHUB_APP_ID=... GITHUB_APP_PRIVATE_KEY_PATH=... GITHUB_APP_INSTALLATION_ID=... $0" >&2
  echo "       (or positionally: $0 <app-id> <private-key-path> <installation-id>)" >&2
  exit 1
fi
if [[ ! -f "$PRIVATE_KEY_PATH" ]]; then
  echo "error: private key not found at $PRIVATE_KEY_PATH" >&2
  exit 1
fi

b64url() { openssl base64 -A | tr '+/' '-_' | tr -d '='; }

now=$(date +%s)
header='{"alg":"RS256","typ":"JWT"}'
payload=$(printf '{"iat":%d,"exp":%d,"iss":"%s"}' "$((now - 60))" "$((now + 600))" "$APP_ID")

signing_input="$(printf '%s' "$header" | b64url).$(printf '%s' "$payload" | b64url)"
signature=$(printf '%s' "$signing_input" | openssl dgst -sha256 -sign "$PRIVATE_KEY_PATH" | b64url)
jwt="${signing_input}.${signature}"

response=$(curl -s -X POST \
  -H "Authorization: Bearer $jwt" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/app/installations/${INSTALLATION_ID}/access_tokens")

token=$(printf '%s' "$response" | jq -r '.token // empty')
if [[ -z "$token" ]]; then
  echo "error: no token in response — $response" >&2
  exit 1
fi

printf '%s\n' "$token"
