#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly REPO_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
readonly TRANSLATOR="${MKDOCS_TRANSLATOR_BIN:-mkdocs-translator}"
readonly WORKERS="${DOC_TRANSLATION_WORKERS:-4}"

command -v "${TRANSLATOR}" >/dev/null 2>&1 || {
    echo "error: mkdocs-translator is not installed: ${TRANSLATOR}" >&2
    exit 1
}

declare -a model_args=()
if [[ -n "${DOC_TRANSLATION_MODEL:-}" ]]; then
    model_args+=(--model "${DOC_TRANSLATION_MODEL}")
fi

cd -- "${REPO_DIR}"
"${TRANSLATOR}" \
    --source export/zh \
    --target export \
    --target-languages en,ja,ko \
    --workers "${WORKERS}" \
    --check-structure \
    --check-chinese \
    --delete-removed-translations \
    "${model_args[@]}"

bash export.sh -c
