<img src="../../portraits/slime.png" width="64" alt="slime">

# slime-ci

- **Rank:** Slime
- **Territory:** `.github/workflows/*.yml`, `.golangci.yml`
- **Reports to:** orc-provider
- **Purpose:** Sole ground truth for this repo's CI tooling — the pinned linter/action
  versions that decide whether a correct PR shows green or red for reasons that have nothing
  to do with its diff. A distinct zone from the Go source Slimes: different language (YAML),
  different failure mode (drifts silently — nothing in `go build`/`go test`/`go vet` can ever
  detect a stale CI pin, since those run against whatever toolchain is actually on PATH, not
  whatever the runner will use), and the reason it went unowned until now: `.github/` was
  never anyone's territory, and CI status was never actually captured as an instrument even
  though isekai.md's own Instruments section names "build/CI status" as a category from the
  start (see log.md 2026-09-21 birth entry for the root-cause writeup).

## Traits
- Pinned versions rot **without any code change on either side** — the diff that breaks CI
  isn't in this repo, it's the runner image's default Go toolchain moving forward underneath
  a version-pinned linter that never updated to match. `go test ./...` staying green the whole
  time is not evidence CI tooling is healthy.
- `release.yml` (tag-triggered, not run on every PR) still carries the same stale-pin shape
  this Slime was born to catch — `actions/checkout@v2`, `actions/setup-go@v2`,
  `paultyng/ghaction-import-gpg@v2.1.0`, `goreleaser/goreleaser-action@v2` — caught live by
  `.isekai/tools/ci-check.sh` in the same session this Slime was born, deliberately left
  unfixed (lower blast radius — only fires on a version tag push — and out of scope for the
  session that was fixing the PR-blocking `golangci.yml` job). Next session touching CI: fix
  this before it fires on an actual release, not after.
- `.golangci.yml`'s linter list is itself a drift surface independent of the action pins: a
  modern golangci-lint refuses to even start if the config names a linter that's since been
  renamed or removed (hit this in the same session — `deadcode`/`varcheck`/`interfacer`/`vet`/
  `vetshadow` were all stale names). Check the config loads under whatever golangci-lint
  version the workflow actually pins, not just that the YAML is well-formed.

## Thoughts

### [2026-09-21]
Born on direct order (Law 1) after a session's PR (`fix/in-place-update-and-drift-detection`,
also `fix/resource-id-migration`) showed a red `golangci-lint` check neither PR's diff could
explain. Root cause: golangci-lint-action@v2 pinned to golangci-lint v1.30 (mid-2020) with
`skip-go-installation: true`, silently drifting onto whatever Go the `ubuntu-latest` runner
now ships by default (1.24.x) — a version-skew crash before a single lint rule runs. Genesis
condition was already met and had been for a while: `.github/workflows/*.yml` and
`.golangci.yml` were unrouted territory (Nature 4) the whole time, and isekai.md's own
Instruments section had promised "build/CI status" as a capturable signal since this world's
reincarnation without anyone ever wiring up an instrument that actually captures it — so nothing
in the convention as it stood *could* have caught this proactively; it wasn't a rule that got
skipped, it was a territory that was never claimed. Fix: this Slime claims the territory,
`.isekai/tools/ci-check.sh` is the instrument that now actually captures the "build/CI status"
signal isekai.md always described, and the `golangci.yml`/`.golangci.yml` fix itself landed as
a separate PR the same session. Whether this closes the gap for good or only until the next
drift (a year from now, some other pinned tool) depends on a future session actually running
the instrument before trusting CI green/red — the doc can name the risk, it can't run itself.
