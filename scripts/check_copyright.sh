#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"

missing_files=()

expected1="// Unless explicitly stated otherwise all files in this repository are licensed"
expected2="// under the MIT License."
expected3="// This product includes software developed at Guance Cloud (https://www.guance.com/)."
expected4="// Copyright 2021-present Guance, Inc."

has_header_block() {
  local file="$1"
  mapfile -t lines < "$file"
  local n=${#lines[@]}
  local i=0
  while (( i + 3 < n )); do
    if [[ "${lines[$i]}" == "$expected1" && \
          "${lines[$((i+1))]}" == "$expected2" && \
          "${lines[$((i+2))]}" == "$expected3" && \
          "${lines[$((i+3))]}" == "$expected4" ]]; then
      return 0
    fi
    ((i++))
  done
  return 1
}

# Find go files excluding vendor
while IFS= read -r -d '' file; do
  if ! has_header_block "$file"; then
    missing_files+=("$file")
  fi
done < <(find "$repo_root" -type f -name "*.go" -not -name "git.go" -not -path "*/vendor/*" -print0)

if (( ${#missing_files[@]} > 0 )); then
  echo "ERROR: The following files are missing the exact 4-line header block anywhere in file:" >&2
  for f in "${missing_files[@]}"; do
    echo "  $f" >&2
  done
  exit 1
fi

echo "Copyright header check passed."
