#!/usr/bin/env bash
# fork-guard: keeps all work on the juliens/yaegi fork invisible from traefik/yaegi.
#
# Modes:
#   fork-guard.sh claude-hook          Claude Code PreToolUse hook (Bash tool JSON on stdin).
#   fork-guard.sh commit-msg <file>    git commit-msg hook.
#   fork-guard.sh pre-push <remote> <url>  git pre-push hook (refs on stdin).
#   fork-guard.sh install              install the git hooks in .git/hooks.
#
# A blocked action exits with status 2 (Claude hook) or 1 (git hooks) and explains why on stderr.

set -euo pipefail

upstream_slug='traefik/yaegi'

deny() {
	echo "fork-guard: $1" >&2
	echo "fork-guard: this repository is a private playground; nothing may reach ${upstream_slug}. See CLAUDE.md." >&2
	exit "${deny_status:-2}"
}

# check_text rejects text that would create a cross-reference or a notification on upstream.
check_text() {
	local text="$1"

	if grep -Eq "${upstream_slug}#[0-9]+" <<<"$text"; then
		deny "reference '${upstream_slug}#N' would show up on the upstream issue. Use 'upstream-issue: N'."
	fi
	if grep -Eq "github\.com/${upstream_slug}/(issues|pull|discussions)/" <<<"$text"; then
		deny "link to an upstream issue/PR would create a public backlink. Use 'upstream-issue: N'."
	fi
	if grep -Eq '(^|[^A-Za-z0-9&/_-])(GH-|#)[0-9]+' <<<"$text"; then
		deny "bare '#N' reference is forbidden (it may resolve to the upstream tracker). Use 'upstream-issue: N'."
	fi
	if grep -Eq '(^|[[:space:](])@[A-Za-z0-9][A-Za-z0-9-]*' <<<"$text"; then
		deny "@mention would notify a GitHub user."
	fi
}

claude_hook() {
	local cmd
	cmd="$(jq -r '.tool_input.command // empty')"
	[[ -z "$cmd" ]] && exit 0

	# Pushes: never to upstream, never tags.
	if grep -Eq '(^|[;&|[:space:]])git([[:space:]]+-C[[:space:]]+[^[:space:]]+)?[[:space:]]+push' <<<"$cmd"; then
		if grep -Eq "push([[:space:]]+-[^[:space:]]+)*[[:space:]]+upstream([[:space:]]|$)|${upstream_slug}" <<<"$cmd"; then
			deny "git push towards upstream is forbidden."
		fi
		if grep -Eq -- '--tags|--follow-tags|refs/tags|--mirror' <<<"$cmd"; then
			deny "pushing tags is forbidden (it would trigger the release workflow)."
		fi
	fi

	# Tags: listing is fine, creating or deleting is not.
	if grep -Eq '(^|[;&|[:space:]])git[[:space:]]+tag[[:space:]]+[^-[:space:]]|git[[:space:]]+tag[[:space:]]+-(a|s|d|f|m)' <<<"$cmd"; then
		deny "creating tags is forbidden."
	fi

	# Remote config: the upstream push URL must stay disabled.
	if grep -Eq 'git[[:space:]]+remote[[:space:]]+(set-url|remove|rm|rename)([[:space:]]+-[^[:space:]]+)*[[:space:]]+upstream([[:space:]]|$)' <<<"$cmd"; then
		deny "changing the upstream remote URL is forbidden."
	fi

	if grep -Eq '(^|[;&|[:space:]])gh[[:space:]]' <<<"$cmd"; then
		# Any gh command naming upstream must be read-only.
		if grep -Eq "${upstream_slug}([^A-Za-z0-9_-]|$)" <<<"$cmd"; then
			if grep -Eq '(pr|issue|release|label|repo|workflow|run|secret|variable)[[:space:]]+(create|comment|edit|close|reopen|merge|review|delete|lock|unlock|pin|transfer|develop|ready|fork|sync|enable|disable|set|rename|archive)' <<<"$cmd"; then
				deny "gh write command targeting ${upstream_slug}."
			fi
			if grep -Eq 'gh[[:space:]]+api' <<<"$cmd" && grep -Eq -- '(-X|--method)[[:space:]]*(POST|PATCH|PUT|DELETE)|(^|[[:space:]])(-f|-F|--field|--raw-field|--input)[[:space:]]' <<<"$cmd"; then
				deny "gh api write call targeting ${upstream_slug}."
			fi
		fi
		# PR creation must name the fork explicitly (gh defaults to the parent repository).
		if grep -Eq 'gh[[:space:]]+pr[[:space:]]+create' <<<"$cmd" && ! grep -Eq -- '(--repo|-R)[[:space:]=]+juliens/yaegi' <<<"$cmd"; then
			deny "'gh pr create' must pass '--repo juliens/yaegi' explicitly."
		fi
	fi

	# Text that ends up on GitHub: commit messages and gh pr/issue bodies.
	if grep -Eq 'git[[:space:]]+commit|gh[[:space:]]+(pr|issue)[[:space:]]+(create|comment|edit|review)' <<<"$cmd"; then
		check_text "$cmd"
	fi

	exit 0
}

commit_msg() {
	deny_status=1
	# Ignore git comment lines.
	check_text "$(grep -v '^#' "$1")"
}

pre_push() {
	deny_status=1
	local remote="$1" url="$2"
	if [[ "$remote" == "upstream" || "$url" == *"${upstream_slug}"* ]]; then
		deny "push to '${remote}' (${url}) is forbidden."
	fi
	local local_ref local_sha remote_ref remote_sha
	while read -r local_ref local_sha remote_ref remote_sha; do
		if [[ "$remote_ref" == refs/tags/* ]]; then
			deny "pushing tag ${remote_ref} is forbidden."
		fi
	done
}

install() {
	local hooks_dir
	hooks_dir="$(git rev-parse --git-path hooks)"
	mkdir -p "$hooks_dir"
	printf '#!/bin/sh\nexec "$(git rev-parse --show-toplevel)/scripts/fork-guard.sh" commit-msg "$@"\n' >"$hooks_dir/commit-msg"
	printf '#!/bin/sh\nexec "$(git rev-parse --show-toplevel)/scripts/fork-guard.sh" pre-push "$@"\n' >"$hooks_dir/pre-push"
	chmod +x "$hooks_dir/commit-msg" "$hooks_dir/pre-push"
	echo "fork-guard: installed commit-msg and pre-push hooks in ${hooks_dir}"
}

case "${1:-}" in
claude-hook) claude_hook ;;
commit-msg) commit_msg "$2" ;;
pre-push) pre_push "$2" "$3" ;;
install) install ;;
*)
	echo "usage: $0 {claude-hook|commit-msg <file>|pre-push <remote> <url>|install}" >&2
	exit 64
	;;
esac
