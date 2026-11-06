# yaegi v2 (fork juliens/yaegi): instructions for Claude sessions

Branch `v2` of this fork is yaegi v2: the interpreter formerly named go-vm
(imported from traefik/go-vm in one commit, without its history; the
history stays in traefik/go-vm), module `github.com/traefik/yaegi/v2`.
Branch `master` keeps yaegi v0 (the current releases, v0.x). In the
comparison docs and tools, "yaegi" alone means yaegi v0 (the reference
engine, module `github.com/traefik/yaegi`). Nothing is pushed until Julien
says so.

Answer Julien in French. Code, comments, commit messages and documentation are in English (easy English for non-native readers).

**Start here: [README.md](README.md)** (what yaegi v2 is, how it works, modes),
**[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)** (layout, build, test, CI, new Go
releases) **and [TODO.md](TODO.md)** (the open work, by priority). "Continue"
means: take the first unchecked item of TODO.md, fix it with a regression
test, then tick it or remove it, and update the numbers of
[docs/RESULTS.md](docs/RESULTS.md) if they change. The history of the work is in git, not in a journal file.

## Setup (fresh clone, local or cloud)

```bash
sh scripts/setup.sh        # fork-guard git hooks + module downloads (network once) + build/vet
export GOFLAGS=-mod=readonly GOPROXY=off GOSUMDB=off
```

go.mod and go.sum are tidy: `-mod=readonly` (as in CI) works offline and fails
on a go.sum change instead of hiding it. `-mod=mod` still works.

`cmd/runtests` and `cmd/aotcheck` force `GOPROXY=off`: the module cache must be
filled by `scripts/setup.sh` first. If the sandbox refuses `~/Library/Caches/go-build`,
set `GOCACHE=$TMPDIR/gocache`.

## Layout and commands

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) (layout, build, run, test; the
comparison tools are the separate module `compare/`),
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), [docs/AOTGEN.md](docs/AOTGEN.md),
[docs/UNSAFE.md](docs/UNSAFE.md), and [compare/yaegi/REPORT.md](compare/yaegi/REPORT.md)
for what works under yaegi v0 vs yaegi v2.

## Non-negotiable rules

- Nothing may reach traefik/yaegi or traefik/traefik: no push, PR, issue or
  comment there; no bare `#N`, no `traefik/yaegi#N`, no upstream issue URL, no
  @mention. Upstream references are written `upstream-issue: N`.
- `scripts/fork-guard.sh` enforces this (git hooks installed by setup and by the
  SessionStart hook, PreToolUse hook in `.claude/settings.json`). Never bypass it
  (`--no-verify`, editing the hooks).
- The repository is the fork juliens/yaegi. The `upstream` remote
  (traefik/yaegi) is read-only: its push URL is deliberately invalid, do not
  change it. juliens/go-vm is an old, archived copy: never push there.
  References `traefik/go-vm#N` and `juliens/go-vm#N` in the history point to
  go-vm pull requests and stay as they are.
- One branch per task (`ai/<slug>` from `v2`), pushed to `origin`
  (juliens/yaegi) only, and only when Julien allows pushes. No merge into
  `v2` or `master`, no tags, no force-push, no PR unless Julien asks (then
  `--repo juliens/yaegi --base v2`).
- Never loosen a comparison or skip a test to raise a number; one regression
  test per fix (in `testdata/`, compared with `go run` by `go test`, or in
  `testdata/aot/`).
- Before each commit: `go build ./... && go vet ./... && go test ./...`, and
  `go run ./cmd/runtests -j 8 -timeout 20s -out $TMPDIR/c.tsv -summary $TMPDIR/c.txt`:
  master `Output:` ≥ 858/868, and no program that passes in
  `docs/conformance/results.tsv` fails. Update `docs/conformance/` when
  conformance changes.
- Commits: existing style (`vm: ...`, a body that explains why), with the
  trailer `Assisted-by: <model>`.
- Work that touches traefik/traefik (section 8 of TODO.md) needs Julien's
  explicit go.
