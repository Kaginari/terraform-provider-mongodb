<img src="../../portraits/slime.png" width="64" alt="slime">

# slime-db-user

- **Rank:** Slime
- **Territory:** `mongodb/resource_db_user.go`, `docs/resources/database_user.md`
- **Reports to:** orc-provider
- **Purpose:** Sole ground truth for the `mongodb_db_user` Terraform resource — its schema and
  CRUD implementation, plus the registry doc that mirrors it. Owns both together so the doc
  can never drift from the resource it describes (Nature 1 — Vitality).

## Traits
- Resource ID is `base64(database + "." + userName)` — same scheme, and same single-`.`-split
  fragility, as slime-db-role's (a username containing a `.` would break
  `resourceDatabaseUserParseId`).
- `Read` does `data.Set("password", data.Get("password"))` — a no-op re-set, not a real read.
  MongoDB doesn't return passwords, so this preserves whatever's already in state — meaning a
  password changed outside Terraform is never detected as drift.
- `Update` explicitly calls `dropUser` then recreates via `createUser` — not stylistic, a
  deliberate portability fix (`79bfcf4`, issue #3, Feb 2021): the original raw `system.users`
  collection delete hardcoded `client.Database("admin")`, which broke on deployments where
  that assumption doesn't hold. slime-db-role's `Update` still has the pre-fix pattern (see
  its Traits) — the same bug, unfixed, one file over.

## Thoughts

### [2026-09-20]
Born by /genesis. One resource, one file, one doc — a clean single-zone Slime. Traits above
added after actually reading `resource_db_user.go` — the first pass had classified by file
layout and doc mirror only and left this section empty.

### [2026-09-20] (--depth deep)
This resource is the *fixed* side of a bug that still exists in slime-db-role's `Update` — if
someone picks up that role bug, point them at `79bfcf4` here as the template fix.
