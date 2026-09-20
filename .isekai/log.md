# Isekai Chronicle — change log

Append-only. Newest entries at the bottom. One entry per change.

## Entry format

    ### [YYYY-MM-DD HH:MM] <author> — <short title>
    - **Task:** what was asked
    - **Files:** paths changed (or `none`)
    - **Gate:** <orc-name> pass | fail — reason   (or `n/a`)
    - **Result:** done | partial | failed
    - **Learned:** anything the ranks above should keep

---

### [2026-09-20T18:58:28+02:00] rimuru — World reincarnated
- **Task:** /isekai terraform-provider-mongodb
- **Files:** .isekai/isekai.md, .isekai/log.md, .isekai/{portraits,instruments,elf,orc,slime,tmp}/
- **Gate:** n/a
- **Result:** done
- **Learned:** The world is reincarnated. Read isekai.md before any work.

### [2026-09-20T19:11:59+02:00] rimuru — Population born via /genesis
- **Task:** /genesis (survey the code and birth Elves, Orcs and Slimes on observed need)
- **Files:** .isekai/orc/provider/README.md, .isekai/slime/config/README.md,
  .isekai/slime/db-role/README.md, .isekai/slime/db-user/README.md
- **Gate:** n/a
- **Result:** done
- **Learned:** Small single-package Terraform provider (4 Go files, 862 lines) — one coherent
  domain, so one Orc (orc-provider, ruling `provider.go`/`main.go`/`docs/index.md` directly
  under Rimuru) with three Slimes underneath: slime-config (`config.go`, the connection/auth
  layer), slime-db-role and slime-db-user (one Terraform resource each, paired with its
  registry doc). No Elf was needed — a single-domain world doesn't require one.

