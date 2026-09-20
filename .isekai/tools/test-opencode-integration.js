#!/usr/bin/env node
'use strict';
/**
 * test-opencode-integration.js — deterministic test: does tempest.js's OpenCode reader
 * actually reflect reality? (Nature 9: compute it, check it, don't eyeball it.)
 *
 * Runs one real, fixed OpenCode message; gets ground-truth token counts from OpenCode's
 * own `export` command (no node:sqlite needed for that — OpenCode's own runtime reads its
 * database fine); then asks tempest.js what IT thinks OpenCode usage looks like
 * (`--json-global`) and checks the two agree — or, on a system Node without node:sqlite
 * (built 2026-09-20: Node 22.5+ only), checks that tempest.js says so honestly instead of
 * silently reporting zero.
 *
 * Usage:
 *   node test-opencode-integration.js [--opencode <path-to-opencode-bin>] [--dir <target>]
 *
 * Exit 0 on PASS, 1 on FAIL — every number printed is computed here, not assumed.
 */
const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');

function arg(name, fallback) {
  const i = process.argv.indexOf(name);
  return i > -1 ? process.argv[i + 1] : fallback;
}
const TEMPEST = path.join(__dirname, 'tempest.js');
const TARGET = path.resolve(arg('--dir', process.cwd()));
const OPENCODE = arg('--opencode',
  fs.existsSync(path.join(__dirname, '..', 'tmp', 'opencode-test', 'node_modules', '.bin', 'opencode'))
    ? path.join(__dirname, '..', 'tmp', 'opencode-test', 'node_modules', '.bin', 'opencode')
    : 'opencode');

let failures = 0;
const check = (label, cond, detail) => {
  console.log(`  [${cond ? 'PASS' : 'FAIL'}] ${label}${detail ? ' — ' + detail : ''}`);
  if (!cond) failures++;
};

console.log(`target: ${TARGET}`);
console.log(`opencode bin: ${OPENCODE}`);

// 1. Run one real, fixed, cheap message through OpenCode.
const title = `tempest-integration-test-${Date.now()}`;
console.log(`\n1. Running a real OpenCode session (title: "${title}")...`);
let runOk = true;
try {
  execFileSync(OPENCODE, ['run', 'reply with exactly the word OK', '-m', 'opencode/big-pickle',
    '--dir', TARGET, '--title', title], { stdio: 'pipe', timeout: 60000 });
} catch (e) {
  runOk = false;
  console.log(`  run failed: ${e.message}`);
}
check('opencode run completed', runOk);
if (!runOk) { console.log('\nCannot continue without a real session. FAILED.'); process.exit(1); }

// 2. Find that exact session and get OpenCode's own ground-truth token counts.
console.log('\n2. Reading ground truth back from OpenCode itself (session list + export)...');
let session = null, ground = null;
try {
  const list = JSON.parse(execFileSync(OPENCODE, ['session', 'list', '-n', '10', '--format', 'json'],
    { encoding: 'utf8', timeout: 15000 }));
  session = list.find(s => s.title === title || (s.title || '').includes(title));
} catch (e) { console.log(`  session list failed: ${e.message}`); }
check('found the session just run', !!session, session ? session.id : 'no match');

if (session) {
  try {
    const raw = execFileSync(OPENCODE, ['export', session.id], { encoding: 'utf8', timeout: 15000 });
    const jsonStart = raw.indexOf('{');
    ground = JSON.parse(raw.slice(jsonStart)).info.tokens;
  } catch (e) { console.log(`  export failed: ${e.message}`); }
}
check('got real token counts from opencode export', !!ground,
  ground ? `input=${ground.input} output=${ground.output} cache.read=${ground.cache.read} cache.write=${ground.cache.write}` : '');

// 3. Ask tempest.js what it thinks, and compare.
console.log("\n3. Asking tempest.js's own harvestOpencodeGlobal() (--json-global)...");
let tempestOut = null;
try {
  tempestOut = JSON.parse(execFileSync(process.execPath, [TEMPEST, TARGET, '--json-global'],
    { encoding: 'utf8', timeout: 15000 })).opencode;
} catch (e) { console.log(`  tempest --json-global failed: ${e.message}`); }
check('tempest.js ran and returned opencode data', !!tempestOut);

if (tempestOut) {
  check('tempest correctly sees opencode.db exists', tempestOut.dbExists === true);
  let sqliteWorks = true;
  try { require('node:sqlite'); } catch { sqliteWorks = false; }
  if (sqliteWorks) {
    check('node:sqlite is available on this Node — expecting real numbers, not just honesty',
      tempestOut.available === true, `node ${process.version}`);
    if (tempestOut.available && ground) {
      check('tempest\'s totals are at least as large as this one session\'s ground truth',
        tempestOut.totalIn >= (ground.input + ground.cache.read + ground.cache.write) && tempestOut.totalOut >= ground.output,
        `tempest in=${tempestOut.totalIn} out=${tempestOut.totalOut}`);
    }
  } else {
    // The documented, expected case on this machine (Node < 22.5): real data exists
    // (dbExists=true, confirmed above) but this runtime genuinely cannot read it. The
    // correct behavior is honesty, not a silent zero — that's what this checks.
    check('node:sqlite genuinely unavailable here — tempest must say so, not report zero',
      tempestOut.available === false && !!tempestOut.error, `node ${process.version}: ${tempestOut.error}`);
    console.log(`  (expected on this machine: Node ${process.version} < 22.5. Ground truth via` +
      ` opencode export confirms real data exists — tempest.js is being honest about not reading it, not wrong.)`);
  }
}

console.log(`\n${failures === 0 ? 'ALL CHECKS PASS' : failures + ' CHECK(S) FAILED'}`);
process.exit(failures === 0 ? 0 : 1);
