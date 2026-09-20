#!/bin/bash
# run-claude.sh -- run the with/without-convention comparison via claude -p (Claude Code).
# Requires: `claude` CLI authenticated. See README.md for the full methodology.
#
# Usage: ./run-claude.sh <trials-dir> [model]
#   <trials-dir>  output of setup-trials.sh
#   [model]       claude model alias, default: sonnet
set -e
DIR="${1:?Usage: run-claude.sh <trials-dir> [model]}"
MODEL="${2:-sonnet}"
P1="$(cat "$DIR/prompt.txt")"
P2="$(cat "$DIR/prompt2.txt")"

for trial in "$DIR"/with-* "$DIR"/without-*; do
  label="$(basename "$trial")"
  echo "=== $label: task 1 ==="
  ( cd "$trial" && claude -p "$P1" --model "$MODEL" --output-format json --permission-mode acceptEdits ) \
    > "$DIR/result-${label}.json" 2> "$DIR/stderr-${label}.log"
  echo "=== $label: task 2 (continue) ==="
  ( cd "$trial" && claude -p "$P2" -c --model "$MODEL" --output-format json --permission-mode acceptEdits ) \
    > "$DIR/result2-${label}.json" 2> "$DIR/stderr2-${label}.log"
  echo "=== $label done ==="
done
echo "ALL CLAUDE TRIALS COMPLETE"
