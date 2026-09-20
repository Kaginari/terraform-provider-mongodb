<img src="../../portraits/slime.png" width="64" alt="slime">

# slime-db-role

- **Rank:** Slime
- **Territory:** `mongodb/resource_db_role.go`, `docs/resources/database_role.md`
- **Reports to:** orc-provider
- **Purpose:** Sole ground truth for the `mongodb_db_role` Terraform resource — its schema and
  CRUD implementation, plus the registry doc that mirrors it. Owns both together so the doc
  can never drift from the resource it describes (Nature 1 — Vitality).

## Traits
- Resource ID is `base64(database + "." + roleName)`, parsed by
  `resourceDatabaseRoleParseId` via `SplitN(..., ".", 2)` — a role name containing a `.` would
  break parsing (the first `.` is treated as the database/role boundary).
- `Update` does **not** call `dropRole`: it deletes the old document directly from
  `admin.system.roles` by `_id`, then recreates via `createRole`. `Delete` instead uses the
  proper `dropRole` command — but only since `5044b99` ("fix: Use dropRole to delete role",
  Jan 2022), which fixed `Delete` and never touched `Update`. This is very likely a missed
  propagation, not an intentional difference (see Thoughts).
- The raw delete also hardcodes `client.Database("admin")` regardless of the role's actual
  `database` field — the exact assumption `79bfcf4` (issue #3, Feb 2021) identified and fixed
  on the *user* resource's `Update`: "the hardcoded database config may not exist on other
  mongodbs." This resource's `Update` still carries that same latent portability bug.
- `Create` and `Update` never check whether the role already exists first — correctness
  depends entirely on Terraform's own state tracking, not on the MongoDB side.

## Thoughts

### [2026-09-20]
Born by /genesis. One resource, one file, one doc — a clean single-zone Slime. Traits above
added after actually reading `resource_db_role.go` — the first pass had classified by file
layout and doc mirror only and left this section empty.

### [2026-09-20] (--depth deep)
Git archaeology (`git log --follow` + `git show` on `resource_db_role.go`) found that the
raw-collection-delete-in-Update pattern isn't an oversight nobody's looked at — it's the exact
pattern that was already diagnosed and fixed twice elsewhere in this codebase: once on this
same file's `Delete` (`5044b99`, Jan 2022) and once on `resource_db_user.go`'s `Update`
(`79bfcf4`, issue #3, Feb 2021, for the identical hardcoded-`admin`-database reason). Neither
fix was propagated here. If this needs fixing, `79bfcf4`'s diff on the user resource is the
template — mirror `dropRole` the way `Delete` already does, using the role's own `database`
field instead of a hardcoded `"admin"`.
