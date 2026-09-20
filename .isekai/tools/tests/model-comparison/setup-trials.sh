#!/bin/bash
# setup-trials.sh -- create N with-convention / N without-convention isolated copies of a
# target repo, for the model-comparison test. See README.md in this directory for the full
# methodology and how to interpret results.
#
# Usage: ./setup-trials.sh <source-repo> <output-dir> [trials-per-arm]
set -e
SRC="${1:?Usage: setup-trials.sh <source-repo> <output-dir> [trials-per-arm]}"
OUT="${2:?Usage: setup-trials.sh <source-repo> <output-dir> [trials-per-arm]}"
N="${3:-3}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

rm -rf "$OUT"
mkdir -p "$OUT"

for i in $(seq 1 "$N"); do
  for cond in with without; do
    dst="$OUT/${cond}-${i}"
    cp -r "$SRC" "$dst"
    rm -rf "$dst/.git"
    if [ "$cond" = "without" ]; then
      rm -rf "$dst/.isekai" "$dst/.opencode" "$dst/.claude"
    else
      # AGENTS.md is OpenCode's project-instructions convention; CLAUDE.md is Claude Code's.
      # Ship both so either harness picks it up without extra flags.
      cat > "$dst/AGENTS.md" <<'EOF'
This directory follows the Isekai convention. Read `.isekai/isekai.md` first, then check
`.isekai/{elf,orc,slime}/*/README.md` for any creature whose territory covers the files you're
about to touch -- their `## Traits` sections hold ground truth about this codebase worth
knowing before you change it.
EOF
      cp "$dst/AGENTS.md" "$dst/CLAUDE.md"
    fi
  done
done
cp "$SCRIPT_DIR/task1-prompt.txt" "$OUT/prompt.txt"
cp "$SCRIPT_DIR/task2-prompt.txt" "$OUT/prompt2.txt"
echo "Created $((N * 2)) trial dirs under $OUT ($N with-, $N without-)."
