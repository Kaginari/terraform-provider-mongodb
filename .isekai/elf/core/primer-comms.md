<img src="../../portraits/elf.png" width="64" alt="elf">

# elf-core's primer: human-facing release/PR/issue comms

Owned by elf-core — the world's voice for anything that leaves `.isekai/` and reaches a human
on GitHub (release notes, PR titles/descriptions, issue comments). This is the human-facing
half of Absolute Rule II's "one siphon that never narrows": full courtesy, plain language,
*and* a consistent visual shape so a maintainer or contributor skimming years of history can
tell at a glance what kind of change they're looking at without reading every word.

## Icons

Two icons, used as the lead character of a release-notes section heading, and optionally at
the start of a PR title or an issue-closing comment's first line when it helps orient the
reader fast:

- 🚀 **Evolution** — a new capability: a new resource, a new provider argument, a new
  authentication mode, anything a user couldn't do before. Pairs with a `feat/`-prefixed
  branch (see Branch choosing below).
- ⚙️ **Maintenance** — a fix, a doc correction, a CI/tooling repair, a dependency bump,
  anything that keeps existing capability correct or working rather than adding to it. Pairs
  with `fix/`, `docs/`, or `ci/` (see below).

Don't invent more icons per release without naming the need twice first (Nature 4, same as
any other addition to this world) — two categories, used consistently, beat five categories
half the team forgets.

### Release notes shape

Group a release's `## Highlights` (or equivalent top section) under these two headers when
both kinds of change are present in the same release; skip a header entirely if a release has
none of that kind, don't leave an empty section:

```markdown
## 🚀 Evolution
- New `mongodb_db_index` resource ...

## ⚙️ Maintenance
- Fixed `mongodb_db_user` update dropping and recreating the user (#46) ...
```

A release that's pure bugfixes gets only `## ⚙️ Maintenance`; a release that's pure new
capability gets only `## 🚀 Evolution`. Don't force both headers into every release just for
symmetry.

### Issue comments

When closing or commenting on an issue because a fix/feature landed, lead with the matching
icon before the summary sentence, e.g. `⚙️ Fixed in v1.0.1 — ...` or `🚀 Added in v1.0.1 — ...`.
Keeps a skimmed issue thread legible the same way the release notes are.

## Branch choosing

The prefix convention this world has already been using in practice, made explicit so a fresh
session doesn't have to reverse-engineer it from `git log`:

| Prefix | For | Icon it pairs with |
|---|---|---|
| `feat/` | a new resource, provider argument, or capability | 🚀 |
| `fix/` | a bug fix, behavior correction | ⚙️ |
| `docs/` | documentation-only changes, no code | ⚙️ |
| `ci/` | CI/workflow/tooling/release-pipeline changes | ⚙️ |

One branch, one concern — a branch that would need both a `feat/` and a `fix/` prefix is
almost always two PRs, not one (this world found that out the hard way once: the resource-ID
migration and the in-place-update fix landed as two separate PRs the same session specifically
so each stayed independently reviewable, rather than one branch trying to be both).

## Release checklist

Tagging and pushing is not the last step — `.goreleaser.yml` has `changelog.disable: true` (see
its own comment for why: this world's commit history isn't clean conventional-commit-per-line,
so an auto-generated changelog would be noisy, not useful), which means **the GitHub Release
GoReleaser creates has an empty body by default, every single time**, until a human or session
attaches real notes on top of it. This is not optional and it is easy to forget once a release
is flanked by five other things happening in the same stretch (found out the hard way on
v1.0.1: notes were drafted, the release shipped, and the attach step got lost in the shuffle of
a goreleaser-deprecation fix, an icon, and a README rewrite all landing around the same time —
caught only because the human checked the release page and asked where the notes were).

1. Draft the notes using the 🚀/⚙️ icon convention above, *before* tagging (so they're ready the
   moment the release exists, not written from memory afterward).
2. `git tag -a vX.Y.Z -m "..."` and `git push origin vX.Y.Z`.
3. Wait for `release.yml` to finish (`gh run list --workflow=release.yml --limit 1`).
4. **`gh release edit vX.Y.Z --notes-file <the draft from step 1>` — do this immediately, in
   the same breath as confirming the release succeeded, before moving on to anything else.**
   Verify it actually landed: `gh release view vX.Y.Z --json body --jq '.body'` should NOT come
   back empty. An empty result here means step 4 didn't happen, not that it's fine.
5. Only after step 4 is confirmed non-empty: move on to registry validation, issue comments, or
   whatever else the release triggers.

## Bot identity

