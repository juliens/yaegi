#!/usr/bin/env bash
# oracle: differential check of yaegi against the Go toolchain.
#
# Usage: scripts/oracle.sh file.go [file.go...]
#        ORACLE_YAEGI_FLAGS=-unsafe scripts/oracle.sh file.go
#
# For each program, compares stdout, stderr presence and exit status of `go run` with `yaegi run`
# built from the current tree. Prints one line per file (MATCH / DIFF / GOFAIL) and a diff for DIFF.
# Exit status is 0 only if every file matches.
#
# GOFAIL means the program does not run under `go run` itself (invalid reproducer, or an expected
# compile error): the oracle cannot judge it.

set -uo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d "${TMPDIR:-/tmp}/oracle.XXXXXX")"
trap 'rm -rf "$work"' EXIT

timeout_s="${ORACLE_TIMEOUT:-20}"

(cd "$root" && go build -o "$work/yaegi" ./cmd/yaegi) || {
	echo "oracle: cannot build yaegi" >&2
	exit 2
}

# run executes a command with a timeout and records its output and exit status.
# The builtin println writes to stderr with gc but to stdout with yaegi, so both streams are
# compared together. Go-specific noise is dropped: the "exit status N" line printed by go run,
# and goroutine stack traces after a panic (only the panic message is compared).
run() {
	local out="$1"
	shift
	perl -e 'alarm shift; exec @ARGV' "$timeout_s" "$@" >"$out.stderr" 2>&1
	echo $? >"$out.status"
	grep -v '^exit status [0-9]*$' "$out.stderr" |
		sed '/^goroutine [0-9]* \[/,$d; /^\[signal /d' >"$out.stdout"
}

status=0
for f in "$@"; do
	if [[ ! -f "$f" ]]; then
		echo "NOFILE $f"
		status=1
		continue
	fi
	abs="$(cd "$(dirname "$f")" && pwd)/$(basename "$f")"
	run "$work/go" go run "$abs"
	# shellcheck disable=SC2086 # ORACLE_YAEGI_FLAGS is a list of flags.
	run "$work/yg" "$work/yaegi" run ${ORACLE_YAEGI_FLAGS:-} "$abs"

	go_status="$(cat "$work/go.status")"
	yg_status="$(cat "$work/yg.status")"

	if [[ "$go_status" != 0 ]] && grep -q '^# command-line-arguments' "$work/go.stderr"; then
		echo "GOFAIL $f"
		sed 's/^/    go: /' "$work/go.stderr" | head -5
		status=1
		continue
	fi

	# go run reports the program's own exit code as its output, then exits 1: compare success only.
	go_ok=$([[ "$go_status" == 0 ]] && echo y || echo n)
	yg_ok=$([[ "$yg_status" == 0 ]] && echo y || echo n)

	# Panic traces have a different shape: gc prints "panic: msg", yaegi prints one
	# "file:line: panic: fn(...)" line per frame then "run: msg". Compare output before the panic
	# plus the message only.
	if grep -q '^panic: ' "$work/go.stdout"; then
		msg="$(grep -m1 '^panic: ' "$work/go.stdout" | sed 's/^panic: //; s/ \[recovered\]$//')"
		sed -i '' "/^panic: /,\$d" "$work/go.stdout"
		echo "panic: $msg" >>"$work/go.stdout"
		ymsg="$(grep '^run: ' "$work/yg.stdout" | tail -1 | sed 's/^run: //')"
		sed -i '' '/: panic: .*(\.\.\.)$/,$d' "$work/yg.stdout"
		echo "panic: $ymsg" >>"$work/yg.stdout"
	fi

	if cmp -s "$work/go.stdout" "$work/yg.stdout" && [[ "$go_ok" == "$yg_ok" ]]; then
		echo "MATCH  $f"
		continue
	fi

	echo "DIFF   $f (exit go=$go_status yaegi=$yg_status)"
	diff -u --label go --label yaegi "$work/go.stdout" "$work/yg.stdout" | head -40 | sed 's/^/    /'
	status=1
done

exit "$status"
