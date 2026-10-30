#!/usr/bin/env bash
# PreToolUse guard for the Bash tool. Implements the risk classes from CLAUDE.md:
#   BLOCK -> exit 2, the agent cannot run the command
#   ASK   -> permissionDecision "ask", the user confirms in the UI
# Matching is regex over the raw command, so quoted text can trigger false
# positives and obfuscated commands can slip through. It is a safety net, not a sandbox.
set -u

if ! command -v jq >/dev/null 2>&1; then
  echo "guard-bash: jq not found, blocking command (fail closed)" >&2
  exit 2
fi

cmd=$(jq -r '.tool_input.command // empty' | tr '\n' ' ')
[ -z "$cmd" ] && exit 0

S='(^|[;&|(`[:space:]])'           # start of a word in a shell command
G="${S}git( +-[cC] +[^ ]+)*"       # git, including git -C dir / -c key=val
RMREC="${S}rm( +-[^ ]+)* +(-[a-zA-Z]*[rR][a-zA-Z]*|--recursive)"

block_rules=(
  "${G} +push( [^;&|]*)?( --force| --mirror| -[a-zA-Z]*f( |$)| \+[^ ]+)|force push / mirror push"
  "${G} +(filter-branch|filter-repo|update-ref +-d|reflog +(expire|delete))|rewriting or destroying git history"
  "${G} +(commit|push|merge|rebase) .*--no-verify|skipping git hooks"
  "${G} +config .*(core\.hooksPath|--global)|changing git hooks path or global git config"
  "${RMREC}( +-[^ ]+)* +(/|/\*|~|~/|~/\*|\\\$HOME|\.|\./|\.\.|\.\./|\*|\.git)( |;|$)|rm -r on root, home, project root or .git"
  "${S}(sudo|doas|mkfs[.a-z0-9]*|shutdown|reboot)( |$)|root privileges or destroying the system"
  "${S}dd .*of=/dev/|writing directly to a device"
  "${S}chmod +-R +0?777|chmod -R 777"
  "(curl|wget)[^|;&]*\| *(sudo +)?(ba|z|da)?sh|piping a download into a shell"
  "\.env([^a-zA-Z0-9._-]|$)|access to .env (secrets)"
  "\.env\.(local|prod|production|staging|dev)|access to .env.* (secrets)"
  "\\\$\{?BOT_TOKEN|reading BOT_TOKEN"
  "${S}(env|printenv)( *$| *[;|&])|dumping environment variables"
  "api\.telegram\.org|direct Telegram API call (uses the production token)"
  "${S}gh +(repo +delete|secret|auth +token)|deleting the repo or touching GitHub secrets/tokens"
  "disableAllHooks|disabling hooks"
)

ask_rules=(
  "${G} +push|git push"
  "${G} +(reset +.*--hard|reset +--hard|clean +-[a-zA-Z]*f|branch +.*-D|stash +(drop|clear)|checkout +.*-- +\.|restore +\.)|discarding local git changes"
  "${G} +rebase|git rebase"
  "${RMREC}|recursive delete"
  "${S}rm( +-[^ ]+)* +-[a-zA-Z]*f|forced delete"
  "${S}find .*(-delete|-exec +rm)|bulk delete with find"
  "${S}docker +(system +prune|volume +(rm|prune)|push)|destructive or publishing docker command"
  "${S}docker[ -]compose +down .*(-v|--volumes)|removing docker volumes"
  "${S}(kubectl +(delete|apply)|terraform +(apply|destroy)|helm +(install|upgrade|uninstall))|changing infrastructure"
  "${S}(DROP +(TABLE|DATABASE)|TRUNCATE +)|destroying database data"
  "${S}gh +(pr +merge|release +(create|delete)|workflow +run)|publishing through GitHub"
  "${S}go +get |adding or changing a dependency"
)

# Writing to the guard itself through the shell. The 2>&1 style is not a write.
if printf '%s' "$cmd" | grep -Eq '\.claude/(settings|hooks)' &&
  printf '%s' "$cmd" | grep -Eq '(^|[^0-9&])>|tee |sed +-i|perl +-[a-z]*i|(^|[ ;&|])(rm|mv|cp|chmod|ln) '; then
  echo "BLOCKED by guard-bash: changing the guard or settings. Ask the user to do it (see CLAUDE.md)." >&2
  exit 2
fi

for rule in "${block_rules[@]}"; do
  if printf '%s' "$cmd" | grep -Eq -- "${rule%|*}"; then
    echo "BLOCKED by guard-bash: ${rule##*|}. Do not work around this; ask the user to run it (see CLAUDE.md)." >&2
    exit 2
  fi
done

for rule in "${ask_rules[@]}"; do
  if printf '%s' "$cmd" | grep -Eq -- "${rule%|*}"; then
    jq -n --arg r "guard-bash: ${rule##*|} needs user confirmation" '{hookSpecificOutput: {
      hookEventName: "PreToolUse", permissionDecision: "ask", permissionDecisionReason: $r}}'
    exit 0
  fi
done

exit 0