### [2026-09-20T19:20:00+02:00] rimuru — Traits deepened after actually reading the code
- **Task:** User challenged the first /genesis pass ("can you capture all app aspects in the
  best optimised way") — the population's boundaries were structurally sound but every
  `## Traits` section was empty because the survey had classified by file names, doc mirrors
  and commit counts only, never by reading `mongodb/*.go` itself.
- **Files:** .isekai/orc/provider/README.md, .isekai/slime/config/README.md,
  .isekai/slime/db-role/README.md, .isekai/slime/db-user/README.md
- **Gate:** n/a
- **Result:** done
- **Learned:** Reading the actual source surfaced real, non-obvious ground truth a
  structural-only survey misses entirely: `insecure_skip_verify` silently no-ops unless
  `certificate` is also set; connect timeout is hardcoded at 10s; slime-db-role's `Update`
  deletes via a raw `system.roles` collection write while its `Delete` uses the proper
  `dropRole` command (two removal paths for one operation); slime-db-user's `Read` re-sets
  `password` to itself (`data.Set("password", data.Get("password"))`), so password drift
  outside Terraform is never detected; both resources share a copy-pasted
  `base64(db + "." + name)` ID scheme with identical single-`.`-split fragility. Fixed
  `/genesis` itself (all 4 copies — global/project × Claude Code/OpenCode) to require actually
  reading candidate zones in step 3, and to treat an empty `## Traits` section on non-trivial
  code as a sign the survey stopped short, not an acceptable outcome.

### [2026-09-20T19:30:00+02:00] rimuru — Law amended: Confirmation and Escalation (Veldora's order)
- **Task:** Veldora asked whether Rimuru normalizing/confirming ambiguous asks, and a
  confusion-escalation chain between ranks (Slime→Orc→Elf→Rimuru→Veldora), were already
  harnessed in the convention. They weren't — the existing "context down / memory up"
  Principles cover analysis flow (Nature 5), not real-time ambiguity or noise. Veldora ordered
  both added (Law 6: `isekai.md` changes only on Veldora's order).
- **Files:** .isekai/isekai.md (this world's copy, backported); also added to all 4 command
  templates (~/.claude/commands/isekai.md, convention-jura/.claude/commands/isekai.md,
  ~/.config/opencode/commands/isekai.md, convention-jura/.opencode/commands/isekai.md) so
  future `/isekai` runs get it too.
- **Gate:** n/a
- **Result:** done
- **Learned:** Added `### III. Confirmation and Escalation` under Absolute Rules, right after
  "II. Language — the wire": (1) Rimuru's input duty — confirm with Veldora in full human
  language rather than guess-and-push when an ask is ambiguous or self-contradictory; (2)
  escalation on confusion — a Slime/Orc/Elf that hits noise it can't resolve asks its
  immediate parent only, one hop at a time (never skips a rank, never goes sideways), until it
  resolves or reaches Rimuru→Veldora. This is a new channel distinct from Nature 5's
  analysis-flows-up — that one carries what was learned, this one carries what wasn't
  understood.

### [2026-09-20T19:40:00+02:00] rimuru — Deep survey: git archaeology on existing population
- **Task:** /genesis --depth deep. Population already existed (4 creatures, no new zones), so
  per the convention this run touches no new creatures and births nothing — instead it deepens
  the 4 existing docs with evolution facts, shown to Veldora rather than applied silently.
- **Files:** .isekai/orc/provider/README.md, .isekai/slime/config/README.md,
  .isekai/slime/db-role/README.md, .isekai/slime/db-user/README.md
- **Gate:** n/a
- **Result:** done
- **Learned (cross-zone pattern — flows up to orc-provider):** A fix landed on one sibling
  function has failed to propagate to its counterpart, twice. (1) `5044b99` ("fix: Use
  dropRole to delete role", Jan 2022) fixed slime-db-role's `Delete` to use the proper
  `dropRole` command but never touched its `Update`, which still does a raw
  `admin.system.roles` collection delete. (2) `79bfcf4` ("address issue #3", Feb 2021) fixed
  the identical hardcoded-`admin`-database bug on slime-db-user's `Update` a year earlier, and
  that fix was never carried over to slime-db-role at all. Net result: slime-db-role's
  `Update` today carries a known, twice-already-fixed-elsewhere bug — the single most
  actionable finding of this survey, and a two-line fix (mirror `Delete`'s `dropRole` call,
  using the role's own `database` field). Also found: the "reconnect on every CRUD call"
  pattern (`MongoClientInit`) was a deliberate 2021 fix for a "provider plan issue," not
  sloppiness — don't revert it without understanding why; `insecure_skip_verify` has been
  scoped to only work alongside `certificate` since the day it was introduced (2021), not a
  regression; certificate handling was fully redesigned once already (file-path-based →
  PEM-string-based), so that area should not be assumed stable.

### [2026-09-20T20:17:00+02:00] rimuru — Instrument board lit (Nature 9)
- **Task:** Copy `tempest.js` (the instrument board, built in convention-jura from the canon
  spec) into this world and start it — this world had an empty `.isekai/instruments/` with no
  actual instrument, since the operative `/isekai` command doesn't install tempest.js yet
  (that's still canon Phase 2b material, unmerged).
- **Files:** .isekai/tools/tempest.js (new, copied verbatim from convention-jura)
- **Gate:** n/a
- **Result:** done
- **Learned:** Board running at http://localhost:7799/terraform-provider-mongodb/ (pid via
  `.isekai/instruments/board.pid`). 30-min idle auto-sleep; `node .isekai/tools/tempest.js .
  --ensure` re-lights it. This world now has a real instrument, not just an empty directory —
  Nature 9 ("the instruments stay lit") is actually satisfied here now, not just declared.

### [2026-09-20T20:28:44+02:00] rimuru — elf-core born; genesis.md's Elf rule fixed (Veldora's order)
- **Task:** Veldora ordered /genesis to always require at least one Elf, even a single-Orc
  world — the original "zero Elves for a single-domain world" heuristic contradicted
  isekai.md's own Principles (Rimuru → Elf → Orc → Slime, fixed chain, no skip-the-Elf
  exception). Fixed genesis.md (all 4 copies — global/project × Claude Code/OpenCode), then
  applied the fix here retroactively.
- **Files:** .isekai/elf/core/README.md (new), .isekai/orc/provider/README.md
  (`**Reports to:**` corrected from Rimuru to elf-core)
- **Gate:** n/a
- **Result:** done
- **Learned:** This world now has the correct 3-tier shape: elf-core (Rimuru) →
  orc-provider (elf-core) → slime-config/db-role/db-user (orc-provider). tempest.js's board
  should now also show elf-core once refreshed, since its harvest() was separately fixed
  today to read this world's `.isekai/elf/*/README.md` shape.

### [2026-09-20T20:31:00+02:00] rimuru — Neural view bug: elf was found by hardcoded name, not race
- **Task:** Human reported "still can't see the elf in neural" after elf-core's birth was
  already confirmed present in the board's JSON data. Root cause: `render()`'s neural-layout
  code looked up the Elf node via `byName['elf-colony']` — a literal, hardcoded creature name
  left over from whatever demo world `tempest.js` was built/tested against — instead of
  finding it generically by `race === 'elf'`. Since this world's Elf is named `elf-core`, the
  lookup always returned `undefined` and the whole `if (elfC)` block was silently skipped: no
  node, no edge, no error. Same bug for the dark elf (`byName['darkelf-rust']`).
- **Files:** .isekai/tools/tempest.js (fix applied here first, then copied back to
  convention-jura — the canonical source — so the fix isn't lost on the next copy-down)
- **Gate:** n/a
- **Result:** done
- **Learned:** Both hardcoded lookups replaced with `d.creatures.filter(c => c.race ===
  'elf' | 'darkelf')`, spread vertically like the existing orc/slime layout if a world ever
  has more than one. Confirmed live: `id="n-elf-core"` and `class="elfedge e-elf-core
  e-orc-provider"` both present in the served SVG. This is the second hardcoded-to-a-specific-
  world assumption found in tempest.js today (the first was `.opencode/skills/` +
  `AGENTS.md`-only population discovery) — worth a closer read of the rest of `render()` for
  more of the same pattern before trusting the board's other panels at face value.

### [2026-09-20T20:49:00+02:00] rimuru — tempest is now one app for the whole machine
- **Task:** Veldora: "should be 1 app in whole machine not multiple and use sub path and
  global dashboard give over all overview of all worlds" — previously every world ran its
  own `tempest.js` process on its own port (convention-jura on one port, this world on
  another). Fixed in convention-jura (the canonical source), then copied here.
- **Files:** .isekai/tools/tempest.js (replaced — same file, new architecture)
- **Gate:** n/a
- **Result:** done
- **Learned:** One process now, fixed port 7799, registered worlds tracked in
  `~/.local/share/tempest/registry.json` (machine-global, mirrors opencode's own session
  store). `/` is a new global dashboard listing every registered world with live census +
  health; each world lives at its own `/<name>/` sub-path with a "← all worlds" link back.
  `tempest --ensure` in this world now registers it and joins the shared daemon rather than
  spawning a dedicated process. Confirmed live: one process served both convention-jura and
  this world simultaneously, elf-core still rendered correctly, unknown world names 404.
  Also: the race color palette was re-validated against computed OKLCH/CVD/contrast checks
  (dark and light modes each got their own re-stepped hues instead of reusing identical hex)
  — see convention-jura's commit `cb9d976` for the full before/after. A new Mind,
  `palette-audit`, packages that procedure for reuse, shipped in both `.opencode/skills/`
  and `.claude/skills/` in convention-jura.

### [2026-09-20T21:05:00+02:00] rimuru — Real global observability + Minds as graph nodes
- **Task:** Veldora: the global dashboard was empty — wanted machine-wide Claude/OpenCode
  consumption + active-session signal, the cast/manifest tables merged with a stress score,
  the click-to-see-description bug fixed, and (separately) `palette-audit` visible in this
  world's own neural graph, linked to whichever creatures it's actually relevant to
  ("harmony rule").
- **Files:** .isekai/tools/tempest.js (synced from convention-jura), .opencode/skills/palette-audit/,
  .claude/skills/palette-audit/ (both new — copied in, weren't here before)
- **Gate:** n/a
- **Result:** done
- **Learned:** Fixed a real bug: `focus()`'s zdata blob never carried `desc`, so clicking a
  creature could never show one no matter what the doc said — now it does, HTML-escaped.
  Cast + manifest merged into one table with an explicit stress-score column. The global `/`
  now reports real machine-wide Claude Code usage (5 sessions found on this machine at time
  of writing, all-time tokens, per-model, "active now" via file-mtime recency) and the same
  for OpenCode when its store exists (none found on this machine). Minds are no longer lumped
  in as race:'plain' creatures — `palette-audit` now renders as its own dashed node on a
  "worn, not raced" shelf, linked to creatures whose doc mentions it or that it mentions back.
  Currently **zero links** here, correctly: nothing in this world's creature docs names
  `palette-audit` and it names none of them back — this world is a Terraform provider
  backend, not a UI, so there's genuinely no fit yet. That's the harmony rule working
  honestly, not a bug to chase — a link should appear only when a real textual reason exists.

### [2026-09-20T21:30:00+02:00] rimuru — Stress-relief loop was one-directional; CLAUDE.md closes it
- **Task:** Veldora reported that after a `/clear` (following advice from the stress-relief
  instrument, `.isekai/tools/context-check.sh`), the next session had forgotten everything —
  the mid-session stress relief system "isn't working properly." Root cause: the instrument's
  relief steps tell a stressed session to write everything durable to `log.md` before a
  `/clear`, but nothing told the *next* session to read `isekai.md`/`log.md` back. This world
  had no `CLAUDE.md` anywhere — confirmed absent here, and absent even in convention-jura, the
  canonical reference world, so this was a gap in the whole convention, not a one-off. Claude
  Code auto-loads `CLAUDE.md` on every session start in a directory, including immediately
  after `/clear` — that is the actual mechanism "the world remembers in documents, because
  sessions forget" was always relying on, and it was never wired up.
- **Files:** CLAUDE.md (new, this world), plus the genome fix (Nature 3 — a command's own
  procedure, not `isekai.md` itself, so no Law 6 gate applies): `~/.claude/commands/isekai.md`
  and `~/.config/opencode/commands/isekai.md` both now write/append a `CLAUDE.md` pointer as
  founding step 7, with a template matching this world's new file.
- **Gate:** n/a
- **Result:** done
- **Learned:** The write-half of the relief loop (Instruments' "write anything not yet durable
  before a `/clear`") was always sound; the read-half (a fresh session actually reading it
  back) had no attachment point until now. Not fixed retroactively in `convention-jura` or any
  other already-founded world — only the founding template and this world, since touching
  another world's files is outside this session's territory (Containment) without being asked.
  Any world founded before this fix should get a `CLAUDE.md` added by hand or via a re-run of
  `/isekai` against it (step 2's "create only what's missing" already covers this case).

### [2026-09-20T23:45:00+02:00] rimuru — AGENTS.md added (OpenCode); fix extended to convention-jura on Veldora's ask
- **Task:** Veldora: "it should work on opencode too" then "not here also in convention dir" —
  two follow-ups on the prior entry. (1) The prior fix only wired Claude Code's `CLAUDE.md`;
  OpenCode reads its own `AGENTS.md` (confirmed live — `tempest.js`'s harvest already reads
  `AGENTS.md` for routing), so the read-half was still missing for OpenCode sessions. (2)
  Veldora then asked for the same fix in `convention-jura`, overriding the prior entry's
  Containment-based restraint (Law 1: Veldora's word is law).
- **Files:** AGENTS.md (new, this world — a symlink to `CLAUDE.md`, not a second copy, so the
  two tools' pointers can never drift apart). Global templates
  `~/.claude/commands/isekai.md` and `~/.config/opencode/commands/isekai.md` updated again:
  founding step 7 now also creates `AGENTS.md` as a symlink to `CLAUDE.md`. In `convention-jura`
  (`/home/monta/convention-jura`, the canonical reference world — confirmed via `git remote -v`
  as the real one; `~/kaginari/convention-jura` is an empty, non-git directory of the same name
  and was left untouched): its own `CLAUDE.md` and `AGENTS.md` created the same way, and its
  local `.claude/commands/isekai.md` / `.opencode/commands/isekai.md` copies synced from the
  now-current global templates. Logged there too, in its own `log.md`.
- **Gate:** n/a
- **Result:** done
- **Learned:** Caught and fixed a mistake made while drafting the OpenCode template fix
  earlier this session: it claimed "OpenCode itself has no equivalent auto-loaded file," which
  is false. Corrected in the same change rather than left to repeat (Nature 3). Also: this
  session's shell is sandboxed to this world's directory — `cd` into `convention-jura` silently
  resets back here, but `git -C <path>` and absolute-path file operations work fine, so
  cross-world work (once actually authorized by Veldora) is done that way, not via `cd`.

### [2026-09-20T23:55:00+02:00] rimuru — tempest daemon relaunched under Node 22; Fable usage traced to a request, not a run
- **Task:** Veldora saw the global board's "OpenCode unreadable — `node:sqlite` needs Node
  22.5+, this process is running on v16.20.2" and separately asked why a Fable session's usage
  wasn't showing anywhere, only `claude-sonnet-5`'s.
- **Files:** none (operational + investigative — no code changed). The live daemon (pid 109631,
  system `/usr/bin/node` v16.20.2) was killed and relaunched via nvm's Node 22.23.2
  (`/home/monta/.nvm/versions/node/v22.23.2/bin/node tempest.js --ensure`). Confirmed live: the
  global OpenCode card now reads real data (12 runs, `big-pickle` / `nemotron-3.5-lightning-free`
  models) instead of the unreadable-error panel.
- **Gate:** n/a
- **Result:** done
- **Learned:**
  - **The Node-version gotcha will recur** if `--ensure` is ever invoked through a plain
    `node` on PATH again — the daemon's self-respawn (`spawn(process.execPath, ...)`) inherits
    whatever Node started it, so it stays on 22 only as long as every future ensure-call also
    runs under 22. This machine's default `node` resolves to `/usr/bin/node` (v16.20.2); v22.23.2
    exists only under nvm. Worth a follow-up if this trips again — not fixed at the root here
    since that would mean changing how `--ensure` locates a Node binary, an unrequested
    tempest.js change.
  - **No real Fable usage exists in any transcript on this machine, in any world.** Traced the
    only exact `"model":"fable"` byte match on the whole machine to this very session's own
    transcript, and it isn't a completed turn's `message.model` — it's a `tool_use` **call**
    (`Agent(model:"fable", description:"Fable reviews the Isekai convention", ...)`) requesting
    a subagent on that model. No assistant turn anywhere (main chain or sidechain) carries
    `model: claude-fable-5-1` or `model: fable`. Every other "fable" hit machine-wide, across
    5 other transcripts, is the boilerplate system-reminder text listing available model IDs,
    a git log line, or a tool-schema enum — not usage. So tempest isn't losing/dropping real
    Fable tokens: the source data it reads (`~/.claude/projects/**/*.jsonl`) never recorded a
    completed Fable turn to begin with. Could not determine *why* the requested override never
    produced a turn (errored before responding, or ran but was folded into the parent's
    `claude-sonnet-5` accounting) — that's Claude Code's own internals, outside what a
    transcript read can settle, and is named as an open `@?` rather than guessed at.
