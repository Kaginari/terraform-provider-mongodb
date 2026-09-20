// tempest — the world's instrument panel (ISEKAI nature law 8 — Instruments;
// tooling law: stdlib node only, no build chain, the shape travels with the world).
// Re-harvests per request. Many worlds run at once: each board answers at
//   http://localhost:<port>/<world-name>/
// the world names itself in .isekai/name (jura-style names lawful), the port is
// derived from the world's root so two boards never collide by default.
// Lifecycle law: the board SLEEPS WHEN THE WORLD RESTS — any request renews the
// lease (--ensure pulses it), after --ttl minutes of silence it dies; --immortal
// opts out (a human's long watch); --stop posts it to sleep now.
// 🎉 holidays also DISPATCHES sequential creature-hat relief runs (opencode run)
// — the tool plans and launches, hat-sessions write the minds; the tool never does.
// Usage: node tempest.js [worldRoot] [--port N] [--ttl minutes] [--immortal] [--json] [--ensure] [--stop]
//
// Prerequisites: plain Node for everything except reading OpenCode's own usage —
// that one path (harvestOpencodeGlobal, the global dashboard's OpenCode card) needs
// `node:sqlite`, a Node 22.5+ builtin. On an older system Node (this machine shipped
// 16.20.2, confirmed 2026-09-20) that card still renders — dbExists still gets
// checked with plain fs — but `available` comes back false with an honest `error`
// instead of a silent zero (see the comment at harvestOpencodeGlobal's definition).
// Fix: install Node 22.5+ (e.g. `nvm install 22`) and run tempest.js with that Node;
// everything else about tempest.js keeps working unmodified either way.
//
// Ported from photographs of the source (2026-09-20) — see .isekai/canon/README.md
// for how much of the original was actually visible. Two deliberate departures from
// what's shown, both kept and flagged inline where they occur:
//   1. Relief dispatch tries `opencode run` first, falls back to `claude -p` — the
//      source only ever named opencode, which would strand Claude-only machines.
//   2. The HTTP route table (GET /<name>/, POST /holidays, POST /party, GET /relief,
//      GET /pulse, POST /shutdown) was never photographed — everything past the
//      start of the --ensure probe logic is this session's own completion, built to
//      satisfy exactly what the (fully visible) client-side script calls.
// Everything else below — harvest(), render(), the relief queue/harm-fence, the
// SVG neural-layer graph, the SQLite session-store reads — is a faithful port.

const fs = require('fs');
const path = require('path');
const http = require('http');
const os = require('os');
const { execSync, spawn, spawnSync } = require('child_process');

const args = process.argv.slice(2);
const JSON_MODE = args.includes('--json');
const JSON_GLOBAL_MODE = args.includes('--json-global');
const COLONY = path.resolve(args.find(a => !a.startsWith('--') && isNaN(+a)) || '.');
// One app for the whole machine (human order 2026-09-20: "should be 1 app in whole
// machine not multiple"), not one process + one hash-derived port per world. A fixed
// default port; --port still overrides for the rare clash.
const portIdx = args.indexOf('--port');
const PORT = portIdx > -1 ? parseInt(args[portIdx + 1], 10) : 7799;

// World home + canon resolution (the reincarnation law): the world's living
// memory rides .isekai/; an elder world's shelf may still be .convention-zero/.
// The canon doc is ISEKAI.md, else the elder CONVENTION-ZERO.md / SLIME.md.
// Computed per-root, not module-level — one process now serves every registered
// world, never just the COLONY it happened to be launched with.
const homeOf = root => fs.existsSync(path.join(root, '.isekai')) ? '.isekai' : '.convention-zero';
const canonOf = root => ['ISEKAI.md', 'CONVENTION-ZERO.md', 'SLIME.md']
  .find(f => fs.existsSync(path.join(root, f))) || 'ISEKAI.md';
const journalPath = (root, home) =>
  ['log.md', 'log-git.md'].map(f => path.join(root, home, f)).find(fs.existsSync)
  || path.join(root, home, 'log.md');

const RACES = ['slime', 'orc', 'elf', 'highorc', 'darkelf', 'kijin'];
const DIET_KB = 6;                    // rule 14 breath law
const DESK_LIMIT = n => n.startsWith('darkelf') ? 10 : 5; // archive law vs ground races
const esc = s => String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
const rd = f => { try { return fs.readFileSync(f, 'utf8'); } catch { return ''; } };
// A YAML frontmatter `description: "..."` captures its quotes literally with a bare .+
// regex — strip one matching pair so the extracted text reads the same whether the source
// quoted it or not.
const unquote = s => (s || '').replace(/^"(.*)"$/, '$1').replace(/^'(.*)'$/, '$1');

// The world name (nature law 8 naming): the home's `name` file holds one line —
// jura-style names lawful, chosen at birth/populate; it travels (rule 12 re-include).
// Also per-root now, for the same reason home/canon are.
const nameOf = (root, home) => {
  const raw = rd(path.join(root, home, 'name')) || path.basename(root);
  return String(raw).trim().toLowerCase().replace(/[^a-z0-9-]+/g, '-').replace(/^-+|-+$/g, '') || 'world';
};

// ------------ the registry: which worlds this one app knows about ------------
// Machine-global, like opencode's own session store — not any one world's .isekai/,
// since the whole point is one app spanning every world. { path, name, lastSeen }[].
const REGISTRY_DIR = path.join(os.homedir(), '.local', 'share', 'tempest');
const REGISTRY_FILE = path.join(REGISTRY_DIR, 'registry.json');
function readRegistry() {
  try { return JSON.parse(rd(REGISTRY_FILE) || '[]'); } catch { return []; }
}
function writeRegistry(list) {
  fs.mkdirSync(REGISTRY_DIR, { recursive: true });
  fs.writeFileSync(REGISTRY_FILE, JSON.stringify(list, null, 2));
}
// Upsert root into the registry (by path) and prune any entry whose .isekai/ (or
// .convention-zero/) is gone — self-healing when a world is moved or deleted.
function registerWorld(root) {
  const home = homeOf(root);
  const name = nameOf(root, home);
  const list = readRegistry().filter(w => fs.existsSync(path.join(w.path, w.home || '.isekai')) || fs.existsSync(path.join(w.path, '.convention-zero')));
  const i = list.findIndex(w => w.path === root);
  const entry = { path: root, name, home, lastSeen: new Date().toISOString() };
  if (i > -1) list[i] = entry; else list.push(entry);
  writeRegistry(list);
  return entry;
}
function prunedRegistry() {
  const list = readRegistry().filter(w => fs.existsSync(path.join(w.path, w.home || homeOf(w.path))));
  writeRegistry(list);
  return list;
}
function worldRootForName(name) {
  const w = prunedRegistry().find(w => w.name === name);
  return w ? w.path : null;
}

// Lifecycle law: the board sleeps when the world rests. Any request (for any
// registered world) renews the one shared lease; --ttl minutes of silence → sleep.
// --immortal opts out.
const ttlIdx = args.indexOf('--ttl');
const TTL_MIN = ttlIdx > -1 ? parseFloat(args[ttlIdx + 1]) : 30;
const TTL_MS = TTL_MIN * 60000;
const IMMORTAL = args.includes('--immortal');
let lastTouch = Date.now();

// ------------ Claude Code usage (no opencode.db — read its JSONL transcripts instead) ------------
// Claude Code has no SQLite session store; each session is one JSONL transcript at
// ~/.claude/projects/<slugified-cwd>/<sessionId>.jsonl. Assistant turns carry
// message.model, message.usage.{input,output}_tokens and a timestamp; d.cwd is matched
// against root with the same LIKE-prefix semantics as the opencode query below.
// Best-effort only: unlike opencode's `agent` column (a precise mounted-body name),
// Claude's transcripts don't cleanly name which minted sub-agent handled a turn, so agent
// identity here is coarse — d.isSidechain: 'subagent' vs 'main session' — not per-body.
// Read beats inferring (see /genesis): this is honestly labeled as coarser, not silently
// passed off as equally precise.
function harvestClaudeUsage(root) {
  const out = { live: { rows: 0, totalIn: 0, totalOut: 0, perDay: {}, perModel: {}, series: {} },
    agentUse: {} };
  const projectsDir = path.join(os.homedir(), '.claude', 'projects');
  if (!fs.existsSync(projectsDir)) return out;
  const rootPrefix = root.endsWith(path.sep) ? root : root + path.sep;
  let slugDirs;
  try { slugDirs = fs.readdirSync(projectsDir, { withFileTypes: true }).filter(e => e.isDirectory()).map(e => e.name); }
  catch { return out; }
  for (const slug of slugDirs) {
    let files;
    try { files = fs.readdirSync(path.join(projectsDir, slug)).filter(f => f.endsWith('.jsonl')); }
    catch { continue; }
    for (const f of files) {
      let text;
      try { text = fs.readFileSync(path.join(projectsDir, slug, f), 'utf8'); }
      catch { continue; }
      for (const line of text.split('\n')) {
        if (!line) continue;
        let d;
        try { d = JSON.parse(line); } catch { continue; }
        if (d.type !== 'assistant') continue;
        const cwd = d.cwd || '';
        if (cwd !== root && !cwd.startsWith(rootPrefix)) continue;
        const msg = d.message || {};
        const usage = msg.usage || {};
        const mid = msg.model || 'unknown';
        // usage.input_tokens alone is only the *uncached* sliver of a turn's real input —
        // found 2026-09-20 after a human asked why "in" showed 0.0M next to a 1.2M "out":
        // with prompt caching (the normal case for any multi-turn Claude Code session),
        // almost all real input is cache_read_input_tokens (context reused from earlier
        // turns) or cache_creation_input_tokens (context newly cached this turn) — both
        // silently excluded before. On this world alone that was a ~177M-token undercount,
        // not a rounding error.
        const tin = (usage.input_tokens || 0) + (usage.cache_read_input_tokens || 0) + (usage.cache_creation_input_tokens || 0);
        const tout = usage.output_tokens || 0;
        const day = String(d.timestamp || '').slice(0, 10) || 'undated';
        const agentKey = d.isSidechain ? 'subagent' : 'main session';

        out.live.rows++; out.live.totalIn += tin; out.live.totalOut += tout;
        const pd = out.live.perDay[day] ||= { in: 0, out: 0 };
        pd.in += tin; pd.out += tout;
        const pm = out.live.perModel[mid] ||= { in: 0, out: 0, rows: 0, agents: [] };
        pm.in += tin; pm.out += tout; pm.rows++;
        if (!pm.agents.includes(agentKey)) pm.agents.push(agentKey);
        const sd = (out.live.series[mid] ||= {})[day] ||= [0, 0, 0];
        sd[0] += tin; sd[1] += tout; sd[2]++;

        const row = out.agentUse[agentKey] ||= { sessions: 0, totIn: 0, totOut: 0, models: [], lastDay: '' };
        row.sessions++; row.totIn += tin; row.totOut += tout;
        if (!row.models.includes(mid)) row.models.push(mid);
        if (day > row.lastDay) row.lastDay = day;
      }
    }
  }
  return out;
}
// Mind usage: how many times each Mind (a Claude Code Skill tool_use, `input.skill`) was
// actually invoked in this world's own transcripts — not just whether it exists. Human order
// 2026-09-20: "show on tempest the skills and how much they are used". Same cwd-prefix
// scoping as harvestClaudeUsage(root); a skill invoked from a subagent still counts (isSidechain
// doesn't change which Mind ran).
function harvestSkillUsage(root) {
  const counts = {};
  const projectsDir = path.join(os.homedir(), '.claude', 'projects');
  if (!fs.existsSync(projectsDir)) return counts;
  const rootPrefix = root.endsWith(path.sep) ? root : root + path.sep;
  let slugDirs;
  try { slugDirs = fs.readdirSync(projectsDir, { withFileTypes: true }).filter(e => e.isDirectory()).map(e => e.name); }
  catch { return counts; }
  for (const slug of slugDirs) {
    let files;
    try { files = fs.readdirSync(path.join(projectsDir, slug)).filter(f => f.endsWith('.jsonl')); }
    catch { continue; }
    for (const f of files) {
      let text;
      try { text = fs.readFileSync(path.join(projectsDir, slug, f), 'utf8'); }
      catch { continue; }
      for (const line of text.split('\n')) {
        if (!line) continue;
        let d;
        try { d = JSON.parse(line); } catch { continue; }
        if (d.type !== 'assistant') continue;
        const cwd = d.cwd || '';
        if (cwd !== root && !cwd.startsWith(rootPrefix)) continue;
        const content = (d.message || {}).content;
        if (!Array.isArray(content)) continue;
        for (const block of content) {
          if (block && block.type === 'tool_use' && block.name === 'Skill' && block.input && block.input.skill) {
            const name = block.input.skill.split(':').pop(); // strip plugin/dir-scope prefix
            counts[name] = (counts[name] || 0) + 1;
          }
        }
      }
    }
  }
  return counts;
}
// ------------ machine-wide activity (for the global / index — not scoped to one world) ------------
// Human order 2026-09-20: "how much consumption in total if opencode session ... maybe
// information if session is active or not". Unlike harvestClaudeUsage(root)/the opencode.db
// query in harvest(root), these are NOT filtered to one colony — every session on this
// machine, whether or not its directory is a reincarnated world.
const ACTIVE_WINDOW_MS = 5 * 60000; // "active" = touched in the last 5 minutes
function harvestClaudeGlobal() {
  const out = { sessions: 0, active: 0, totalIn: 0, totalOut: 0, perModel: {} };
  const projectsDir = path.join(os.homedir(), '.claude', 'projects');
  if (!fs.existsSync(projectsDir)) return out;
  const now = Date.now();
  let slugDirs;
  try { slugDirs = fs.readdirSync(projectsDir, { withFileTypes: true }).filter(e => e.isDirectory()).map(e => e.name); }
  catch { return out; }
  for (const slug of slugDirs) {
    let files;
    try { files = fs.readdirSync(path.join(projectsDir, slug)).filter(f => f.endsWith('.jsonl')); }
    catch { continue; }
    for (const f of files) {
      const fp = path.join(projectsDir, slug, f);
      let stat; try { stat = fs.statSync(fp); } catch { continue; }
      out.sessions++;
      if (now - stat.mtimeMs <= ACTIVE_WINDOW_MS) out.active++;
      let text; try { text = fs.readFileSync(fp, 'utf8'); } catch { continue; }
      for (const line of text.split('\n')) {
        if (!line) continue;
        let d; try { d = JSON.parse(line); } catch { continue; }
        if (d.type !== 'assistant') continue;
        const usage = (d.message || {}).usage || {};
        const mid = (d.message || {}).model || 'unknown';
        // See harvestClaudeUsage(root)'s comment: input_tokens alone excludes cache
        // reads/writes, which is almost all of it under prompt caching.
        const tin = (usage.input_tokens || 0) + (usage.cache_read_input_tokens || 0) + (usage.cache_creation_input_tokens || 0);
        const tout = usage.output_tokens || 0;
        out.totalIn += tin; out.totalOut += tout;
        const pm = out.perModel[mid] ||= { in: 0, out: 0 };
        pm.in += tin; pm.out += tout;
      }
    }
  }
  return out;
}
// OpenCode: only the columns already proven against a real store elsewhere in this file
// (model, tokens_input, tokens_output, time_created) — no session-id column is used
// anywhere else here, so "sessions" isn't claimed, only rows and recent-row activity.
function harvestOpencodeGlobal() {
  // Nature 9: an instrument that's gone silent is itself a finding, not a thing to route
  // around. `dbExists` (the store is there) and `available` (we could actually read it) are
  // deliberately separate — found the hard way 2026-09-20: node:sqlite is a Node 22.5+
  // built-in, so on any older system Node (this machine ships 16.20.2), `require('node:sqlite')`
  // throws and a REAL opencode.db with real sessions in it was silently reported as "OpenCode
  // not in use here" — indistinguishable from the file genuinely not existing. That's exactly
  // the failure mode this Nature warns against.
  const out = { rows: 0, activeRows: 0, totalIn: 0, totalOut: 0, perModel: {}, dbExists: false, available: false, error: null };
  const dbp = path.join(os.homedir(), '.local', 'share', 'opencode', 'opencode.db');
  out.dbExists = fs.existsSync(dbp);
  if (!out.dbExists) return out;
  try {
    const { DatabaseSync } = require('node:sqlite');
    const db = new DatabaseSync(dbp, { readOnly: true });
    out.available = true;
    try {
      const sinceMs = Date.now() - ACTIVE_WINDOW_MS;
      // Same undercount class as harvestClaudeUsage/harvestClaudeGlobal above: tokens_input
      // alone is only the uncached sliver of a turn's real input. Found 2026-09-20 by the
      // deterministic test in test-opencode-integration.js — tokens_cache_read/_write must
      // be summed in too, or a session showing 11,008 real cache-read tokens in `opencode
      // export` reports as if it cost 51.
      for (const r of db.prepare(
        `SELECT COALESCE(model,'') m, COUNT(*) runs,
                COALESCE(SUM(tokens_input + tokens_cache_read + tokens_cache_write),0) tin,
                COALESCE(SUM(tokens_output),0) tout,
                SUM(CASE WHEN time_created >= ? THEN 1 ELSE 0 END) recent
         FROM session GROUP BY m`).all(sinceMs)) {
        let mid = 'unknown';
        try { mid = JSON.parse(r.m).id || 'unknown'; } catch { mid = String(r.m) || 'unknown'; }
        out.rows += r.runs; out.activeRows += r.recent || 0;
        out.totalIn += r.tin; out.totalOut += r.tout;
        const pm = out.perModel[mid] ||= { in: 0, out: 0 };
        pm.in += r.tin; pm.out += r.tout;
      }
    } finally { db.close(); }
  } catch (e) { out.error = e.message; }
  return out;
}
function mergeLive(into, from) {
  into.rows += from.rows; into.totalIn += from.totalIn; into.totalOut += from.totalOut;
  for (const [day, v] of Object.entries(from.perDay)) {
    const pd = into.perDay[day] ||= { in: 0, out: 0 };
    pd.in += v.in; pd.out += v.out;
  }
  for (const [mid, v] of Object.entries(from.perModel)) {
    const pm = into.perModel[mid] ||= { in: 0, out: 0, rows: 0, agents: [] };
    pm.in += v.in; pm.out += v.out; pm.rows += v.rows;
    for (const a of v.agents) if (!pm.agents.includes(a)) pm.agents.push(a);
  }
  for (const [mid, days] of Object.entries(from.series || {})) {
    const s = into.series[mid] ||= {};
    for (const [day, arr] of Object.entries(days)) {
      const sd = s[day] ||= [0, 0, 0];
      sd[0] += arr[0]; sd[1] += arr[1]; sd[2] += arr[2];
    }
  }
}
function mergeAgentUse(into, from) {
  for (const [agent, v] of Object.entries(from)) {
    const row = into[agent] ||= { sessions: 0, totIn: 0, totOut: 0, models: [], lastDay: '' };
    row.sessions += v.sessions; row.totIn += v.totIn; row.totOut += v.totOut;
    for (const m of v.models) if (!row.models.includes(m)) row.models.push(m);
    if (v.lastDay > row.lastDay) row.lastDay = v.lastDay;
  }
}

