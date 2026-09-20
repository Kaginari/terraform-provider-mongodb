---
name: palette-audit
description: Audit and fix an existing UI's color palette and typography for accessibility — computed OKLCH/CVD/contrast checks instead of eyeballing, dark AND light modes re-stepped separately, external palette/font references used as inspiration not gospel. Trigger on "review the colors", "is this accessible", "check contrast", "fix the palette", or when a dashboard/UI's dark and light modes reuse identical hex values.
license: Same terms as this repository.
---

# Palette audit — compute it, don't eyeball it

Born 2026-09-20 out of a real review of `tempest.js`'s dashboard, ordered by a human who
pointed at two reference sites (a Figma fonts guide, Color Hunt) and asked for the colors to
be reviewed. This is the distilled procedure, not the one-off findings — those live in
`tempest.js`'s own `<style>` block comment.

## When to reach for this

- Someone asks to review, audit, or fix a UI's colors, contrast, or "make it accessible."
- You're about to pick colors for a dashboard, chart, or any UI with more than one
  categorical state (race/type/status colors, series colors, node fills).
- You notice a dark-mode palette whose light-mode variant reuses the *identical* hex values
  with only the background swapped — that is almost always a bug, not a deliberate choice
  (see Finding below).

If you're building a *new* chart from scratch (not auditing an existing UI), check first
whether a `dataviz`-type skill is already loaded in this session — it covers chart-type
selection, mark specs, and interaction, which this Mind doesn't duplicate. This Mind is
specifically about the **color-and-type audit** step, self-contained so it doesn't depend on
that skill being present (e.g. when running under OpenCode, or a Claude session without it).

## The procedure

1. **Read the current palette and font stack as they're actually written** — the CSS custom
   properties or theme tokens, not a screenshot impression. Note every place a color is
   reused between light and dark mode without modification; that's the #1 real bug this audit
   finds.
2. **Compute, don't eyeball.** Run `scripts/validate_palette.js` (bundled beside this file —
   copy it out or run in place; it's a zero-dependency ES module) against the palette:
   ```
   node scripts/validate_palette.js "#hex,#hex,..." --mode dark --surface "#actual-bg-hex"
   node scripts/validate_palette.js "#hex,#hex,..." --mode light --surface "#actual-bg-hex"
   ```
   It reports five checks: OKLCH lightness band for the mode, chroma floor (below it a hue
   reads as gray), CVD separation (simulated protan/deutan/tritan Delta E between colors —
   adjacent pairs by default, `--pairs all` for scatter/bubble/map-shaped UIs where every
   pair can appear on screen together), the **normal-vision floor** (a hard gate — even
   full-color-vision readers must tell colors apart; below 15 is a genuine bug, not a nitpick),
   and WCAG contrast against the actual surface color.
3. **Dark and light are separate palettes, not one palette with a background swap.** Re-step
   each hue's lightness/chroma for its own mode's band and re-validate against that mode's
   actual surface. A palette that only validates in one mode is half-fixed.
4. **When a real constraint can't be fully solved, know when to stop.** Three or more mutually
   adjacent warm hues (orange/red/gold family) are a documented hard case — full CVD
   separation across all of them often isn't reachable without wrecking the hue identity.
   The acceptable fallback: land in the WARN band (Delta E 6–8, not below), pass the
   normal-vision hard floor (>=15, non-negotiable), and make sure the UI already has
   **secondary encoding** — a persistent label or tooltip next to every colored element, not
   color as the only signal. If that's already true, stop iterating; don't chase a
   mathematically-hard perfect pass at the expense of the actual task.
5. **Typography: match the UI's own register, don't impose generic advice.** General
   guidance (e.g. sans-serif for body text, serif for elegance) is a default for *prose*
   sites. A data-dense technical dashboard that already commits to a monospace aesthetic
   (tabular alignment, terminal/CRT theming) has a legitimate reason to stay monospace
   throughout — check whether the existing choice already serves the UI's actual job
   (numeric alignment, thematic consistency) before "fixing" it toward generic best practice.
6. **External references are inspiration, not import-and-done.** A site like Color Hunt gives
   candidate hues; a Figma-style fonts guide gives category defaults. Neither is validated for
   *your* surface colors or *your* specific hue clashes — always run step 2 on whatever you
   borrow. (Practical note: Color Hunt's palette swatches are client-rendered JS, so a plain
   page fetch won't return hex codes — you may need an actual browser tool to read them, or
   just use them as a naming/mood reference and pick your own validated hex.)
7. **Log the before/after and the reasoning**, not just the new hex values — a future session
   (or a future you) needs to know *why* a color looks the way it does, especially for a
   deliberately-imperfect WARN-band case, so nobody "fixes" it back into a worse state.

## Notes

- Run experiments in the world's own `.isekai/tmp/` (Law 5), never bare `/tmp` — and if the
  world's `.gitignore` doesn't already exclude `.isekai/tmp/*`, add that (keep `.gitkeep`
  tracked) before leaving scratch tooling there.
- `validate_palette.js`'s CLI entry point only fires when the file is literally named
  `validate_palette.js` (it checks `process.argv[1]` — see its own footer) and Node treats it
  as an ES module. If you copy it somewhere without a `"type": "module"` `package.json`
  beside it, add one (already bundled here) or Node will refuse the `export` syntax.
