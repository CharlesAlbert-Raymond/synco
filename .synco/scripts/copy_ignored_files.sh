#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(git -C "$script_dir/../.." rev-parse --show-toplevel)"
file_list="${SYNCO_IGNORED_FILES_LIST:-$repo_root/.synco/ignored-files.txt}"

if [[ -z "${SYNCO_WORKTREE_PATH:-}" ]]; then
  echo "SYNCO_WORKTREE_PATH is required" >&2
  exit 1
fi

if [[ ! -f "$file_list" ]]; then
  echo "ignored file list not found: $file_list" >&2
  exit 1
fi

while IFS= read -r rel_path || [[ -n "$rel_path" ]]; do
  [[ -z "$rel_path" || "$rel_path" == \#* ]] && continue

  source_path="$repo_root/$rel_path"
  target_path="$SYNCO_WORKTREE_PATH/$rel_path"

  if [[ ! -e "$source_path" ]]; then
    echo "ignored file listed but not found: $rel_path" >&2
    exit 1
  fi

  mkdir -p "$(dirname "$target_path")"
  cp -R "$source_path" "$target_path"
  echo "copied ignored file: $rel_path"
done < "$file_list"
