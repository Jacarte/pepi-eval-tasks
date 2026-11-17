# shellcheck shell=bash
# Sourced by check-milestone.sh and prepare-workspace.sh: minimal reader for the flat
# "key: value" lines of task.yaml (the README schema; no multi-line values).

# task_value <task.yaml> <key>: prints the value of a top-level key ("" when absent).
task_value() {
  awk -v want="$2" '
    function clean(s) {
      sub(/[[:space:]]+#.*$/, "", s); sub(/^[[:space:]]+/, "", s); sub(/[[:space:]]+$/, "", s)
      if (s ~ /^".*"$/ || s ~ /^\x27.*\x27$/) s = substr(s, 2, length(s) - 2)
      return s
    }
    index($0, want ":") == 1 { v = $0; sub(/^[^:]*:/, "", v); print clean(v); exit }
  ' "$1"
}

# task_owned_dirs <task.yaml>: prints each owned_dirs entry on its own line.
task_owned_dirs() {
  local raw
  raw=$(task_value "$1" owned_dirs)
  raw=${raw#\[}; raw=${raw%\]}
  raw=${raw//[[:space:]\"\']/}
  if [ -n "$raw" ] && [ "$raw" != null ]; then
    printf '%s\n' "${raw//,/$'\n'}"
  fi
}

# task_restore_seed_tests <task.yaml>: prints "true" or "false"; returns 1 on any other value.
task_restore_seed_tests() {
  local v
  v=$(task_value "$1" restore_seed_tests)
  case "$v" in
    true) echo true ;;
    ""|false) echo false ;;
    *) return 1 ;;
  esac
}

# workspace_special_files <dir>: prints, relative to <dir> and at most 20, every entry that is
# neither a regular file nor a directory: symlinks, fifos, sockets and devices. A legitimate
# solution never needs them, and a symlink would let a file slip past every find that selects
# regular files (the seed-test restore, the cheat scan) while go still compiles it.
workspace_special_files() {
  (cd "$1" && find . ! -type d ! -type f -print | sort | head -n 20)
}
