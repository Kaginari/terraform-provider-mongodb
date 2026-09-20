<img src="../../portraits/slime.png" width="64" alt="slime">

# slime-config

- **Rank:** Slime
- **Territory:** `mongodb/config.go`
- **Reports to:** orc-provider
- **Purpose:** Sole ground truth for the MongoDB client/connection and authentication config —
  the one integration point every resource depends on to reach a MongoDB server. The
  most-changed file in the codebase (16 commits), which is exactly the shape a Slime exists
  for: one narrow zone, one fact anyone can trust about it.

## Traits
- `insecure_skip_verify` only takes effect when `certificate` is also set — the custom TLS
  path that reads it (`MongoClient()`) is nested inside `if c.Certificate != ""`. Setting
  `insecure_skip_verify = true` with `ssl = true` but no `certificate` does nothing. This has
  been true since the field was introduced (`f8061aa`, May 2021) — it's the original design,
  not drift, though the schema description ("ignore hostname verification") never discloses
  the scoping.
- Connect timeout is hardcoded to 10s (`MaxConnLifetime: 10` in `provider.go`, consumed as
  `conf.MaxConnLifetime*time.Second` in `MongoClientInit`) — not exposed as a schema field.
- All resource CRUD authenticates as one identity: `AuthSource: c.DB` (the provider's
  `auth_database`, default `admin`) with the single provider-level username/password. There is
  no per-resource credential.
- The URI is hand-built (`mongodb://host:port` + query args joined by `addArgs`, which
  special-cases the first param with a literal `/?` prefix) — not via the driver's own
  connection-string builder.
- `getTLSConfigWithAllServerCertificates` exists to work around a documented MongoDB Go driver
  limitation: the driver only loads the *first* CA cert from a PEM bundle, so this function
  manually appends every cert found in it.

## Thoughts

### [2026-09-20]
Born by /genesis. Config/connection logic is a distinct, single-purpose zone separate from
either resource's CRUD — worth its own ground truth rather than folding into a resource slime.
Traits above added after actually reading `config.go` — the first pass had classified by file
layout and commit count only and left this section empty.

### [2026-09-20] (--depth deep)
Certificate handling has been redesigned at least twice: a file-path-based scheme
(`CertPath` pointing at `ca.pem`/`cert.pem`/`key.pem` on disk, `9cd29fd`) was later replaced
by the current PEM-content string field `certificate` (`1425707` "Change Ca to string instead
of path"). This area churns — re-check history before changing certificate handling again,
not just the current diff.
