#!/bin/bash
# run-opencode.sh -- run the with/without-convention comparison via OpenCode.
# Requires: `opencode` CLI on PATH. See README.md for the full methodology.
#
# Usage: ./run-opencode.sh <trials-dir> [model]
#   <trials-dir>  output of setup-trials.sh
#   [model]       provider/model string, default: opencode/big-pickle (free)
#
# To point at an on-prem model: configure it as an OpenCode provider first (OpenCode
# supports any OpenAI-compatible endpoint -- Ollama, vLLM, LM Studio, etc. -- via
# `opencode auth login` or its config file), then pass that provider/model string here,
# e.g. ./run-opencode.sh <trials-dir> ollama/llama3.1:70b
#
# Session continuity note: unlike Claude Code, OpenCode does NOT continue a session by
# default across separate `opencode run` invocations -- each call starts fresh unless you
# pass -c/--continue or -s <sessionID>. This script captures the session ID from task 1's
# output and explicitly passes it to task 2 for that reason.
set -e
DIR="${1:?Usage: run-opencode.sh <trials-dir> [model]}"
MODEL="${2:-opencode/big-pickle}"
P1="$(cat "$DIR/prompt.txt")"
P2="$(cat "$DIR/prompt2.txt")"

for trial in "$DIR"/with-* "$DIR"/without-*; do
  label="$(basename "$trial")"
  echo "=== $label: task 1 ==="
  out1=$(cd "$trial" && opencode run "$P1" -m "$MODEL" --title "oc-task1-$label" --format json)
  echo "$out1" > "$DIR/raw-task1-${label}.jsonl"
  sid=$(echo "$out1" | head -1 | python3 -c "import json,sys; print(json.load(sys.stdin)['sessionID'])")
  echo "session=$sid"
  ( cd "$trial" && opencode export "$sid" ) > "$DIR/export-task1-${label}.json"
  echo "=== $label: task 2 (continue $sid) ==="
  out2=$(cd "$trial" && opencode run "$P2" -m "$MODEL" -s "$sid" --format json)
  echo "$out2" > "$DIR/raw-task2-${label}.jsonl"
  ( cd "$trial" && opencode export "$sid" ) > "$DIR/export-task2-${label}.json"
  echo "=== $label done ==="
done
echo "ALL OPENCODE TRIALS COMPLETE"
