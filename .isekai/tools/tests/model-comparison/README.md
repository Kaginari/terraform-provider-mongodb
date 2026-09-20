# Model comparison test — does the convention actually help?

A reusable, deterministic test package (Nature 9: compute it, don't eyeball it) that answers
two separate questions, kept separate on purpose:

1. **Correctness/completeness** — does a session with `.isekai/` present produce a better
   fix than one without? (a controlled with/without comparison, graded by objective
   pass/fail criteria, not a read of the diff)
2. **Cost** — does having `.isekai/` present cost more or less, both for one isolated task
   and across a sequence of chained tasks in one session (where prompt-cache economics
   matter)?

First run: 2026-09-20, on `terraform-provider-mongodb`, via `claude -p` (Sonnet 5). Findings
are logged in that world's own `.isekai/log.md` (entries at 22:36 and 22:43) — read those for
the actual numbers and the honest, mixed verdict (better fix, ~3x higher cost, gap doesn't
shrink across a sequence). This README documents the *method* so it can be re-run — on a
different repo, a different task, a different model, or a different harness — not just once.

## The design

- `setup-trials.sh` makes N isolated copies of a target repo per arm: N with `.isekai/`
  present (plus an `AGENTS.md`/`CLAUDE.md` pointer telling the agent to read it — this is
  what a real deployment would look like, not just an unread directory), N with `.isekai/`,
  `.opencode/`, `.claude/` stripped entirely.
- Each trial runs the **same two prompts**, task 1 then task 2 chained onto the same session
  (`claude -c` / `opencode -s <sessionID>`), so cost can be measured both cold (task 1) and
  amortized (task 2's marginal cost).
- The prompts describe a bug's *shape* without naming the fix, so a trial has to actually
  search — a prompt that names the fix defeats the point of testing whether pre-existing
  knowledge (the Traits doc) shortens that search.
- Grading is objective: automatable pass/fail checks (does it compile, does it use the right
  pattern), never a subjective "looks better" call.

## Running it

```bash
# 1. Set up trial directories (default: 3 per arm)
./setup-trials.sh /path/to/target-repo /path/to/output-dir [trials-per-arm]

# 2a. Run via Claude Code
./run-claude.sh /path/to/output-dir [model-alias]   # default: sonnet

# 2b. Run via OpenCode (free models, or an on-prem model once configured)
./run-opencode.sh /path/to/output-dir [provider/model]   # default: opencode/big-pickle

# 3. Grade correctness (example task only -- see below)
./grade-example.sh /path/to/output-dir
```

Cost/token numbers come straight from each run's output:
- Claude: `result-<label>.json` / `result2-<label>.json` — `total_cost_usd`, `usage.*`
- OpenCode: `export-task1-<label>.json` / `export-task2-<label>.json` — `info.tokens.*`
  (no dollar cost on free models — compare token counts, not `$`)

## Using this on-prem, later

`run-opencode.sh` takes a `provider/model` string as its second argument. Once an on-prem
model is registered as an OpenCode provider (OpenCode speaks any OpenAI-compatible endpoint —
Ollama, vLLM, LM Studio, etc. — configured via `opencode auth login` or its config file), point
the script at it the same way:

```bash
./run-opencode.sh /path/to/output-dir ollama/llama3.1:70b
```

Nothing else about the test changes. If the on-prem model is meaningfully weaker than Sonnet
5, expect the correctness gap (with vs. without) to matter *more*, not less — a pre-written
Traits doc replaces search a weaker model is less likely to complete correctly on its own.
Worth also tracking wasted/unproductive turns on a weaker model specifically (failed tool
calls, redundant re-reads, edits that get corrected) — `raw-task*-<label>.jsonl` has the full
per-event stream needed to count those; this package doesn't automate that count yet since it
depends on what "wasted" means for your task, but the raw data is captured either way.

## Mechanical difference worth knowing before you run this on OpenCode

**Session continuity is not automatic in OpenCode the way it is in Claude Code.** In Claude
Code, prompt caching and multi-turn continuity happen by default within one `claude` session —
no flag needed. In OpenCode, every `opencode run` invocation starts a **new** session unless
you explicitly pass `-c`/`--continue` or `-s <sessionID>` — `run-opencode.sh` does this for
you (captures task 1's session ID, passes it explicitly to task 2), but if you're scripting
something ad hoc, don't assume continuity without that flag.

## Adapting the task

`task1-prompt.txt`/`task2-prompt.txt` and `grade-example.sh` are specific to the bug this test
was first built against (`slime-db-role`'s `Update` in `terraform-provider-mongodb`). To reuse
this package on a different repo/task:

1. Find a real, documented bug in a world that has actual populated Slime/Orc Traits (run
   `/genesis --depth deep` first if the world isn't populated yet — the Traits section is
   what task 1's "with" advantage actually comes from; an empty or shallow Traits doc makes
   this a test of nothing).
2. Write a new prompt describing the bug's shape without naming the fix.
3. Write new grading criteria — objective, automatable, same shape as `grade-example.sh`.
