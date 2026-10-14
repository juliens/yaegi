# yaegi — fork juliens/yaegi

This checkout is a **private playground** on the fork `juliens/yaegi`. Everything happens on the fork;
nothing may be visible from `traefik/yaegi` (the upstream). Read the rules below before any git or gh command.

## Invisibility rules (non-negotiable)

1. **Never write to upstream.** No push, PR, issue, comment, review, label, reaction or `gh api` write on
   `traefik/yaegi`. Reading upstream (`git fetch upstream`, `gh issue list -R traefik/yaegi`, `gh api` GET) is fine.
2. **PRs target the fork only**: always `gh pr create --repo juliens/yaegi --base master ...`.
   `gh` and the GitHub UI default to the parent repository otherwise.
3. **No cross-references.** In commit messages, PR/issue titles and bodies, never write `#123`, `GH-123`,
   `traefik/yaegi#123`, or a `github.com/traefik/yaegi/issues|pull/...` URL: each creates a public
   "referenced this issue" event on upstream and notifies its subscribers. Write `upstream-issue: 123` instead.
4. **No @mentions** of anyone.
5. **No tags, no releases.** A pushed `v*` tag triggers `.github/workflows/release.yml`.
6. Do not modify the `upstream` remote: its push URL is deliberately invalid.

These rules are enforced by `scripts/fork-guard.sh`:
- as a Claude Code `PreToolUse` hook on Bash (`.claude/settings.json`), which blocks the offending command;
- as git `commit-msg` and `pre-push` hooks, installed with `scripts/fork-guard.sh install` (needed once per clone).

If the guard blocks you, fix the command; never work around the guard.

## Environment

- The repository must live at `$GOPATH/src/github.com/traefik/yaegi`: some `_test/` programs import local packages
  resolved through `build.Default.GOPATH`. Here `GOPATH=/Users/juliensalleyron/dev/go` (set in `.claude/settings.json`;
  the user's shell default is `~/go`, which makes ~10 `TestFile` cases fail with "not in GOPATH").
- Go toolchain: current stable (1.26 at the time of writing). The module declares `go 1.21`.

## Commands

```bash
go test ./interp -run 'TestFile/closure0.go'   # one _test program (~seconds)
go test ./interp                                # full interpreter suite (~8 minutes)
go test ./...                                   # everything
go test -race ./interp                          # CI also runs this
golangci-lint run                               # lint (config in .golangci.yml)
scripts/oracle.sh prog.go [...]                 # differential check: go run vs yaegi run
```

Known local failure: `cmd/yaegi` `TestYaegiCmdCancel` (Ctrl-C on `for {}` / `select {}` kills the process instead of
being handled). Not yet investigated: may be environmental (macOS, no TTY) or a real bug.

## The oracle

`scripts/oracle.sh` is the source of truth for correctness: the Go toolchain decides what a program must print,
not the agent. It compares combined stdout+stderr and success/failure of `go run` and `yaegi run` (built from the
current tree), normalizing `println` (stderr in gc, stdout in yaegi) and panic trace shapes.
Output per file: `MATCH`, `DIFF` (with a diff), `GOFAIL` (the program does not even run with `go run`: invalid
reproducer), `NOFILE`.

A fix is acceptable only if:
1. the reproducer is `MATCH` under the oracle;
2. it is added to `_test/` with its expected output (see below) and `go test ./interp` stays green;
3. `go test -race ./interp` and `golangci-lint run` stay green.

## Architecture (interp/)

Source → AST → three passes → closure-based execution:

| Step | File | Role |
|---|---|---|
| Parse | `ast.go` | go/parser AST → yaegi `node` tree (`kind`, `action`) |
| Global types/decls | `gta.go` | first pass over package-level declarations (types, funcs, vars) — enables out-of-order declarations |
| CFG + typing | `cfg.go` | main pass: scopes, type inference, constant folding, control-flow graph (`tnext`/`fnext`) |
| Type system | `type.go`, `typecheck.go`, `typestring.go` | yaegi `itype`, conversion to `reflect.Type`, checks |
| Generics | `generic.go` | generic instantiation (by AST rewriting) |
| Code generation | `run.go` | `genRun`/builders that turn each node into a Go closure (`exec` funcs) |
| Operators | `op.go` | **generated** by `internal/cmd/genop` — do not edit by hand, edit the generator then `go generate ./interp` |
| Values | `value.go` | frames and value access |
| Runtime glue | `interp.go`, `program.go`, `use.go`, `src.go`, `build.go` | public API, import of binary symbols, source loading, build tags |

`stdlib/` holds the generated wrappers exposing the Go standard library (`go1_21_*.go`, `go1_22_*.go`, produced by
`internal/cmd/extract`). They lag behind the current Go release.

## Tests in `_test/`

Each `_test/*.go` file is a full `package main` program; `TestFile` runs all of them. The **last comment** of the file
gives the expectation:

```go
// Output:
// 12
```

or, for an expected compile/run error (substring match):

```go
// Error:
// 17:11: undefined selector Get
```

A file without `Output:`/`Error:` is skipped. Name regression tests after the upstream issue: `_test/issue-1634.go`.

## Working conventions

- Follow the Traefik Go conventions (`backend-engineer` skill).
- One bug = one branch = one PR on the fork: branch `ai/fix-upstream-<N>` or `ai/<topic>`, from `master`.
- Commit messages: `interp: <what>`, a body explaining the root cause, then `upstream-issue: <N>` when relevant.
- Keep each change minimal and self-contained so that it can later be proposed upstream as-is.
- Sync with upstream only by reading: `git fetch upstream && git merge --ff-only upstream/master` (or rebase), then
  push to `origin`.