// ------------ harvest ------------
function harvest(root) {
  const home = homeOf(root), canonFile = canonOf(root), worldName = nameOf(root, home);
  const skillsDir = path.join(root, '.opencode', 'skills');
  const docOf = d => path.join(skillsDir, d, 'SKILL.md');
  const dirs = fs.existsSync(skillsDir)
    ? fs.readdirSync(skillsDir, { withFileTypes: true }).filter(e => e.isDirectory()).map(e => e.name).sort()
    : [];
  // A .opencode/skills/<name>/ entry is a creature only if its name carries a race
  // prefix (orc-, slime-, ...) — the "real source" shape where creatures live here.
  // Anything else is a genuine Mind (reusable know-how, not a creature): e.g.
  // palette-audit. Harmony rule (human order 2026-09-20): a Mind links to whichever
  // creatures its own doc names, or whose doc names it back — never assigned by hand.
  const creatures = [];
  const minds = [];
  for (const name of dirs) {
    const doc = rd(docOf(name));
    const race = RACES.find(r => name.startsWith(r + '-'));
    const bytes = Buffer.byteLength(doc);
    const desc = unquote((doc.match(/^description:\s*(.+)$/m) || [])[1] || '');
    if (!race) { minds.push({ name, kb: +(bytes / 1024).toFixed(1), desc, doc, links: [] }); continue; }
    // thoughts: section-scoped dated bullets (>- ## Thoughts until next ## or EOF)
    const m = doc.match(/##\s*Thoughts([\s\S]*?)(?=\n##\s|\n#\s|$)/i);
    const tBody = m ? m[1] : '';
    const thoughtLines = tBody.split('\n').filter(l => /^\s*(-|###)/.test(l) && /\d{4}-\d{2}-\d{2}/.test(l));
    const thoughtDates = thoughtLines.map(l => (l.match(/\d{4}-\d{2}-\d{2}/) || [])[0]).filter(Boolean);
    creatures.push({ name, race, kb: +(bytes / 1024).toFixed(1), thoughts: thoughtLines.length,
      thoughtDates, limit: DESK_LIMIT(name), desc, doc });
  }

  // operative /isekai + /genesis shape: .isekai/{elf,orc,slime}/<name>/*.md — no
  // .opencode/skills/ or AGENTS.md required. Merged in alongside any skills-shaped
  // creatures above (not a replacement), so either shape, or both at once, renders.
  const isekaiOrcCommands = {}; // orc name -> [slime names], read off its own "Commands:" line
  for (const race of ['elf', 'orc', 'slime']) {
    const raceDir = path.join(root, home, race);
    if (!fs.existsSync(raceDir)) continue;
    for (const zone of fs.readdirSync(raceDir, { withFileTypes: true })
      .filter(e => e.isDirectory()).map(e => e.name).sort()) {
      const name = `${race}-${zone}`;
      if (creatures.some(c => c.name === name)) continue; // skills-shaped already has it
      const zoneDir = path.join(raceDir, zone);
      const mdFile = ['README.md', ...fs.readdirSync(zoneDir).filter(f => f.endsWith('.md'))]
        .find(f => fs.existsSync(path.join(zoneDir, f)));
      if (!mdFile) continue;
      const doc = rd(path.join(zoneDir, mdFile));
      const bytes = Buffer.byteLength(doc);
      const m = doc.match(/##\s*Thoughts([\s\S]*?)(?=\n##\s|\n#\s|$)/i);
      const tBody = m ? m[1] : '';
      const thoughtLines = tBody.split('\n').filter(l => /^\s*(-|###)/.test(l) && /\d{4}-\d{2}-\d{2}/.test(l));
      const thoughtDates = thoughtLines.map(l => (l.match(/\d{4}-\d{2}-\d{2}/) || [])[0]).filter(Boolean);
      const desc = unquote((doc.match(/^description:\s*(.+)$/m) || [])[1]
        || (doc.match(/^-\s*\*\*Purpose:\*\*\s*(.+)$/m) || [])[1] || '');
      if (race === 'orc') {
        const cmds = (doc.match(/^-\s*\*\*Commands:\*\*\s*(.+)$/m) || [])[1] || '';
        isekaiOrcCommands[name] = cmds.split(',').map(s => s.trim()).filter(Boolean);
      }
      creatures.push({ name, race, kb: +(bytes / 1024).toFixed(1), thoughts: thoughtLines.length,
        thoughtDates, limit: DESK_LIMIT(name), desc, doc });
    }
  }
  // crosslink index: mentions of other creature names (dir name or map alias) inside a doc
  const aliases = {}; // dir -> [names it answers to]
  const cz = rd(path.join(root, canonFile));
  const ag = rd(path.join(root, 'AGENTS.md'));
  for (const c of creatures) aliases[c.name] = [c.name];
  for (const hit of (cz + ag).matchAll(/`([a-z0-9][a-z0-9-]+)`/g)) {
    const bare = hit[1];
    for (const c of creatures)
      if (bare.endsWith(c.name) && bare !== c.name) (aliases[c.name] ||= []).push(bare); // ui-design ↔ benchforge-ui-design
  }
  // Mind ↔ creature harmony links, computed here while c.doc/mind.doc still exist:
  // bidirectional name-mention, exactly the same "does the text actually say so"
  // rule crosslinks use between creatures — never a hand-assigned pairing.
  for (const mind of minds) {
    mind.links = creatures.filter(c => mind.doc.includes(c.name) || c.doc.includes(mind.name)).map(c => c.name);
  }
  const skillUses = harvestSkillUsage(root);
  for (const mind of minds) mind.uses = skillUses[mind.name] || 0;
  for (const c of creatures) {
    c.crosslinks = creatures.filter(o => o.name !== c.name)
      .reduce((n, o) => n + aliases[o.name].reduce((k, al) =>
        k + (c.doc.split(al).length - 1), 0), 0);
    delete c.doc;
    c.stressPct = Math.round(100 * c.thoughts / c.limit);
    c.dietPct = Math.round(100 * c.kb / DIET_KB);
  }
  const median = xs => xs.length ? xs.slice().sort((a, b) => a - b)[Math.floor(xs.length / 2)] : 0;
  const linkMed = median(creatures.map(c => c.crosslinks));
  for (const c of creatures)
    c.genesisSignal = c.stressPct >= 80 && c.crosslinks >= Math.max(2, linkMed * 1.5); // heuristic, labeled in UI

  // architecture tree from the map's routing table
  const orcs = [];
  for (const l of ag.split('\n')) {
    const m = l.match(/\|\s*`?(orc-[^`|]+)`?\s*\|([^|]*)\|([^|]*)\|/);
    if (m) orcs.push({ orc: m[1], domain: m[2].trim(),
      slimes: [...m[3].matchAll(/`([^`]+)`/g)].map(x => x[1]).filter(s => s !== '-') });
  }
  // same tree, read straight off each isekai-shaped orc's own "Commands:" line —
  // no AGENTS.md required for the operative /isekai + /genesis shape.
  for (const [orc, slimes] of Object.entries(isekaiOrcCommands)) {
    if (!orcs.some(o => o.orc === orc)) orcs.push({ orc, domain: '', slimes });
  }
  // canon stamps
  const canonV = (cz.match(/canon v(\d+)/) || [])[1] || '?';
  const chartDebt = rd(path.join(root, home, 'assets', canonFile.replace(/\.md$/, '.drawio')))
    .match(/canon v(\d+)/);
  const chartV = chartDebt ? chartDebt[1] : null;

  // evolution series per day
  const days = {};
  const bump = (d, k) => { if (d) (days[d] ||= { journal: 0, commits: 0, thoughts: 0 })[k]++; };
  for (const l of rd(journalPath(root, home)).split('\n')) {
    // entry headers: isekai `### [YYYY-MM-DD HH:MM] being - title` and elder `# Log — date`
    const m = l.match(/^#{1,3}\s+(?:\[?(\d{4}-\d{2}-\d{2})|Log\s.*?\[?(\d{4}-\d{2}-\d{2}))/);
    if (m) bump(m[1] || m[2], 'journal');
  }
  for (const c of creatures) for (const d of c.thoughtDates) bump(d, 'thoughts');
  try {
    for (const l of execSync('git log --date=format:%Y-%m-%d --format=%ad', { cwd: root, encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] }).trim().split('\n'))
      bump(l, 'commits');
  } catch { /* not a git repo — series simply lacks commits */ }
  for (const c of creatures) { c.lastThought = c.thoughtDates.length ? c.thoughtDates.slice().sort().pop() : null; delete c.thoughtDates; }

  // model mounts (colony genome pins) + ledger sightings
  const models = {};
  // Bodies live natively at .opencode/agents/ (OpenCode) or .claude/agents/ (Claude Code,
  // via /mint) — both scanned and merged, since a world can have either or both.
  const agentDirs = [path.join(root, '.opencode', 'agents'), path.join(root, '.claude', 'agents')];
  const agentFiles = agentDirs.flatMap(d => fs.existsSync(d)
    ? fs.readdirSync(d).filter(f => f.endsWith('.md')).map(f => path.join(d, f)) : []);
  for (const f of agentFiles) {
    const mm = rd(f).match(/^model:\s*(\S+)/m);
    if (mm) (models[mm[1]] ||= { mounted: [], mentions: 0 }).mounted.push(path.basename(f, '.md'));
  }
  const ledger = rd(journalPath(root, home)) + '\n' + ag + '\n' + cz;
  for (const id of Object.keys(models)) models[id].mentions = ledger.split(id).length - 1;

  // session token ledger (rule: sessions append JSONL lines; the board reads).
  // A line may carry "model" (law extended 2026-09-15 — "usage token per model");
  // absent, the session's agent joins through the mount pins above; unmounted = alone.
  const agentModel = {};
  for (const [mid, mo] of Object.entries(models)) for (const a of mo.mounted) agentModel[a] = mid;
  const tokens = { rows: 0, totalIn: 0, totalOut: 0, perDay: {}, perModel: {} };
  for (const l of rd(path.join(root, home, 'metrics', 'tokens.jsonl')).split('\n').filter(Boolean)) {
    try {
      const j = JSON.parse(l); tokens.rows++;
      tokens.totalIn += +j.in || 0; tokens.totalOut += +j.out || 0;
      const day = String(j.ts || '').slice(0, 10) || 'undated';
      const pd = tokens.perDay[day] ||= { in: 0, out: 0 };
      pd.in += +j.in || 0; pd.out += +j.out || 0;
      const pmid = String(j.model || '') || agentModel[String(j.agent || '')] || 'unmounted';
      const pm = tokens.perModel[pmid] ||= { in: 0, out: 0, rows: 0, agents: [] };
      pm.in += +j.in || 0; pm.out += +j.out || 0; pm.rows++;
      const ag2 = String(j.agent || ''); if (ag2 && !pm.agents.includes(ag2)) pm.agents.push(ag2);
    } catch { /* a malformed line is noise, never a crash */ }
  }

  // live usage from opencode's own session store (human 2026-09-15: "i still
  // dont see the token usage" — the manual ledger was a law nobody fed). Truth
  // source: session.agent/model/tokens_*, colony-scoped by directory, opened
  // read-only. Absent db / unreadable schema → the manual ledger above stands.
  const live = { rows: 0, totalIn: 0, totalOut: 0, perDay: {}, perModel: {}, series: {} };
  try {
    const dbp = path.join(os.homedir(), '.local', 'share', 'opencode', 'opencode.db');
    if (fs.existsSync(dbp)) {
      const { DatabaseSync } = require('node:sqlite');
      const db = new DatabaseSync(dbp, { readOnly: true });
      try {
        for (const r of db.prepare(
          `SELECT COALESCE(model,'') m, COALESCE(agent,'main session') a, COUNT(*) runs,
                  COALESCE(SUM(tokens_input),0) tin, COALESCE(SUM(tokens_output),0) tout,
                  date(time_created/1000,'unixepoch') day
           FROM session WHERE directory LIKE ? GROUP BY m, a, day`).all(root + '%')) {
          let mid = 'unknown';
          try { mid = JSON.parse(r.m).id || 'unknown'; } catch { mid = String(r.m) || 'unknown'; }
          live.rows += r.runs; live.totalIn += r.tin; live.totalOut += r.tout;
          const pd = live.perDay[r.day || 'undated'] ||= { in: 0, out: 0 };
          pd.in += r.tin; pd.out += r.tout;
          const pm = live.perModel[mid] ||= { in: 0, out: 0, rows: 0, agents: [] };
          pm.in += r.tin; pm.out += r.tout; pm.rows += r.runs;
          if (!pm.agents.includes(r.a)) pm.agents.push(r.a);
          const sd = (live.series[mid] ||= {})[r.day || 'undated'] ||= [0, 0, 0];
          sd[0] += r.tin; sd[1] += r.tout; sd[2] += r.runs;
        }
      } finally { db.close(); }
    }
  } catch { /* an unreadable store is noise — the manual ledger still stands */ }
  // Claude Code's own session transcripts — merged additively, not a replacement, so a
  // machine running both ecosystems on the same world sees combined numbers.
  const claudeUsage = harvestClaudeUsage(root);
  mergeLive(live, claudeUsage.live);

  // ==== agents: minted bodies on disk + per-session breath (the agents ledger) ====
  const bodies = [];
  for (const f of agentFiles) {
    const t = rd(f);
    const claudeBody = f.includes(`${path.sep}.claude${path.sep}agents${path.sep}`);
    bodies.push({ name: path.basename(f, '.md'),
      model: (t.match(/^model:\s*(\S+)/m) || [null, '(inherits session)'])[1],
      // Claude Code sub-agents carry no `mode:` field (no primary/all distinction the way
      // OpenCode has) — every .claude/agents/ file is Court-shaped (Agent-tool invoked), so
      // default to 'subagent' there instead of an unhelpful '?'.
      mode: (t.match(/^mode:\s*(\S+)/m) || [null, claudeBody ? 'subagent' : '?'])[1],
      born: fs.statSync(f).birthtime.toISOString().slice(0, 10),
      kb: +(Buffer.byteLength(t) / 1024).toFixed(1) });
  }
  // per-session agent breath — opencode's store, read-only; avg context = the mean
  // token draw (in+out) per session answering as that agent.
  const agentUse = {};
  try {
    const dbp2 = path.join(os.homedir(), '.local', 'share', 'opencode', 'opencode.db');
    if (fs.existsSync(dbp2)) {
      const { DatabaseSync } = require('node:sqlite');
      const db = new DatabaseSync(dbp2, { readOnly: true });
      try {
        for (const r of db.prepare(
          `SELECT COALESCE(agent,'?') a, COALESCE(model,'') m, COUNT(*) n,
                  COALESCE(SUM(tokens_input),0) tin, COALESCE(SUM(tokens_output),0) tout,
                  MAX(date(time_created/1000,'unixepoch')) lastDay
           FROM session WHERE directory LIKE ? GROUP BY a, m`).all(root + '%')) {
          let mid = 'unknown';
          try { mid = JSON.parse(r.m).id || 'unknown'; } catch { mid = String(r.m) || 'unknown'; }
          const row = agentUse[r.a] ||= { sessions: 0, totIn: 0, totOut: 0, models: [], lastDay: '' };
          row.sessions += r.n; row.totIn += r.tin || 0; row.totOut += r.tout || 0;
          if (!row.models.includes(mid)) row.models.push(mid);
          if (r.lastDay > row.lastDay) row.lastDay = r.lastDay;
        }
      } finally { db.close(); }
    }
  } catch { /* store absent/unreadable — the ledger simply lacks usage */ }
  mergeAgentUse(agentUse, claudeUsage.agentUse);
  for (const u of Object.values(agentUse))
    u.avgCtx = u.sessions ? Math.round((u.totIn + u.totOut) / u.sessions) : 0;

  // the agents ledger — a metrics artifact regenerated when reality moves
  // (instruments write metrics, never minds — nature law 8). Populated by working.
  try {
    const fmtN = v => v >= 1e6 ? (v / 1e6).toFixed(1) + 'M' : v >= 1e3 ? (v / 1e3).toFixed(1) + 'k' : String(v);
    const ledgerMd = [
      `# Agents ledger — ${worldName} (${path.basename(root)})`,
      `Regenerated live by the instruments when reality moves. Bodies live on disk; breath lives in opencode's session store.`,
      ``,
      `## Minted bodies (.opencode/agents/)`,
      `| Agent | Mode | Mount | Born | KB |`, `|---|---|---|---|---|`,
      ...(bodies.length
        ? bodies.map(b => `| \`${b.name}\` | ${b.mode} | ${b.model} | ${b.born} | ${b.kb} |`)
        : ['| – | – | minds-only world (embodiment E2) | – | – |']),
      ``,
      `## Session breath per agent (avg context = mean tokens in+out per session)`,
      `| Agent | Sessions | Avg context | Total in | Total out | Mounts seen | Last day |`,
      `|---|---|---|---|---|---|---|`,
      ...(Object.keys(agentUse).length
        ? Object.entries(agentUse).sort((a, b) => (b[1].totIn + b[1].totOut) - (a[1].totIn + a[1].totOut))
          .map(([a, u]) => `| ${a} | ${u.sessions} | ${fmtN(u.avgCtx)} | ${fmtN(u.totIn)} | ${fmtN(u.totOut)} | ${u.models.join(', ') || '–'} | ${u.lastDay || '–'} |`)
        : ['| – | 0 | – | – | – | no sessions recorded in this world yet | – |']),
    ].join('\n');
    const mdir = path.join(root, home, 'metrics'); fs.mkdirSync(mdir, { recursive: true });
    const fp = path.join(mdir, 'agents-usage.md');
    if (rd(fp) !== ledgerMd) fs.writeFileSync(fp, ledgerMd);
  } catch { /* metrics unwritable — the panel still renders */ }

  for (const m of minds) delete m.doc;

  const census = {};
  for (const c of creatures) census[c.race] = (census[c.race] || 0) + 1;
  const stressed = creatures.filter(c => c.thoughts > c.limit || c.dietPct > 100)
    .map(c => `${c.name} (thoughts ${c.thoughts}/${c.limit}, ${c.kb}KB/${DIET_KB}KB)`);
  return { root, world: worldName, home, canonFile, port: PORT, bodies, agentUse, when: new Date().toISOString(), canonV, chartV: chartV ? +chartV : null,
    chartDebt: chartV !== null && +chartV !== +canonV, census, orcs, creatures, minds, days, models, tokens, live,
    genesisWatch: creatures.filter(c => c.genesisSignal).map(c => c.name),
    health: stressed.length ? stressed.join(' · ') : 'all minds within budget' };
}

// ------------ render ------------
// The constellation render (human 2026-09-14: "futuristic, worthy of an AI").
// Pure SVG + CSS — no scripts, no CDN, no fonts; the shape travels (rule 23).
// Layout deterministic: elf at the heart, orcs in orbit, slimes outer ring
// clustered by their orc, the dark elf burns apart. Node size = doc weight,
// aura = stress; a hot node breathes. Colors honor the colony's night themes.
// CSS custom properties, not raw hex — lets light/dark carry their own validated
// steps (see the :root / body.light palette comment) instead of one hex per race
// baked into server-rendered SVG regardless of theme.
const RCOL = { slime: 'var(--r-slime)', orc: 'var(--r-orc)', elf: 'var(--r-elf)', darkelf: 'var(--r-darkelf)', highorc: 'var(--r-highorc)', kijin: 'var(--r-kijin)', plain: 'var(--r-plain)', mind: 'var(--r-mind)' };
const aura = c => c.stressPct >= 100 ? 'hot' : c.stressPct >= 60 || c.dietPct >= 100 ? 'warn' : 'cool';

const PAGE_STYLE = `<style>
:root{--bg:#03060c;--ink:#c9d4e3;--dim:#616b79;--cy:#2695bd;--vi:#7c5cd6;--em:#d6402a;--gd:#bd8c24;--nodefill:#05070d;
--r-slime:#2695bd;--r-orc:#7c5cd6;--r-elf:#bd8c24;--r-darkelf:#d6402a;--r-highorc:#8a7300;--r-kijin:#c53d34;--r-plain:#616b79;--r-mind:#9fb0c3}
/* Palette re-tuned 2026-09-20 against the dataviz skill's validate_palette.js (OKLCH
   lightness band, CVD/normal-vision Delta E, WCAG contrast) — see .isekai/tmp/palette-check/.
   Was: identical hex reused for both themes, several near the lightness ceiling for a
   dark surface, and the core slime/orc/elf trio sat below the CVD normal-vision floor
   (worst pair Delta E 7.8, need >=15). Now: core trio passes every hard gate in both
   modes; darkelf/highorc/kijin (rare "born in time" races) keep a softer separation —
   a known-hard 3-way warm-hue constraint the skill's own reference palette hits past
   3-4 slots too — mitigated by the mandatory node label + tooltip every creature
   already carries (the skill's required secondary encoding for a WARN-band pair). */
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--ink);
font:13px/1.55 ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;overflow-x:hidden;padding:34px 28px 60px}
body::before{content:"";position:fixed;inset:0;pointer-events:none;z-index:0;
background:radial-gradient(900px 480px at 78% -8%,rgba(89,214,255,.09),transparent 62%),
           radial-gradient(760px 420px at 6% 10%,rgba(167,139,250,.08),transparent 60%),
           radial-gradient(1100px 700px at 50% 118%,rgba(255,122,89,.06),transparent 62%)}
body::after{content:"";position:fixed;inset:0;pointer-events:none;z-index:0;opacity:.35;
background:repeating-linear-gradient(0deg,transparent 0 3px,rgba(0,0,0,.14) 3px 4px)}
main{position:relative;z-index:1;max-width:1220px;margin:0 auto}
h1{font-size:15px;letter-spacing:.34em;text-transform:uppercase;margin:0;font-weight:600;color:#eaf2ff;
text-shadow:0 0 18px rgba(89,214,255,.5)}
h1 .sigil{color:var(--cy)}
.allworlds{display:inline-block;color:var(--dim);text-decoration:none;font-size:11px;letter-spacing:.14em;
text-transform:uppercase;margin-bottom:14px;border:1px solid #1a2436;border-radius:999px;padding:4px 14px;
background:rgba(13,20,32,.6)}
.allworlds:hover{color:var(--cy);border-color:var(--cy)}
body.light .allworlds{background:rgba(255,255,255,.8);border-color:#cfdae9}
.wcards{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:14px;margin-top:14px}
.wcard{display:block;text-decoration:none;color:inherit;border:1px solid #121c2e;border-radius:12px;
background:rgba(10,16,27,.5);padding:16px 18px;transition:border-color .2s,transform .2s}
.wcard:hover{border-color:var(--cy);transform:translateY(-2px)}
.wcard h3{margin:0 0 8px;font-size:13px;letter-spacing:.08em;color:#eaf2ff}
.wcard .wstat{font-size:11px;color:var(--dim);margin-top:4px}
.wcard .wcensus span{margin-right:10px}
body.light .wcard{background:rgba(255,255,255,.74);border-color:#dbe2ee}
body.light .wcard h3{color:#0b1424}
h2{font-size:11px;letter-spacing:.3em;text-transform:uppercase;color:var(--dim);margin:42px 0 10px;
border-bottom:1px solid #101826;padding-bottom:6px}
.meta{color:var(--dim);margin-top:8px}.chip{display:inline-block;border:1px solid #1a2436;border-radius:999px;
padding:2px 12px;margin-right:8px;background:rgba(13,20,32,.6)}
.ok{color:#3fb950}.debt{color:var(--em);text-shadow:0 0 10px rgba(255,122,89,.6)}
.mood{border:1px solid #1a2436;border-left:3px solid var(--em);background:rgba(16,24,38,.55);
padding:12px 16px;border-radius:0 8px 8px 0;letter-spacing:.02em}
.panel{display:block;width:100%;background:rgba(10,16,27,.5);border:1px solid #121c2e;border-radius:12px}
.node text{fill:var(--dim);font-size:10px;text-anchor:middle;letter-spacing:.06em}
.node .halo{opacity:.12}.node.warn .halo{opacity:.22}.node.hot .halo{opacity:.45}
.node.hot .core{fill:var(--em)}
.node.gs{filter:drop-shadow(0 0 6px rgba(242,214,117,.8))}
.node:hover .halo{opacity:.5}line{stroke:#22304a;stroke-width:.7}line.elfedge{stroke:#3a3a2a;stroke-dasharray:2 5}
line.mindedge{stroke:var(--r-mind);stroke-width:.9;stroke-dasharray:1 4;opacity:.55}
g.node{cursor:pointer;transition:transform .25s ease,opacity .25s ease;transform-box:fill-box;transform-origin:center}
body.focused g.node{opacity:.16}body.focused g.node.focus{opacity:1;transform:scale(1.6)}
body.focused line{opacity:.1}line.lit{opacity:1;stroke:var(--cy);stroke-width:1.4}
#focusPanel{padding:12px 16px;min-height:64px;letter-spacing:.02em}
#focusPanel b{color:#eaf2ff}#focusPanel .dim{color:var(--dim)}
tr[data-name]:hover{background:rgba(89,214,255,.06)}
.tbtns{float:right}.tbtns button{background:rgba(13,20,32,.6);border:1px solid #1a2436;color:var(--ink);
font:inherit;font-size:10px;letter-spacing:.18em;text-transform:uppercase;padding:6px 14px;border-radius:999px;
cursor:pointer;margin-left:8px}
.tbtns button.active{border-color:var(--em);color:var(--em);box-shadow:0 0 10px rgba(255,122,89,.35)}
.tbtns button.rng.active{border-color:var(--cy);color:var(--cy);box-shadow:0 0 10px rgba(89,214,255,.35)}
@keyframes blink{50%{opacity:.12}}
body.stressmode .node.hot{animation:blink 1.05s ease-in-out infinite}
body.stressmode .node.warn{animation:blink 1.9s ease-in-out infinite}
body.light{--bg:#edf1f7;--ink:#1c2634;--dim:#4b5568;--nodefill:#ffffff;
--cy:#1a7fa3;--vi:#5c3fc9;--em:#b8341f;--gd:#8a6600;
--r-slime:#1a7fa3;--r-orc:#5c3fc9;--r-elf:#8a6600;--r-darkelf:#b8341f;--r-highorc:#6b7400;--r-kijin:#a12e26;--r-plain:#4b5568;--r-mind:#546578}
/* Light steps are their own re-stepped values, not the dark hexes with the background
   flipped — the same ramps, validated separately against the light surface. */
body.light::before{background:radial-gradient(900px 480px at 78% -8%,rgba(30,140,200,.10),transparent 62%),
    radial-gradient(760px 420px at 6% 10%,rgba(120,90,220,.09),transparent 60%),
    radial-gradient(1100px 700px at 50% 118%,rgba(230,110,70,.08),transparent 62%)}
body.light::after{background:repeating-linear-gradient(0deg,transparent 0 3px,rgba(28,38,52,.045) 3px 4px);opacity:.5}
body.light .panel{background:rgba(255,255,255,.74);border-color:#dbe2ee}
body.light h1{color:#0b1424;text-shadow:none}
body.light line{stroke:#c6cfe0}body.light line.elfedge{stroke:#c9c39a}
body.light .mood{background:rgba(255,255,255,.74);border-color:#dbe2ee}
body.light .chip{background:rgba(255,255,255,.74);border-color:#dbe2ee}
body.light td,body.light th{border-color:#e6ebf4}
body.light .tbtns button{background:rgba(255,255,255,.8);border-color:#cfdae9;color:#1c2634}
.rib{filter:drop-shadow(0 0 5px currentColor)}.ax{fill:var(--dim);font-size:10px;letter-spacing:.12em}
table{border-collapse:collapse;width:100%}td,th{padding:5px 12px;border-bottom:1px solid #0e1624;text-align:left;
font-size:12px}th{color:var(--dim);font-weight:400;letter-spacing:.18em;font-size:10px;text-transform:uppercase}
.num{text-align:right;font-variant-numeric:tabular-nums}.dim{color:var(--dim)}.gs{color:var(--gd)}
.pill{border:1px solid;border-radius:999px;padding:1px 10px;font-size:10px;letter-spacing:.14em}
.pill.cool{color:var(--cy);border-color:#17384a}.pill.warn{color:var(--gd);border-color:#3a3016}
.pill.hot{color:var(--em);border-color:#4a2117;text-shadow:0 0 8px rgba(255,122,89,.7)}
td.desc{color:var(--dim);font-size:11px}td.desc b{color:var(--ink)}
body.calm .halo{transition:opacity 1.2s;opacity:.04!important}
@keyframes fest{30%{filter:saturate(1.9) hue-rotate(30deg) brightness(1.3)}}
body.fest main{animation:fest 2.6s ease}
@media (prefers-reduced-motion:reduce){*{animation:none!important}}
</style>`;

function render(d) {
  // --- neural-layer geometry ---
  // The colony as a neural net: slimes = input layer, orcs = hidden layer,
  // elf = output, dark elf burns beyond. Radius ∝ √(KB / 6KB law): an
  // over-diet mind GROWS INTO its neighbors — collisions aren't layout bugs,
  // they're the noise made visible (human doctrine 2026-09-14).
  const W = 1180, H = 640;
  const byName = {};
  for (const c of d.creatures) byName[c.name] = c;
  const resolve = n => byName[n] || byName[Object.keys(byName).find(k => n.endsWith(k) || k.endsWith(n)) || ''];
  const rad = c => Math.round(100 * 11 * Math.sqrt(Math.max(c.kb, 0.5) / 6)) / 100;
  const X = { slime: 215, orc: 560, elf: 905 };
  const nodes = [], edges = [];
  const spread = list => list.map((it, i) => ({ ...it, y: 74 + (H - 128) * (i + 1) / (list.length + 1) }));
  const slimeSeq = [];
  for (const o of d.orcs) for (const s of o.slimes) slimeSeq.push({ s, orc: o.orc });
  const slimeNodes = spread(slimeSeq).map(({ s, orc, y }) => {
    const c = resolve(s) || { kb: 6, stressPct: 0, race: 'slime', thoughts: 0, limit: 5, crosslinks: 0 };
    const r = rad(c);
    return { name: s, c, x: X.slime, y, r, orc, anch: 'end', lx: X.slime - r - 10, ly: y + 3 };
  });
  const orcNodes = spread(d.orcs.map(o => ({ o }))).map(({ o, y }) => {
    const c = resolve(o.orc) || { kb: 6, stressPct: 0, race: 'orc', thoughts: 0, limit: 5, crosslinks: 0 };
    const r = rad(c);
    return { name: o.orc, c, x: X.orc, y, r, anch: 'middle', lx: X.orc, ly: y - r - 10 };
  });
  const orcByName = {}; for (const n of orcNodes) orcByName[n.name] = n;
  for (const sn of slimeNodes) { const oc = orcByName[sn.orc];
    if (oc) edges.push(`<line class="e-${oc.name} e-${sn.name}" x1="${sn.x.toFixed(1)}" y1="${sn.y.toFixed(1)}" x2="${oc.x.toFixed(1)}" y2="${oc.y.toFixed(1)}"/>`); }
  nodes.push(...slimeNodes, ...orcNodes);
  // Elf(s): found generically by race, never by an assumed literal name — a world's Elf
  // is not always named "elf-colony" (that was one demo world's own name, not a schema).
  // Spread vertically like orcs/slimes if a world ever has more than one (Nature 6 — an
  // elf-colony forming from 2+ elves thinking alike is a real possibility, not the default).
  const elfNodes = spread(d.creatures.filter(c => c.race === 'elf').map(c => ({ c }))).map(({ c, y }) => {
    const r = rad(c);
    return { name: c.name, c, x: X.elf, y, r, anch: 'start', lx: X.elf + r + 10, ly: y + 3 };
  });
  nodes.push(...elfNodes);
  for (const en of elfNodes) for (const oc of orcNodes)
    edges.push(`<line class="elfedge e-${en.name} e-${oc.name}" x1="${oc.x.toFixed(1)}" y1="${oc.y.toFixed(1)}" x2="${en.x.toFixed(1)}" y2="${en.y.toFixed(1)}"/>`);
  // Dark elf(s): same generic-by-race fix, stacked if more than one (rare, but not assumed-single).
  const darkelfNodes = d.creatures.filter(c => c.race === 'darkelf').map((c, i) => {
    const r = rad(c), y = 66 + i * (r * 2 + 14);
    return { name: c.name, c, x: X.elf, y, r, anch: 'middle', lx: X.elf, ly: y - r - 10 };
  });
  nodes.push(...darkelfNodes);
  // Minds (skills) — not a layer, not raced: a shelf worn by the whole colony. Linked
  // to whichever creature nodes the harmony pass above actually found a textual
  // reason to connect (see harvest()'s "Mind ↔ creature harmony links") — never
  // hand-assigned, and drawn with zero edges when nothing in the world actually
  // names it yet, which is itself an honest reading, not a bug.
  const nodeByName = {}; for (const n of nodes) nodeByName[n.name] = n;
  const mindNodes = (d.minds || []).map((m, i, arr) => {
    const mc = { race: 'mind', kb: m.kb, stressPct: 0, dietPct: 0, thoughts: 0, limit: 1, crosslinks: m.links.length, genesisSignal: false, desc: m.desc, uses: m.uses };
    const r = Math.max(9, Math.min(18, rad(mc) * 0.55 + Math.min(m.uses, 10) * 0.4));
    const x = X.slime + (X.elf - X.slime) * (i + 1) / (arr.length + 1);
    const y = H - 30;
    return { name: m.name, c: mc, x, y, r, anch: 'middle', lx: x, ly: y + r + 14, links: m.links };
  });
  nodes.push(...mindNodes);
  for (const mn of mindNodes) for (const linkName of mn.links) {
    const t = nodeByName[linkName];
    if (t) edges.push(`<line class="mindedge e-${mn.name} e-${linkName}" x1="${mn.x.toFixed(1)}" y1="${mn.y.toFixed(1)}" x2="${t.x.toFixed(1)}" y2="${t.y.toFixed(1)}"/>`);
  }
  const layerTags = `<text class="ax" x="${X.slime}" y="36" text-anchor="middle">INPUT — SLIMES</text>` +
    `<text class="ax" x="${X.orc}" y="36" text-anchor="middle">HIDDEN — ORCS</text>` +
    `<text class="ax" x="${X.elf}" y="36" text-anchor="middle">OUTPUT — ELF ⋄ AWAKENED ABOVE</text>` +
    (mindNodes.length ? `<text class="ax" x="${W / 2}" y="${H - 8}" text-anchor="middle">⋄ MINDS — WORN, NOT RACED</text>` : '');
  const nodeSvg = layerTags + nodes.map(({ name, c, x, y, r, lx, ly, anch }) => {
    const col = RCOL[c.race] || RCOL.plain, au = aura(c);
    const tip = c.race === 'mind'
      ? `${esc(name)} — Mind · ${c.kb}KB · ${c.uses} use${c.uses === 1 ? '' : 's'} · worn by ${c.crosslinks} creature${c.crosslinks === 1 ? '' : 's'}${c.desc ? ' — ' + esc(c.desc) : ''}`
      : `${esc(name)} — ${esc(c.race)} · ${c.kb}KB · desk ${c.thoughts}/${c.limit} · links ${c.crosslinks ?? '–'}${c.genesisSignal ? ' · ⋄ genesis watch' : ''}`;
    return `<g id="n-${esc(name)}" data-name="${esc(name)}" class="node ${au}${c.race === 'mind' ? ' mind' : ''}${c.genesisSignal ? ' gs' : ''}">
      <circle class="halo" cx="${x.toFixed(1)}" cy="${y.toFixed(1)}" r="${(r * 1.7).toFixed(1)}" fill="${col}"/>
      <circle cx="${x.toFixed(1)}" cy="${y.toFixed(1)}" r="${r.toFixed(1)}" fill="var(--nodefill)" stroke="${col}" stroke-width="1.6" stroke-dasharray="${c.race === 'mind' ? '3 2' : 'none'}"/>
      <circle class="core" cx="${x.toFixed(1)}" cy="${y.toFixed(1)}" r="${(r * 0.45).toFixed(1)}" fill="${col}"/>
      <text text-anchor="${anch}" x="${lx.toFixed(1)}" y="${ly.toFixed(1)}">${esc(name.replace(/^(slime|orc|elf|darkelf|highorc|kijin)-/, ''))}</text>
      <title>${tip}</title></g>`;
  }).join('');

  // --- breath timeline: unified scale, real axes (human 2026-09-15 —
  // "xaxis should be in bottom of chart not there": the dates used to be
  // crammed into the top caption; now a true bottom axis with ticks/grid) ---
  const dates = Object.keys(d.days).sort();
  const breath = (() => {
    if (!dates.length) return '<div class="dim panel" style="padding:14px 16px">no days on record yet.</div>';
    const keys = [['journal', '#59d6ff'], ['thoughts', '#a78bfa'], ['commits', '#ff7a59']];
    const gMax = Math.max(1, ...dates.flatMap(x => keys.map(([k]) => d.days[x][k] || 0)));
    const H2 = 250, ml = 52, mr = 20, mt = 26, mb = 44, pw = W - ml - mr, ph = H2 - mt - mb;
    const sx = i => ml + (dates.length === 1 ? pw / 2 : i * pw / (dates.length - 1));
    const sy = v => mt + ph * (1 - v / gMax);
    let s = '';
    for (let t = 0; t <= 4; t++) { const v = Math.round(gMax * t / 4), yy = sy(v).toFixed(1);
      s += `<line x1="${ml}" y1="${yy}" x2="${W - mr}" y2="${yy}" stroke="#22304a" stroke-width="0.6" opacity="${t ? 0.45 : 1}"/>`
        + `<text x="${ml - 8}" y="${+yy + 3}" class="ax" text-anchor="end">${v}</text>`; }
    const step = Math.ceil(dates.length / 12);
    dates.forEach((x, i) => { if (i % step && i !== dates.length - 1) return; const xx = sx(i).toFixed(1);
      s += `<line x1="${xx}" y1="${H2 - mb}" x2="${xx}" y2="${H2 - mb + 5}" stroke="#22304a"/>`
        + `<text x="${xx}" y="${H2 - mb + 17}" class="ax" text-anchor="middle">${esc(x.slice(5))}</text>`; });
    for (const [k, col] of keys) {
      const pts = dates.map((x, i) => `${sx(i).toFixed(1)},${sy(d.days[x][k] || 0).toFixed(1)}`);
      s += `<polyline points="${pts.join(' ')}" fill="none" stroke="${col}" stroke-width="1.8" class="rib"/>`; }
    s += `<text x="${ml}" y="14" class="ax">events / day — unified scale (max ${gMax})</text>`
      + `<text x="${W - mr}" y="14" class="ax" text-anchor="end">${keys.map(([k, col]) => `<tspan fill="${col}">■ ${k}</tspan>`).join(' · ')}</text>`
      + `<text x="${W - mr}" y="${H2 - 4}" class="ax" text-anchor="end">day →</text>`;
    return `<svg viewBox="0 0 ${W} ${H2}" class="panel">${s}</svg>`;
  })();

  // Cast and manifest used to be two separate tables (same creatures, split columns) —
  // merged into one at the human's request, with an explicit numeric stress score
  // (not just the color pill) alongside the description.
  const brief = s => { let b = (s || '').split(/ [—-] /)[0].trim();
    if (b.length > 118) b = b.slice(0, 115).replace(/\s\S*$/, '') + '…'; return b; };
  const castRows = d.creatures.map(c => `<tr data-name="${esc(c.name)}" style="cursor:pointer">
      <td><span style="color:${RCOL[c.race] || RCOL.plain}">●</span> <b>${esc(c.name)}</b>${c.genesisSignal ? ' <span class="gs">⋄</span>' : ''}</td>
      <td class="dim">${esc(c.race)}</td>
      <td class="num">${c.kb}</td><td class="num">${c.thoughts}/${c.limit}</td><td class="num">${c.crosslinks}</td>
      <td><span class="pill ${aura(c)}">${c.stressPct}%</span></td>
      <td class="desc" title="${esc(c.desc || '')}">${c.desc ? esc(brief(c.desc)) : '–'}</td></tr>`).join('');
  const modelRows = Object.entries(d.models || {})
    .sort((a, b) => b[1].mentions - a[1].mentions)
    .map(([id, v]) => `<tr><td><b>${esc(id)}</b></td><td class="dim">${esc(v.mounted.join(' · ') || '–')}</td><td class="num">${v.mentions}</td></tr>`).join('');
  const mindRows = (d.minds || []).slice().sort((a, b) => b.uses - a.uses)
    .map(m => `<tr><td><b>${esc(m.name)}</b></td><td class="num">${m.uses}</td><td class="num">${m.kb}</td>
      <td class="dim">${m.links.length ? esc(m.links.join(' · ')) : '–'}</td>
      <td class="desc" title="${esc(m.desc || '')}">${m.desc ? esc(brief(m.desc)) : '–'}</td></tr>`).join('');

  const fmtN = v => v >= 1e6 ? (v / 1e6).toFixed(1) + 'M' : v >= 1e3 ? (v / 1e3).toFixed(1) + 'k' : String(v);
  const bRows = (d.bodies || []).map(b => `<tr><td><b>${esc(b.name)}</b></td><td class="dim">${esc(b.mode)}</td><td>${esc(b.model)}</td><td class="dim">${b.born}</td><td class="num">${b.kb}</td></tr>`).join('');
  const uRows = Object.entries(d.agentUse || {}).sort((a, b) => (b[1].totIn + b[1].totOut) - (a[1].totIn + a[1].totOut))
    .map(([a, u]) => `<tr><td><b>${esc(a)}</b></td><td class="dim">${esc(u.models.join(' · ') || '–')}</td><td class="num">${u.sessions}</td><td class="num">${fmtN(u.avgCtx)}</td><td class="num">${fmtN(u.totIn)} / ${fmtN(u.totOut)}</td><td class="dim">${u.lastDay || '–'}</td></tr>`).join('');

  return `<!doctype html><meta charset="utf-8"><title>tempest ⋄ ${esc(d.world)} — ${esc(path.basename(d.root))}</title>
${PAGE_STYLE}
<main>
<a href="/" class="allworlds">← all worlds</a>
<h1><span class="sigil">⋄</span> TEMPEST <span style="letter-spacing:.1em;color:var(--dim);font-size:11px"> ${d.world.toUpperCase()} — THE WORLD, OBSERVED · /${d.world}/ · :${PORT}</span></h1>
<div class="meta"><span class="tbtns"><button id="themeBtn" title="light/dark">◐ light</button><button id="stressBtn" title="the suffering blink, while you watch">⚡ stress detect</button>
<button class="rng" data-r="1" title="last 24h — today (day-bucket)">24h</button><button class="rng" data-r="7" title="last 7 days">7d</button><button class="rng active" data-r="30" title="last 30 days">30d</button><button class="rng" data-r="0" title="all time">all</button></span><span class="chip">${esc(path.basename(d.root))}</span><span class="chip">canon v${d.canonV}</span>
<span class="chip">${d.chartV === null ? 'chart: none' : d.chartDebt ? `<span class="debt">chart v${d.chartV} × debt 22/22a</span>` : `<span class="ok">chart v${d.chartV} ✓</span>`}</span>
<span class="chip">${esc(d.when.replace('T', ' ').slice(0, 19))}</span>
<span class="chip">genesis watch: ${d.genesisWatch.length ? esc(d.genesisWatch.join(' · ')) : '<span class="ok">none</span>'}</span></div>

<h2>⋄ cast — who holds what</h2>
<table><tr><th>creature</th><th>race</th><th>KB</th><th>desk</th><th>links</th><th>stress</th><th>expertise (brief — full text on hover)</th></tr>${castRows}</table>

<h2>⋄ minds — worn, not raced</h2>
<table><tr><th>mind</th><th>uses</th><th>KB</th><th>worn by</th><th>purpose (brief — full text on hover)</th></tr>${mindRows || '<tr><td colspan="5" class="dim">no Minds in this world yet — /don brings one in from .opencode/skills/</td></tr>'}</table>
<div class="dim">uses = Skill tool_use invocations counted from this world's own Claude Code transcripts (~/.claude/projects/) — a Mind that exists but reads 0 has never actually been invoked here, only referenced.</div>

<h2>⋄ models — mounted &amp; mentioned</h2>
<table><tr><th>model</th><th>mounted on (colony genomes)</th><th>ledger sightings</th></tr>${modelRows || '<tr><td class="dim">no model pins found</td></tr>'}</table>
<div class="dim">mounts = <b>model:</b> pins in .opencode/agents/*.md; sightings = name-drops across the colony ledger + maps (traces, not wires — real call frequency lives in the gateway's telemetry).</div>

<h2>⋄ agents — minted bodies &amp; session breath</h2>
${(() => {
    return `<table><tr><th>minted body</th><th>mode</th><th>mount</th><th>born</th><th>KB</th></tr>${bRows || '<tr><td colspan="5" class="dim">minds-only world — no bodies minted (embodiment E2)</td></tr>'}</table>
<div class="dim" style="margin:10px 0 4px">per-session breath — avg context = mean tokens (in+out) per session answering as that agent; the living ledger: <b>${d.home}/metrics/agents-usage.md</b> (rewritten when reality moves)</div>
<table><tr><th>agent</th><th>mounts seen</th><th>sessions</th><th>avg ctx</th><th>in / out</th><th>last day</th></tr>${uRows || '<tr><td colspan="6" class="dim">no sessions recorded for this world yet — the table fills itself as sessions work</td></tr>'}</table>`;
  })()}

<h2>⋄ system mood</h2><div class="mood">${esc(d.health)}</div>

<h2>⋄ remedies</h2>
<div class="panel" style="padding:14px 16px">
  <span class="tbtns" style="float:none"><button id="holiBtn" title="write the dated relief worklist from live metrics AND launch sequential creature-hat relief runs">🎒 holidays</button><button id="partyBtn" title="celebrate + name the genesis-watch births (naming #1)">🎉 party</button></span>
  <span id="actOut" class="dim"></span>
  <div class="dim" style="margin-top:8px">the board prescribes, records, dispatches (law 2026-09-15): <b>🎒 holidays</b> writes the relief worklist into the day's scratch <b>and launches sequential relief</b> — one creature-hat <span style="font-family:inherit">opencode run</span> per stressed mind (distil the desk · diet split · ⋄ review the SPLIT — every creature snapshotted into the day's scratch <b>before</b> its run, frontmatter byte-restored if drifted <b>after</b>), progress streamed above, steps logged to <span style="font-family:inherit">${d.home}/metrics/relief.jsonl</span>; the tool itself never writes a mind.
  <b>🎉 party</b> logs a celebration to <span style="font-family:inherit">${d.home}/metrics/celebrations.jsonl</span> — and any ⋄ names it carries become naming #1 of the genesis law (a birth still needs its need named twice, by a mind that means it).</div>
</div>

<h2>⋄ tokens — session breath</h2>${(() => { const U = d.live.rows ? d.live : d.tokens;
    if (!U.rows) return `<div class="dim">no usage yet — the board reads opencode's session store (<span style="font-family:inherit">~/.local/share/opencode/opencode.db</span>) live, and the manual law stands beside it: a session may append one JSONL line to <b>${d.home}/metrics/tokens.jsonl</b> (<span style="font-family:inherit">{"ts","agent","model","in","out"}</span>). USD truth lives in the gateway's telemetry, not here.</div>`;
    const withSrc = d.live.rows ? 'opencode session store (read-only, colony-scoped by directory)' : 'manual ledger';
    return `<div class="cards">
  <div class="card"><b>${(U.totalIn / 1e6).toFixed(1)}M</b> tokens in</div>
  <div class="card"><b>${(U.totalOut / 1e6).toFixed(1)}M</b> tokens out</div>
  <div class="card"><b>${U.rows}</b> ${d.live.rows ? 'sessions counted' : 'ledger rows'}</div></div>
<div class="dim" style="margin:10px 0 4px">per model — who the breath cost</div>
<table id="tokModelTable"><tr><th>model</th><th>sessions (agents)</th><th>runs</th><th>in</th><th>out</th><th>share</th></tr>
${Object.entries(U.perModel).sort((a, b) => (b[1].in + b[1].out) - (a[1].in + a[1].out)).map(([mid, v]) => { const all = (U.totalIn + U.totalOut) || 1; return `<tr><td>${esc(mid)}</td><td class="dim">${esc(v.agents.join(', ') || '–')}</td><td class="num">${v.rows}</td><td class="num">${v.in}</td><td class="num">${v.out}</td><td class="num">${(100 * (v.in + v.out) / all).toFixed(1)}%</td></tr>`; }).join('')}</table>
<div class="dim" style="margin:10px 0 4px">per day</div>
<table id="tokDayTable"><tr><th>day</th><th>in</th><th>out</th></tr>
${Object.entries(U.perDay).sort().map(([k, v]) => `<tr><td>${esc(k)}</td><td class="num">${v.in}</td><td class="num">${v.out}</td></tr>`).join('')}</table>
<div class="dim" style="margin:14px 0 4px">consumption — pick a model · solid in · dashed out · range obeys the toolbar (24h = today, day-bucket)</div>
<div id="tokSel" style="margin-bottom:6px"></div>
<div id="tokChart"></div>
<script type="application/json" id="tokData">${esc(JSON.stringify(d.live.series || {}))}</script>
<script type="application/json" id="tokAgents">${esc(JSON.stringify(Object.fromEntries(Object.entries(d.live.perModel).map(([k, v]) => [k, v.agents]))))}</script>
<div class="dim" style="margin-top:6px">source: ${withSrc}${d.live.rows && d.tokens.rows ? ` + manual ledger (${d.tokens.rows} rows beside it)` : ''} — USD still lives in gateway telemetry.</div>`; })()}

<h2>⋄ the colony — size is weight, glow is stress</h2>
<svg viewBox="0 0 ${W} ${H}" class="panel">${edges.join('')}${nodeSvg}</svg>
<div id="focusPanel" class="panel" style="margin-top:10px"></div>
<script type="application/json" id="zdata">${esc(JSON.stringify({ creatures: Object.fromEntries(d.creatures.map(c => [c.name, { race: c.race, kb: c.kb, dietPct: c.dietPct, thoughts: c.thoughts, limit: c.limit, stressPct: c.stressPct, links: c.crosslinks, g: !!c.genesisSignal, last: c.lastThought || null, desc: c.desc || '' }])), orcOf: Object.fromEntries(d.orcs.flatMap(o => o.slimes.map(s => { const c = (byName[s] || byName[Object.keys(byName).find(k => s.endsWith(k) || k.endsWith(s))] || ''); return c ? [c.name, o.orc] : null; }).filter(Boolean))) }))}</script>
<script>
(function(){
var Z=JSON.parse(document.getElementById('zdata').textContent);
var P=document.getElementById('focusPanel');
function esc2(s){return String(s==null?'':s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');}
function clearAll(){document.body.classList.remove('focused');
  document.querySelectorAll('.node.focus,.lit').forEach(function(e){e.classList.remove('focus','lit');});
  P.innerHTML='<span class="dim">click a creature — node or cast row — and it steps forward; click again (or the void) to release. Nothing moves on its own: calm is ambient, attention is yours.</span>';}
function focus(name){var c=Z.creatures[name];if(!c)return;clearAll();
  var g=document.getElementById('n-'+name);if(g){document.body.classList.add('focused');g.classList.add('focus');}
  document.querySelectorAll('.e-'+CSS.escape(name)).forEach(function(e){e.classList.add('lit');});
  P.innerHTML='<b>'+esc2(name)+'</b> <span class="dim">'+esc2(c.race)+(Z.orcOf[name]?' under '+esc2(Z.orcOf[name]):'')+'</span><br>'
   +(c.desc?'<span class="focusdesc">'+esc2(c.desc)+'</span><br>':'')
   +'doc '+c.kb+'KB <span class="dim">(diet '+c.dietPct+'% of the 6KB law)</span> · desk '+c.thoughts+'/'+c.limit
   +' <span class="dim">(stress '+c.stressPct+'%)</span> · crosslinks '+c.links
   +(c.g?' · <span class="gs">⋄ genesis watch</span>':'')
   +'<br><span class="dim">last thought written: '+esc2(c.last||'none yet')+'</span>';}
document.querySelectorAll('g.node,tr[data-name]').forEach(function(el){
  el.addEventListener('click',function(ev){ev.stopPropagation();var n=el.getAttribute('data-name');
   var g=document.getElementById('n-'+n);
   if(g&&g.classList.contains('focus'))clearAll();else focus(n);});});
document.body.addEventListener('click',clearAll);
var tb=document.getElementById('themeBtn');
if(tb)tb.addEventListener('click',function(ev){ev.stopPropagation();
 var l=document.body.classList.toggle('light');tb.textContent=l?'◑ dark':'◐ light';});
var sb=document.getElementById('stressBtn');
if(sb)sb.addEventListener('click',function(ev){ev.stopPropagation();
 var s=document.body.classList.toggle('stressmode');sb.classList.toggle('active',s);
 sb.textContent=s?'⚡ stress live':'⚡ stress detect';});
function act(url){var o=document.getElementById('actOut');o.textContent='…';
 fetch(url,{method:'POST'}).then(function(r){return r.json();}).then(function(j){
  if(url==='holidays'){
   if(!j.stressed){o.textContent='all minds within budget — nothing to relieve.';return;}
   o.textContent='worklist → '+j.file+' ('+j.stressed+' stressed)';
   if(j.relief&&j.relief.refused)o.textContent+=' — run already in flight: ';
   if(j.relief&&(j.relief.launched||j.relief.refused))pollRelief(o);
  }else{
   o.textContent='logged — births named: '+(j.births&&j.births.length?j.births.join(', '):'none tonight');
  }
 }).catch(function(){o.textContent='failed — is the board alive?';});}
function pollRelief(o){fetch('relief').then(function(r){return r.json();}).then(function(j){
 if(!j.total&&!j.active)return;
 if(j.active){
  o.textContent='relieving '+j.current+' ('+(j.done.length+j.failed.length+1)+'/'+j.total+') · done '+j.done.length+' · failed '+j.failed.length+' · next: '+(j.remaining.slice(0,3).join(', ')||'–');
  setTimeout(function(){pollRelief(o);},5000);
 }else{
  o.textContent='relief finished — relieved '+j.done.length+' ('+j.done.join(', ')+')'+(j.failed.length?', failed '+j.failed.length+' ('+j.failed.join(', ')+')':'')+' — re-reading minds…';
  setTimeout(function(){location.reload();},3000);}}).catch(function(){});}
var hb=document.getElementById('holiBtn');
if(hb)hb.addEventListener('click',function(ev){ev.stopPropagation();act('holidays');
 document.body.classList.add('calm');setTimeout(function(){document.body.classList.remove('calm');},3000);});
var pb=document.getElementById('partyBtn');
if(pb)pb.addEventListener('click',function(ev){ev.stopPropagation();act('party');
 document.body.classList.add('fest');setTimeout(function(){document.body.classList.remove('fest');},2600);});
clearAll();
// an open page holds the board awake (lifecycle law — silence is what sleeps it)
setInterval(function(){fetch('pulse').catch(function(){});},60000);
})();
</script>
<div class="meta">layers <span style="color:var(--cy)">slimes →</span> <span style="color:var(--vi)">orcs →</span> <span style="color:var(--gd)">elf</span>, <span style="color:var(--em)">darkelf</span> burns apart. Size = doc weight
(radius ∝ √(KB/6KB) — an over-fed mind <i>leans on its neighbors</i>: that crowding is the disharmony, drawn true). Glow + ember core = stress. Click a node to focus; <b>⚡ stress detect</b> arms the clignotement. ⋄ = genesis watch
(stress ≥80% ∧ crosslinks ≥1.5× median — heuristic; a birth still needs its need named twice).</div>

<h2>⋄ evolution — the breath of days</h2><div id="breathBox">${breath}</div><script type="application/json" id="breathData">${esc(JSON.stringify(d.days))}</script>

<h2>⋄ provenance</h2><div class="meta">harvested live at request time from .opencode/skills/* · ${d.canonFile} stamps · AGENTS.md routing table · the ${d.home} journal · git log — nothing stored. stdlib node, zero scripts, zero CDN; the tooling law (nature law 8). <span class="dim">port ${PORT}</span></div>
<script>
// consolidated client runtime: range filter drives BOTH charts + token tables
// (human order 2026-09-15). No backticks or interpolation markers allowed in
// here — this text rides inside the server template literal.
// Ranges are day-bucketed: 1d = today.
(function(){
  var RANGE=0;
  var $=function(id){return document.getElementById(id);};
  function fmt(n){if(n>=1e9)return (n/1e9).toFixed(1)+'B';if(n>=1e6)return (n/1e6).toFixed(1)+'M';if(n>=1e3)return (n/1e3).toFixed(1)+'k';return String(Math.round(n));}
  function cutoff(){if(!RANGE)return null;var t=new Date(Date.now()-(RANGE-1)*864e5);var p=function(x){return String(x).length<2?'0'+x:''+x;};return t.getFullYear()+'-'+p(t.getMonth()+1)+'-'+p(t.getDate());}
  function inR(day){var c=cutoff();return !c||day>=c;}
  function hj(s){return String(s).replace(/</g,'&lt;');}
  // ---- evolution (server paints the all-range first frame; we redraw) ----
  var BD={};var btag=$('breathData');if(btag){try{BD=JSON.parse(btag.textContent);}catch(e){}}
  function drawBreath(){var box=$('breathBox');if(!box)return;
   var dates=Object.keys(BD).filter(inR).sort();
   if(!dates.length){box.innerHTML='<div class="dim panel" style="padding:14px 16px">no days in this range.</div>';return;}
   var keys=[['journal','#59d6ff'],['thoughts','#a78bfa'],['commits','#ff7a59']];
   var gM=1;dates.forEach(function(x){keys.forEach(function(kv){var v=BD[x][kv[0]]||0;if(v>gM)gM=v;});});
   var W=1180,H=250,ml=52,mr=20,mt=26,mb=44,pw=W-ml-mr,ph=H-mt-mb;
   var sx=function(i){return ml+(dates.length===1?pw/2:i*pw/(dates.length-1));};
   var sy=function(v){return mt+ph*(1-v/gM);};
   var s='',t,i,yy,xx;
   for(t=0;t<=4;t++){var v=Math.round(gM*t/4);yy=sy(v).toFixed(1);
    s+='<line x1="'+ml+'" y1="'+yy+'" x2="'+(W-mr)+'" y2="'+yy+'" stroke="#22304a" stroke-width="0.6" opacity="'+(t?0.45:1)+'"/>';
    s+='<text x="'+(ml-8)+'" y="'+(+yy+3)+'" class="ax" text-anchor="end">'+v+'</text>';}
   var step=Math.ceil(dates.length/12);
   for(i=0;i<dates.length;i++){if(i%step&&i!==dates.length-1)continue;xx=sx(i).toFixed(1);
    s+='<line x1="'+xx+'" y1="'+(H-mb)+'" x2="'+xx+'" y2="'+(H-mb+5)+'" stroke="#22304a"/>';
    s+='<text x="'+xx+'" y="'+(H-mb+17)+'" class="ax" text-anchor="middle">'+dates[i].slice(5)+'</text>';}
   keys.forEach(function(kv){var pts=[];for(i=0;i<dates.length;i++)pts.push(sx(i).toFixed(1)+','+sy(BD[dates[i]][kv[0]]||0).toFixed(1));
    s+='<polyline points="'+pts.join(' ')+'" fill="none" stroke="'+kv[1]+'" stroke-width="1.8"/>';});
   s+='<text x="'+ml+'" y="14" class="ax">events / day — unified scale (max '+gM+')</text>';
   var lg='';keys.forEach(function(kv){lg+=(lg?' · ':'')+'<tspan fill="'+kv[1]+'">■ '+kv[0]+'</tspan>';});
   s+='<text x="'+(W-mr)+'" y="14" class="ax" text-anchor="end">'+lg+'</text>';
   s+='<text x="'+(W-mr)+'" y="'+(H-4)+'" class="ax" text-anchor="end">day →</text>';
   box.innerHTML='<svg viewBox="0 0 '+W+' '+H+'" class="panel">'+s+'</svg>';}
  // ---- tokens ----
  var TD={},TAG={};var ttag=$('tokData'),atag=$('tokAgents');
  if(ttag){try{TD=JSON.parse(ttag.textContent);}catch(e){}}
  if(atag){try{TAG=JSON.parse(atag.textContent);}catch(e){}}
  var mids=Object.keys(TD);var cur=null;
  function totR(m){var t=0,dm=TD[m];for(var k in dm)if(inR(k))t+=dm[k][0]+dm[k][1];return t;}
  function drawTok(){var box=$('tokChart');if(!box||!cur)return;
   var days=Object.keys(TD[cur]||{}).filter(inR).sort(),n=days.length,i,t;
   if(!n){box.innerHTML='<div class="dim" style="padding:10px 0">'+hj(cur)+' — nothing in this range.</div>';return;}
   var W=1180,H=300,ml=64,mr=18,mt=22,mb=42,pw=W-ml-mr,ph=H-mt-mb,mx=1;
   for(i=0;i<n;i++){var v=TD[cur][days[i]];if(v[0]>mx)mx=v[0];if(v[1]>mx)mx=v[1];}
   var sx=function(q){return ml+(n===1?pw/2:q*pw/(n-1));};
   var sy=function(q){return mt+ph*(1-q/mx);};
   var s='<text x="'+ml+'" y="14" class="ax">tokens / day — '+hj(cur)+'</text>';
   s+='<text x="'+(W-mr)+'" y="14" class="ax" text-anchor="end"><tspan fill="#59d6ff">■ in</tspan> · <tspan fill="#a78bfa">■ out</tspan></text>';
   for(t=0;t<=4;t++){var tv=Math.round(mx*t/4);yy=sy(tv).toFixed(1);
    s+='<line x1="'+ml+'" y1="'+yy+'" x2="'+(W-mr)+'" y2="'+yy+'" stroke="#22304a" stroke-width="0.6" opacity="'+(t?0.45:1)+'"/>';
    s+='<text x="'+(ml-8)+'" y="'+(+yy+3)+'" class="ax" text-anchor="end">'+fmt(tv)+'</text>';}
   var step=Math.ceil(n/12);
   for(i=0;i<n;i++){if(i%step&&i!==n-1)continue;xx=sx(i).toFixed(1);
    s+='<line x1="'+xx+'" y1="'+(H-mb)+'" x2="'+xx+'" y2="'+(H-mb+5)+'" stroke="#22304a"/>';
    s+='<text x="'+xx+'" y="'+(H-mb+17)+'" class="ax" text-anchor="middle">'+days[i].slice(5)+'</text>';}
   s+='<text x="'+(W-mr)+'" y="'+(H-4)+'" class="ax" text-anchor="end">day →</text>';
   var pin=[],pout=[];
   for(i=0;i<n;i++){var vv=TD[cur][days[i]];pin.push(sx(i).toFixed(1)+','+sy(vv[0]).toFixed(1));pout.push(sx(i).toFixed(1)+','+sy(vv[1]).toFixed(1));}
   s+='<polyline points="'+pin.join(' ')+'" fill="none" stroke="#59d6ff" stroke-width="1.8"/>';
   s+='<polyline points="'+pout.join(' ')+'" fill="none" stroke="#a78bfa" stroke-width="1.8" stroke-dasharray="4 3"/>';
   box.innerHTML='<svg viewBox="0 0 '+W+' '+H+'" class="panel">'+s+'</svg>';}
  function drawTokTables(){var mt=$('tokModelTable'),dt=$('tokDayTable');var tot=1;mids.forEach(function(m){tot+=totR(m);});tot=tot-1||1;
   if(mt){var rows='<tr><th>model</th><th>sessions (agents)</th><th>runs</th><th>in</th><th>out</th><th>share</th></tr>';
    mids.slice().sort(function(a,b){return totR(b)-totR(a);}).forEach(function(m){
     var ti=0,to=0,rn=0,dm=TD[m];for(var k in dm)if(inR(k)){ti+=dm[k][0];to+=dm[k][1];rn+=dm[k][2]||0;}
     rows+='<tr><td>'+hj(m)+'</td><td class="dim">'+hj((TAG[m]||[]).join(', ')||'–')+'</td><td class="num">'+rn+'</td><td class="num">'+ti+'</td><td class="num">'+to+'</td><td class="num">'+((100*(ti+to))/tot).toFixed(1)+'%</td></tr>';});
    mt.innerHTML=rows;}
   if(dt){var all={};mids.forEach(function(m){var dm=TD[m];for(var k in dm){if(inR(k)){var e=all[k]||[0,0];e[0]+=dm[k][0];e[1]+=dm[k][1];all[k]=e;}}});
    var rr='<tr><th>day</th><th>in</th><th>out</th></tr>';Object.keys(all).sort().forEach(function(kd){
     rr+='<tr><td>'+kd+'</td><td class="num">'+all[kd][0]+'</td><td class="num">'+all[kd][1]+'</td></tr>';});
    dt.innerHTML=rr;}}
  function initTok(){var sel=$('tokSel');if(!sel)return;
   if(!mids.length){sel.innerHTML='<span class="dim">no consumption data</span>';return;}
   mids.sort(function(a,b){return totR(b)-totR(a);});cur=mids[0];sel.innerHTML='';
   mids.forEach(function(m){var b=document.createElement('button');b.textContent=m;b.style.margin='0 6px 6px 0';
    b.onclick=function(){cur=m;var ch=sel.children;for(var k=0;k<ch.length;k++)ch[k].style.borderColor='';b.style.borderColor='var(--cy)';drawTok();};
    if(m===cur)b.style.borderColor='var(--cy)';
    sel.appendChild(b);});
   drawTok();}
  var btns=document.querySelectorAll('.rng');
  function apply(r){RANGE=r;for(var k=0;k<btns.length;k++){btns[k].classList.toggle('active',+btns[k].getAttribute('data-r')===r);}drawBreath();drawTokTables();drawTok();}
  for(var bi=0;bi<btns.length;bi++){(function(b){b.onclick=function(){apply(+b.getAttribute('data-r'));};})(btns[bi]);}
  drawBreath();initTok();drawTokTables();
})();
</script>
</main>`;
}

// ------------ global index — every world this one app knows about ------------
// Human order 2026-09-20: "1 app in whole machine not multiple ... global dashboard
// give overall overview of all worlds". Harvests every registered world fresh on
// each request (same "nothing stored, harvested live" law as a single world's page)
// and renders a card per world, linking to /<name>/.
function renderIndex(worlds) {
  const fmtN = v => v >= 1e6 ? (v / 1e6).toFixed(1) + 'M' : v >= 1e3 ? (v / 1e3).toFixed(1) + 'k' : String(v);
  const worldData = worlds.map(w => {
    let d, err = null;
    try { d = harvest(w.path); } catch (e) { err = e.message; }
    return { w, d, err };
  });

  // ---- machine-wide activity: every session on this machine, not filtered to a
  // registered world (human order 2026-09-20: "how much consumption in total if
  // opencode session etc ... information if session is active or not") ----
  const claudeG = harvestClaudeGlobal();
  const openG = harvestOpencodeGlobal();
  const claudeModelRows = Object.entries(claudeG.perModel).sort((a, b) => (b[1].in + b[1].out) - (a[1].in + a[1].out))
    .map(([m, v]) => `<tr><td><b>${esc(m)}</b></td><td class="num">${fmtN(v.in)}</td><td class="num">${fmtN(v.out)}</td></tr>`).join('');
  const openModelRows = Object.entries(openG.perModel).sort((a, b) => (b[1].in + b[1].out) - (a[1].in + a[1].out))
    .map(([m, v]) => `<tr><td><b>${esc(m)}</b></td><td class="num">${fmtN(v.in)}</td><td class="num">${fmtN(v.out)}</td></tr>`).join('');

  const claudePanel = `<div class="panel actpanel">
    <h3>⋄ Claude Code <span class="pill ${claudeG.active ? 'hot' : 'cool'}">${claudeG.active ? claudeG.active + ' active now' : 'idle'}</span></h3>
    <div class="wstat">${claudeG.sessions} session${claudeG.sessions === 1 ? '' : 's'} on this machine (any directory, not just registered worlds)</div>
    <div class="wstat">${fmtN(claudeG.totalIn)} in / ${fmtN(claudeG.totalOut)} out — all-time, every transcript under <span style="font-family:inherit">~/.claude/projects/</span></div>
    ${claudeModelRows ? `<table><tr><th>model</th><th>in</th><th>out</th></tr>${claudeModelRows}</table>` : '<div class="dim">no usage recorded yet</div>'}
  </div>`;
  const openPanel = !openG.dbExists ? `<div class="panel actpanel">
    <h3>⋄ OpenCode</h3><div class="dim">no <span style="font-family:inherit">~/.local/share/opencode/opencode.db</span> on this machine — OpenCode not in use here, or never run.</div>
  </div>` : !openG.available ? `<div class="panel actpanel">
    <h3>⋄ OpenCode <span class="pill hot">unreadable</span></h3>
    <div class="wstat">The database exists — real sessions may be sitting in it — but this board's own Node runtime can't read it: <span style="font-family:inherit">${esc(openG.error || 'unknown error')}</span>.</div>
    <div class="wstat dim"><span style="font-family:inherit">node:sqlite</span> needs Node 22.5+; this process is running on ${esc(process.version)}. Nature 9: a silent instrument is itself a finding — this panel says so instead of quietly reporting zero.</div>
  </div>` : `<div class="panel actpanel">
    <h3>⋄ OpenCode <span class="pill ${openG.activeRows ? 'hot' : 'cool'}">${openG.activeRows ? openG.activeRows + ' recent run' + (openG.activeRows === 1 ? '' : 's') : 'idle'}</span></h3>
    <div class="wstat">${openG.rows} run${openG.rows === 1 ? '' : 's'} recorded, all-time, machine-wide</div>
    <div class="wstat">${fmtN(openG.totalIn)} in / ${fmtN(openG.totalOut)} out</div>
    ${openModelRows ? `<table><tr><th>model</th><th>in</th><th>out</th></tr>${openModelRows}</table>` : '<div class="dim">no usage recorded yet</div>'}
    <div class="wstat dim">"recent" = a row touched in the last 5 minutes — opencode's schema carries no session-id here, so this is a run count, not a precise open-session count.</div>
  </div>`;

  const cards = worldData.length ? worldData.map(({ w, d, err }) => {
    const censusLine = d ? Object.entries(d.census).map(([race, n]) => `<span>${esc(race)} ${n}</span>`).join('') : '';
    const health = d ? esc(d.health) : `unreadable: ${esc(err || 'unknown error')}`;
    const stressedN = d ? d.creatures.filter(c => c.stressPct >= 60).length : 0;
    return `<a class="wcard" href="/${esc(w.name)}/">
      <h3>⋄ ${esc(w.name)}${stressedN ? ` <span class="pill warn">${stressedN} stressed</span>` : ''}</h3>
      <div class="wstat wcensus">${censusLine || '<span class="dim">no population yet</span>'}</div>
      <div class="wstat">${health}</div>
      <div class="wstat">last seen ${esc((w.lastSeen || '').slice(0, 10) || '–')} · ${esc(w.path)}</div>
    </a>`;
  }).join('') : `<div class="dim panel" style="padding:14px 16px;grid-column:1/-1">
      no worlds registered yet. Run <b>/isekai</b> or <b>node tempest.js &lt;dir&gt; --ensure</b> in a world to add it here.</div>`;

  const totalCreatures = worldData.reduce((n, { d }) => n + (d ? d.creatures.length : 0), 0);

  return `<!doctype html><meta charset="utf-8"><title>tempest ⋄ all worlds</title>
${PAGE_STYLE}
<style>.actpanel{padding:16px 18px}.actpanel h3{margin:0 0 8px;font-size:13px;letter-spacing:.06em}
.actgrid{display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:14px;margin-top:14px}</style>
<main>
<h1><span class="sigil">⋄</span> TEMPEST <span style="letter-spacing:.1em;color:var(--dim);font-size:11px"> ALL WORLDS, OBSERVED · :${PORT}</span></h1>
<div class="meta">${worlds.length} world${worlds.length === 1 ? '' : 's'} registered · ${totalCreatures} creature${totalCreatures === 1 ? '' : 's'} total ·
one app, one process, one port — each world lives at its own <span style="font-family:inherit">/&lt;name&gt;/</span> sub-path.</div>

<h2>⋄ machine — total breath across every session, every world</h2>
<div class="actgrid">${claudePanel}${openPanel}</div>

<h2>⋄ worlds</h2>
<div class="wcards">${cards}</div>
</main>`;
}

// ------------ relief dispatch (law amended 2026-09-15, human order "do on dashboard") ------------
// The board prescribes, records, and DISPATCHES: it never writes a mind itself.
// Each relief step spawns a sequential creature-hat session (`opencode run`, cwd =
// colony root → AGENTS.md + colony law load with it) which performs the distillation.
// Steps log to the world home's metrics/relief.jsonl; one run in flight at a time
// PER WORLD — reliefByWorld is keyed by world name since one app now serves every
// world at once, and a relief run in world A must never touch world B's queue.
//
// Departure from source: the source hardcodes `opencode run`. On a Claude-only
// machine that would silently strand relief runs, so this ports the fallback this
// session already built and tested: prefer opencode, fall back to `claude -p`.
const reliefByWorld = new Map(); // name -> {active, queue, done, failed, current, total, startedAt, finishedAt}

const reliefLog = (root, home, rec) => {
  const f = path.join(root, home, 'metrics', 'relief.jsonl');
  fs.mkdirSync(path.dirname(f), { recursive: true });
  fs.appendFileSync(f, JSON.stringify(rec) + '\n');
};
// desk-overflow first (worst ratio first), then diet-only by weight descending.
const stressedOf = d => d.creatures.filter(c => c.thoughts > c.limit || c.dietPct > 100)
  .sort((a, b) => {
    const ao = a.thoughts > a.limit, bo = b.thoughts > b.limit;
    if (ao !== bo) return bo - ao;
    if (ao && bo) return (b.thoughts / b.limit) - (a.thoughts / a.limit) || b.kb - a.kb;
    return b.kb - a.kb;
  });
function reliefBrief(c, home) {
  return [
    `Relief duty, tempest dispatch (the instruments law: the board dispatches, creature-hat sessions rewrite minds).`,
    `You ARE the creature "${c.name}" of this colony for this session only. Territory: ONLY .opencode/skills/${c.name}/ (its SKILL.md and any reference/ beside it). Touch no other creature's doc, no other file.`,
    `State: desk ${c.thoughts}/${c.limit} dated Thoughts entries; doc ${c.kb}KB against the ${DIET_KB}KB diet law (the breath law).`,
    `Act, in order:`,
    `1. Read .opencode/skills/${c.name}/SKILL.md in full, its ## Thoughts desk included.`,
    `2. Distil the desk: fold durable rules/pitfalls into the doc's own sections; bulky detail goes to reference/ files beside the doc. A lesson shared with sibling creatures is left as a one-line pointer to its orc instead.`,
    `3. Rewrite the ## Thoughts desk to at most ${c.limit} dated (YYYY-MM-DD), first-person entries. Lessons now living in the body are cleared, never restated there.`,
    c.kb > DIET_KB ? `4. Diet breach: move reference-grade detail into reference/ until SKILL.md sits near ${DIET_KB}KB. The frontmatter description block stays byte-identical (it is the load trigger).` : '',
    c.genesisSignal ? `5. Genesis-watch ⋄: your stress+crosslinks hold the signal and naming #1 is already logged for you (metrics/celebrations.jsonl) — review the SPLIT and record the verdict (need stands named a second time / rejected, with one line of why) as a dated entry in the rewritten ## Thoughts — naming #1 rides ${home}/metrics/celebrations.jsonl.` : '',
    `Absolute: no git verbs at all (no commit, no push), no network, no files outside your territory. Reply with exactly one line: creature, thoughts kept, final KB.`,
  ].filter(Boolean).join('\n');
}
function reliefCheck(root, home, c, snap) {
  // post-step harm check (human 2026-09-15: "it doesn't cause harm, right?")
  const rec = { ts: new Date().toISOString(), kind: 'relief-check', creature: c.name };
  try {
    const docP = path.join(root, '.opencode', 'skills', c.name, 'SKILL.md');
    const fm = s => (s.match(/^---\n[\s\S]*?\n---\n/) || [''])[0];
    const was = rd(path.join(snap, 'SKILL.md'));
    let now = rd(docP);
    if (was && now && fm(now) !== fm(was)) {
      // frontmatter is the load trigger — drift is hard-restored from the snapshot
      fs.writeFileSync(docP, now.replace(/^---\n[\s\S]*?\n---\n/, fm(was)));
      now = rd(docP);
      rec.frontmatter = 'restored';
    } else rec.frontmatter = 'ok';
    const m = now.match(/##\s*Thoughts([\s\S]*?)(?=\n##\s|\n#\s|$)/i);
    const desk = m ? m[1].split('\n').filter(l => /^\s*(-|###)/.test(l) && /\d{4}-\d{2}-\d{2}/.test(l)).length : 0;
    rec.desk = `${desk}/${c.limit}`; if (desk > c.limit) rec.breach = 'desk still over limit';
    rec.kbFrom = c.kb; rec.kbTo = +(fs.statSync(docP).size / 1024).toFixed(1);
  } catch (e) { rec.checkError = e.message; }
  reliefLog(root, home, rec);
}
function reliefStep(root, home, worldName) {
  const relief = reliefByWorld.get(worldName);
  const c = relief.queue.shift();
  if (!c) {
    relief.active = false; relief.current = null; relief.finishedAt = new Date().toISOString();
    reliefLog(root, home, { ts: relief.finishedAt, kind: 'relief-run', status: 'finished', done: relief.done, failed: relief.failed });
    return;
  }
  relief.current = c.name;
  // harm fence 1: snapshot the creature BEFORE its hat-session runs — minds
  // live outside git (the tracking law), so this scratch copy is the only undo in town
  const snap = path.join(root, home, 'tmp', relief.startedAt.slice(0, 10), 'relief-snapshot', c.name);
  try { fs.cpSync(path.join(root, '.opencode', 'skills', c.name), snap, { recursive: true }); }
  catch (e) { reliefLog(root, home, { ts: new Date().toISOString(), kind: 'relief-step', creature: c.name, status: 'snapshot-failed', what: e.message }); }
  reliefLog(root, home, { ts: new Date().toISOString(), kind: 'relief-step', creature: c.name, status: 'start', desk: `${c.thoughts}/${c.limit}`, kb: c.kb });
  const cli = detectReliefCli();
  if (!cli) { relief.failed.push(c.name); reliefLog(root, home, { ts: new Date().toISOString(), kind: 'relief-step', creature: c.name, status: 'spawn-failed', what: 'no relief CLI on PATH' }); return reliefStep(root, home, worldName); }
  let child;
  try { child = spawn(cli.bin, cli.args(reliefBrief(c, home)), { cwd: root, stdio: 'ignore' }); }
  catch (e) { relief.failed.push(c.name); reliefLog(root, home, { ts: new Date().toISOString(), kind: 'relief-step', creature: c.name, status: 'spawn-failed', what: e.message }); return reliefStep(root, home, worldName); }
  let timed = false, settled = false; // 'error' AND 'close' both fire on failed spawns — settle once
  const killer = setTimeout(() => { timed = true; child.kill('SIGKILL'); }, 10 * 60 * 1000); // 10 min per creature, then the next
  const settle = (status, extra) => {
    if (settled) return; settled = true; clearTimeout(killer);
    (status === 'done' ? relief.done : relief.failed).push(c.name);
    reliefLog(root, home, Object.assign({ ts: new Date().toISOString(), kind: 'relief-step', creature: c.name, status }, extra));
    if (status === 'done') reliefCheck(root, home, c, snap); // harm fence 2: frontmatter byte-check + desk re-count
    reliefStep(root, home, worldName);
  };
  child.on('error', e => settle('error', { what: e.message }));
  child.on('close', code => settle(timed ? 'timeout' : code === 0 ? 'done' : 'failed', { code }));
}
// prefer opencode (the CLI the source spec was written against: "opencode run"); fall
// back to Claude Code's own headless mode (`claude -p`) so relief runs work on Claude-only machines too
function detectReliefCli() {
  const which = process.platform === 'win32' ? 'where' : 'which';
  if (spawnSync(which, ['opencode']).status === 0) return { bin: 'opencode', args: prompt => ['run', prompt] };
  if (spawnSync(which, ['claude']).status === 0) return { bin: 'claude', args: prompt => ['-p', prompt] };
  return null;
}

// ------------ serve / dump ------------
// --ensure: instruments stay lit (nature law 8 — while a session thinks, the
// board is up). Idempotent: up → register this world and say so; down → spawn the
// ONE global daemon detached and forget. Any session, for any world, may run this —
// human order 2026-09-20: "1 app in whole machine not multiple ... use sub path".
const ENSURE = args.includes('--ensure');
const STOP = args.includes('--stop');
if (STOP) {
  // post the resident (global) board to sleep — affects every registered world at
  // once, which is the accepted tradeoff of one app for the whole machine.
  const rq = http.request({ host: '127.0.0.1', port: PORT, path: '/shutdown', method: 'POST', timeout: 1500 },
    r => { r.resume(); process.stdout.write(`tempest told to sleep :${PORT}\n`); });
  rq.on('timeout', () => { rq.destroy(); process.stdout.write(`tempest unresponsive on :${PORT}\n`); });
  rq.on('error', () => process.stdout.write(`tempest not up on :${PORT}\n`));
  rq.end();
} else if (ENSURE) {
  if (!fs.existsSync(path.join(COLONY, '.isekai')) && !fs.existsSync(path.join(COLONY, '.convention-zero'))) {
    process.stdout.write(`no .isekai/ world at ${COLONY} — run /isekai first\n`); process.exit(1);
  }
  const entry = registerWorld(COLONY);
  const probe = http.createServer();
  probe.once('error', () => { probe.close(); process.stdout.write(`tempest ${entry.name} already lit — http://localhost:${PORT}/${entry.name}/\n`); });
  probe.once('listening', () => {
    probe.close(() => {
      // No COLONY arg for the daemon — it serves every registered world, not just
      // the one that happened to light it; each request resolves its own root.
      const child = spawn(process.execPath, [__filename, '--port', String(PORT)].concat(IMMORTAL ? ['--immortal'] : []).concat(ttlIdx > -1 ? ['--ttl', String(TTL_MIN)] : []),
        { detached: true, stdio: 'ignore' });
      child.unref();
      process.stdout.write(`tempest ${entry.name} lit — http://localhost:${PORT}/${entry.name}/\n`);
    });
  });
  probe.listen(PORT, '127.0.0.1');
} else if (JSON_MODE) {
  process.stdout.write(JSON.stringify(harvest(COLONY), null, 2) + '\n');
} else if (JSON_GLOBAL_MODE) {
  // One-shot, no daemon needed — for scripts/tests to verify the global panel's own
  // numbers directly instead of scraping rendered HTML (Nature 9: compute it, check it).
  process.stdout.write(JSON.stringify({ claude: harvestClaudeGlobal(), opencode: harvestOpencodeGlobal() }, null, 2) + '\n');
} else {
  // default: run the ONE global board in the foreground (the docker-compose shape —
  // a long-lived process, not a spawn-and-forget heartbeat). Every world it knows
  // about (the registry) is served from this single process, one sub-path each.
  const server = http.createServer((req, res) => {
    lastTouch = Date.now();
    const url = new URL(req.url, `http://127.0.0.1:${PORT}`);
    const parts = url.pathname.split('/').filter(Boolean); // [] | [name] | [name, action]

    if (req.method === 'POST' && url.pathname === '/shutdown') {
      res.writeHead(200); res.end('sleeping'); server.close(() => process.exit(0)); return;
    }
    if (url.pathname === '/pulse' || (parts.length === 2 && parts[1] === 'pulse')) { res.writeHead(204); res.end(); return; }

    if (req.method === 'GET' && parts.length === 0) {
      res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
      res.end(renderIndex(prunedRegistry())); return;
    }

    const worldName = parts[0], action = parts[1] || '';
    const root = worldRootForName(worldName);
    if (!root) { res.writeHead(404, { 'Content-Type': 'text/plain' }); res.end(`unknown world: ${worldName} — is it registered? run /isekai or tempest --ensure in it first`); return; }
    const home = homeOf(root);

    if (req.method === 'GET' && !action) {
      res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
      res.end(render(harvest(root))); return;
    }
    if (req.method === 'POST' && action === 'holidays') {
      const d = harvest(root);
      const relief = reliefByWorld.get(worldName);
      if (relief && relief.active) { res.writeHead(200, { 'Content-Type': 'application/json' }); res.end(JSON.stringify({ stressed: stressedOf(d).length, relief: { refused: true, launched: true } })); return; }
      const worklist = stressedOf(d);
      const day = new Date().toISOString().slice(0, 10);
      const dayDir = path.join(root, home, 'tmp', day); fs.mkdirSync(dayDir, { recursive: true });
      const file = path.join(dayDir, 'holidays.md');
      fs.writeFileSync(file, `# Relief worklist — ${day}\n\n` + (worklist.map(c => `- ${c.name}: thoughts ${c.thoughts}/${c.limit}, ${c.kb}KB/${DIET_KB}KB`).join('\n') || '(nothing over the diet)') + '\n');
      if (!worklist.length) { res.writeHead(200, { 'Content-Type': 'application/json' }); res.end(JSON.stringify({ stressed: 0 })); return; }
      reliefByWorld.set(worldName, { active: true, queue: worklist.slice(), done: [], failed: [], current: null, total: worklist.length, startedAt: new Date().toISOString() });
      reliefStep(root, home, worldName);
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ stressed: worklist.length, file, relief: { launched: true } })); return;
    }
    if (req.method === 'POST' && action === 'party') {
      const d = harvest(root);
      const cf = path.join(root, home, 'metrics', 'celebrations.jsonl');
      fs.mkdirSync(path.dirname(cf), { recursive: true });
      fs.appendFileSync(cf, JSON.stringify({ ts: new Date().toISOString(), kind: 'party', births: d.genesisWatch }) + '\n');
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ births: d.genesisWatch })); return;
    }
    if (req.method === 'GET' && action === 'relief') {
      const relief = reliefByWorld.get(worldName);
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify(relief ? Object.assign({}, relief, { total: relief.total, remaining: relief.queue.map(c => c.name) }) : { total: 0, active: false })); return;
    }
    res.writeHead(404, { 'Content-Type': 'text/plain' }); res.end('not found');
  });
  server.listen(PORT, '127.0.0.1', () => process.stdout.write(`tempest — http://localhost:${PORT}/\n`));
  if (!IMMORTAL) {
    setInterval(() => { if ((Date.now() - lastTouch) >= TTL_MS) { process.stdout.write(`tempest sleeping (${TTL_MIN}m silence)\n`); process.exit(0); } }, 60000).unref();
  }
}
