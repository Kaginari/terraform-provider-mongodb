#!/bin/bash
# context-check.sh -- Nature 9 (Perception) instrument: is THIS session's context window
# approaching its budget? Not a guess -- reads the running session's own transcript, the
# same way tempest.js reads real usage, and reports a measured number.
#
# Method: Claude Code resends the full conversation on every turn, so the most recent
# assistant message's own usage (input_tokens + cache_read + cache_creation) is the current
# context occupancy, not a cumulative sum across turns (that's a different, much larger
# number -- total spend, not window size; see tempest.js's harvestClaudeUsage for that one).
# This is an estimate, not exact -- cross-checked against Claude Code's own /context command
# to within ~7% on 2026-09-20. Good enough to catch the stress zone, not a billing figure.
#
# Usage: ./context-check.sh [--limit N] [--stress N]
#   --limit N   nominal context budget in tokens (default 200000)
#   --stress N  stress-zone threshold in tokens (default 180000, 90% of limit)
set -e
LIMIT=200000
STRESS=180000
while [ $# -gt 0 ]; do
  case "$1" in
    --limit) LIMIT="$2"; shift 2 ;;
    --stress) STRESS="$2"; shift 2 ;;
    *) shift ;;
  esac
done

SLUG=$(pwd | sed 's/[^a-zA-Z0-9]/-/g')
PROJDIR="$HOME/.claude/projects/$SLUG"
if [ ! -d "$PROJDIR" ]; then
  echo "no Claude Code project transcripts found for $(pwd) -- nothing to check"
  exit 0
fi
LATEST=$(ls -t "$PROJDIR"/*.jsonl 2>/dev/null | head -1)
if [ -z "$LATEST" ]; then
  echo "no transcript found under $PROJDIR"
  exit 0
fi

python3 -c "
import json, sys

limit, stress = $LIMIT, $STRESS
last_usage = None
with open('$LATEST') as fh:
    for line in fh:
        line = line.strip()
        if not line: continue
        try: d = json.loads(line)
        except: continue
        if d.get('type') != 'assistant': continue
        u = d.get('message', {}).get('usage')
        if u: last_usage = u

if not last_usage:
    print('no assistant turns with usage data yet -- nothing to check')
    sys.exit(0)

tin = (last_usage.get('input_tokens', 0) + last_usage.get('cache_read_input_tokens', 0)
       + last_usage.get('cache_creation_input_tokens', 0))
pct = 100 * tin / limit

print(f'context occupancy (estimate): {tin:,} / {limit:,} tokens ({pct:.0f}%)')
if tin >= stress:
    print()
    print('STRESS ZONE -- past the stress threshold.')
    print('Relief, in order:')
    print('  1. Write anything not yet durable to .isekai/log.md or the relevant doc now --')
    print('     do not let a compaction or restart be the first time it gets recorded.')
    print('  2. Any remaining heavy work (a large read, a big diff, a long investigation)')
    print('     belongs dispatched to a Court Body from here on, not run inline -- its')
    print('     context dies with the task; only the terse wire report returns.')
    print('  3. Recommend to the human, plainly: this is a good point for a break, a')
    print('     /clear, or a fresh session picking up from what log.md now holds.')
elif tin >= stress * 0.75:
    print('approaching the stress zone -- no action needed yet, worth noting.')
else:
    print('within budget.')
"