Every PR, issue comment, and commit this world has produced so far shows up under the human's
own forge account (`ITMonta` on GitHub) — including everything a dispatched Court Body
sub-agent did, since sub-agents share this session's authentication with no separate identity
of their own. A forge (GitHub, GitLab, whichever this world's remote actually is) has no
concept of "which agent" acted, only "which authenticated account" did — so a human skimming
PR/MR history can't tell dispatched work from the human's own manual commits, or one Orc's
work from another's, just by looking at the forge.

**The fix**: a shared bot identity, one per world (not one per creature — that would mean
maintaining many bot accounts for little added clarity over what `tempest.js` already shows,
Nature 9), distinguishing "the world's automation did this" from "a human did this." The two
forges this world has needed so far get there by genuinely different mechanisms, not the same
mechanism under a different name — `.isekai/tools/bot-token.sh` is the forge-agnostic
entrypoint so a caller never needs to know which one is in play:

- **GitHub** — a GitHub App (`kaginari-mongodb-bot`, or whatever name was actually chosen —
  check `GITHUB_APP_NAME` in the credentials file below), installed on the repo, same mechanism
  `dependabot[bot]`/`renovate[bot]` use. Requires minting a short-lived (~1h) installation
  token per use (JWT-signed with the App's private key) — that's what
  `.isekai/tools/github-app-token.sh` does; `bot-token.sh` calls it automatically when
  `GITHUB_APP_ID`/`GITHUB_APP_PRIVATE_KEY_PATH`/`GITHUB_APP_INSTALLATION_ID` are set.
- **GitLab** — a Project or Group Access Token. Simpler than GitHub's flow: GitLab creates a
  distinct bot user (e.g. `project_123_bot_...`) the moment the token is created in its own
  UI/API, and the token *is* the credential directly — no signing, no per-call minting, it just
  stays valid until its own expiry (GitLab caps these at 1 year; renew before it lapses).
  `bot-token.sh` uses it directly whenever `GITLAB_BOT_TOKEN` is set. No wizard written for
  this yet — no GitLab remote has actually existed for this world so far (Nature 4: this is
  the mechanism's design, not a birth; the actual wizard gets written when a GitLab world
  names the need for real, not speculatively here).

**How to use it, once set up** (portable — plain shell, works under any harness, and works the
same way regardless of which forge is actually in play):
```sh
TOKEN=$(.isekai/tools/bot-token.sh)
GH_TOKEN="$TOKEN" gh pr create ...                 # GitHub
git -c http.extraHeader="PRIVATE-TOKEN: $TOKEN" push ...   # GitLab
git commit --author="<bot-name> <bot-identity-email>" ...  # either forge
```
None of the underlying credentials (GitHub App private key, GitLab access token value) live in
this repo, committed or otherwise. `.isekai/tools/setup-github-app.sh` (the wizard that
captures GitHub's, walking the human through GitHub's UI, first run 2026-09-21) writes them to
`~/.config/kaginari-mongodb-bot/github-app.env` on the machine it's run on — durable,
machine-local, outside any repo. A session on that same machine can `source` it before calling
`bot-token.sh`; a session on a *different* machine won't find them there and needs the human to
either re-run the wizard (idempotent — offers existing values as defaults, per the wizard
skill's own design) or copy the credentials file over (it's a credentials file — copy it the
way you'd copy any other secret, not casually).

**Status as of 2026-09-21: mechanism built, not yet completed.** `bot-token.sh` and
`github-app-token.sh` exist and are portable/tested-for-syntax; the human was walked through
creating the actual GitHub App via `/wizard`, but this file was written *before* confirmation
that the wizard run finished (App ID / installation ID / key in hand). GitLab's path is
designed but has no credentials to test against — no GitLab remote exists for this world yet.
A session picking this up should verify with the human whether GitHub setup actually completed
before assuming `bot-token.sh` will work — don't take this doc's existence as proof the
mechanism is live, per Nature 9 ("a document's claim about current state is a memory, not a
fact").

### Creature avatars on commits (extension, not set up yet)

The bot identity above is deliberately one identity for the whole world — but a *commit's*
avatar on GitHub/GitLab doesn't come from the authenticated account at all, it comes from
matching the commit's **author email** against a registered account or a **Gravatar** profile
(gravatar.com — a free, separate service, no forge account needed). That's a second, finer
layer this world can use without contradicting "one shared bot identity, not one per creature"
for PRs/comments: a per-creature *commit-author email*, each registered on Gravatar with that
creature's existing portrait (`.isekai/portraits/orc.png`, `slime.png`, etc.) as the profile
image. The PR/comment itself still shows the one bot identity; individual commits inside it can
show the specific creature's portrait in file history/blame.

Not built yet — needs a human to actually register each email on Gravatar (email-verification
click, can't be done by an agent). If/when set up: `git commit --author="orc-provider
<some-address+orc-provider@...> "` per creature, using whatever alias scheme the human sets up
(an email `+` alias under an existing address works fine for this — Gravatar just needs a
distinct address per identity, not a distinct inbox). Worth a `/wizard` run to actually do the
Gravatar registrations when/if this gets picked up for real, same shape as
`setup-github-app.sh`.

## Thoughts

### [2026-09-21]
Written on direct order (Law 1) after a full release cycle (v1.0.0) shipped without any of
this being codified — branch prefixes were chosen consistently by feel, not by a written rule,
and release notes had no visual structure beyond prose headers. Both worked, but only because
one session held the whole pattern in its own head; the point of a primer is that the *next*
session doesn't have to.

### [2026-09-21] (bot identity)
Human noticed GitHub PRs/comments never distinguished dispatched sub-agent work from the
human's own or from each other, asked whether "multiple identities per agent" was possible.
Answered honestly in the human tongue first (Absolute Rule III doesn't gate this file, but the
same discipline applies): plain multiple PATs under one account don't produce a distinct
GitHub-visible identity (GitHub attributes API actions to the account, not the token) — only a
GitHub App does, since it's a genuinely separate actor type. Scoped to one shared bot identity,
not one per creature, for the same reason `tempest.js` already covers per-agent distinction
without needing GitHub to also do it. Explicitly asked to keep the mechanism harness-agnostic
("make sure opencode compatible") — `github-app-token.sh` is plain bash/openssl/curl/jq for
exactly that reason, no Claude Code-specific tool calls anywhere in it.
