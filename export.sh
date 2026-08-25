#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly EXPORT_DIR="${SCRIPT_DIR}/export"
readonly CONFIG_FILE="${EXPORT_DIR}/config.env"
readonly -a LANGUAGES=(zh en ja ko)
readonly -a TRANSLATED_LANGUAGES=(en ja ko)

doc_repo="${HOME}/git/dataflux-doc"
check_only=false

usage() {
    cat <<'EOF'
Export DataKit Operator documents to dataflux-doc.

Usage:
  ./export.sh [-c] [-D dataflux-doc-dir]

Options:
  -c      check source and rendered documents without exporting
  -D DIR  dataflux-doc repository (default: ~/git/dataflux-doc)
  -h      show this help
EOF
}

fail() {
    echo "error: $*" >&2
    exit 1
}

while getopts ":cD:h" opt; do
    case "${opt}" in
    c)
        check_only=true
        ;;
    D)
        doc_repo="${OPTARG}"
        ;;
    h)
        usage
        exit 0
        ;;
    :)
        fail "option -${OPTARG} requires an argument"
        ;;
    \?)
        fail "unknown option: -${OPTARG}"
        ;;
    esac
done
shift $((OPTIND - 1))

[[ $# -eq 0 ]] || fail "unexpected argument: $1"
[[ -f "${CONFIG_FILE}" ]] || fail "missing config: ${CONFIG_FILE}"

declare -a language_dirs=()
for lang in "${LANGUAGES[@]}"; do
    [[ -d "${EXPORT_DIR}/${lang}" ]] || fail "missing ${lang} document sources"
    language_dirs+=("${EXPORT_DIR}/${lang}")
done

if [[ "${check_only}" == false ]]; then
    [[ -d "${doc_repo}" ]] || fail "dataflux-doc directory does not exist: ${doc_repo}"
    doc_repo="$(cd -- "${doc_repo}" && pwd)"
    [[ -f "${doc_repo}/mkdocs.zh.yml" && -f "${doc_repo}/mkdocs.en.yml" ]] || \
        fail "target is not a dataflux-doc repository: ${doc_repo}"
fi

list_documents() {
    find "$1" -maxdepth 1 -type f -name '*.md' -printf '%f\n' | LC_ALL=C sort
}

list_placeholders() {
    { grep -rhoE --include='*.md' '\{\{\.[A-Za-z][A-Za-z0-9]*\}\}' "$1" || true; } | LC_ALL=C sort -u
}

require_command() {
    command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

run_mdcheck() {
    local markdown_dir="$1"
    local check_section="$2"

    (
        cd -- "${SCRIPT_DIR}"
        GOFLAGS=-mod=vendor go run ./cmd/doccheck \
            -dir "${markdown_dir}" \
            -check-section="${check_section}"
    )
}

run_translation_check() {
    (
        cd -- "${SCRIPT_DIR}"
        GOFLAGS=-mod=vendor go run ./cmd/doctranslationcheck
    )
}

check_brand_names() {
    local markdown_dir="$1"
    local keyword
    local matches
    local found=false
    local keywords=("观测云" "Guance Cloud" "Guance" "guance.com")

    for keyword in "${keywords[@]}"; do
        matches="$({ grep -RFn --include='*.md' -- "${keyword}" "${markdown_dir}" || true; })"
        if [[ -n "${matches}" ]]; then
            printf '%s\n' "${matches}" >&2
            found=true
        fi
    done

    [[ "${found}" == false ]] || fail "hard-coded brand names found in rendered documents"
}

check_rendered_documents() {
    local markdown_dir="$1"
    local -a markdown_files=()
    local path

    run_mdcheck "${markdown_dir}/zh" false
    run_mdcheck "${markdown_dir}/en" false
    check_brand_names "${markdown_dir}"

    while IFS= read -r -d '' path; do
        markdown_files+=("${path#"${markdown_dir}/en/"}")
    done < <(find "${markdown_dir}/en" -type f -name '*.md' -print0 | LC_ALL=C sort -z)
    [[ ${#markdown_files[@]} -gt 0 ]] || fail "no rendered Markdown documents found"

    (
        cd -- "${markdown_dir}/en"
        cspell lint --show-suggestions \
            -c "${SCRIPT_DIR}/scripts/cspell.json" \
            --no-progress "${markdown_files[@]}"
    )
    markdownlint -c "${SCRIPT_DIR}/scripts/markdownlint.yml" "${markdown_dir}"
}

if [[ "${check_only}" == true ]]; then
    require_command go
    require_command cspell
    require_command markdownlint
    cspell_version="$(cspell --version)" || fail "unable to run cspell"
    markdownlint_version="$(markdownlint --version)" || fail "unable to run markdownlint"
    echo "cspell ${cspell_version}"
    echo "markdownlint ${markdownlint_version}"
    run_mdcheck "${EXPORT_DIR}/zh" true
    run_mdcheck "${EXPORT_DIR}/en" true
    run_translation_check
fi

for lang in "${TRANSLATED_LANGUAGES[@]}"; do
    if ! diff -u <(list_documents "${EXPORT_DIR}/zh") <(list_documents "${EXPORT_DIR}/${lang}"); then
        fail "zh/${lang} document filenames differ"
    fi

    if ! diff -u <(list_placeholders "${EXPORT_DIR}/zh") <(list_placeholders "${EXPORT_DIR}/${lang}"); then
        fail "zh/${lang} template placeholders differ"
    fi
done

declare -A config_values=()
declare -a config_keys=()
line_number=0

while IFS= read -r line || [[ -n "${line}" ]]; do
    line_number=$((line_number + 1))
    [[ -z "${line}" || "${line}" == \#* ]] && continue

    if [[ ! "${line}" =~ ^([A-Za-z][A-Za-z0-9]*)=(.+)$ ]]; then
        fail "invalid config at ${CONFIG_FILE}:${line_number}"
    fi

    key="${BASH_REMATCH[1]}"
    value="${BASH_REMATCH[2]}"
    [[ -z "${config_values[${key}]+configured}" ]] || fail "duplicate config key: ${key}"
    [[ "${value}" != *$'\r'* ]] || fail "invalid carriage return in config key: ${key}"

    config_keys+=("${key}")
    config_values["${key}"]="${value}"
done <"${CONFIG_FILE}"

[[ ${#config_keys[@]} -gt 0 ]] || fail "no template values configured"

declare -a sed_expressions=()
for key in "${config_keys[@]}"; do
    placeholder="{{.${key}}}"
    if ! grep -RFq --include='*.md' -- "${placeholder}" "${language_dirs[@]}"; then
        fail "unused config key: ${key}"
    fi

    value="${config_values[${key}]}"
    escaped_value="${value//\\/\\\\}"
    escaped_value="${escaped_value//&/\\&}"
    escaped_value="${escaped_value//|/\\|}"
    sed_expressions+=("-e" "s|{{\\.${key}}}|${escaped_value}|g")
done

stage_dir="$(mktemp -d)"
trap 'rm -rf -- "${stage_dir}"' EXIT
for lang in "${LANGUAGES[@]}"; do
    mkdir -p "${stage_dir}/${lang}"
done

for lang in "${LANGUAGES[@]}"; do
    while IFS= read -r filename; do
        source_file="${EXPORT_DIR}/${lang}/${filename}"
        rendered_file="${stage_dir}/${lang}/${filename}"
        sed "${sed_expressions[@]}" "${source_file}" >"${rendered_file}"
    done < <(list_documents "${EXPORT_DIR}/${lang}")
done

unresolved="$({ grep -RnoE '\{\{\.[A-Za-z][A-Za-z0-9]*\}\}' "${stage_dir}" || true; })"
if [[ -n "${unresolved}" ]]; then
    printf 'error: unresolved template placeholders:\n%s\n' "${unresolved}" >&2
    exit 1
fi

if [[ "${check_only}" == true ]]; then
    check_rendered_documents "${stage_dir}"
    document_count="$(find "${stage_dir}" -type f -name '*.md' | wc -l)"
    echo "Checked ${document_count} source and rendered documents"
    exit 0
fi

document_count=0
for lang in "${LANGUAGES[@]}"; do
    target_dir="${doc_repo}/docs/${lang}/datakit"
    [[ -d "${doc_repo}/docs/${lang}" ]] || fail "missing dataflux-doc language directory: docs/${lang}"
    mkdir -p "${target_dir}"

    while IFS= read -r filename; do
        install -m 0644 "${stage_dir}/${lang}/${filename}" "${target_dir}/${filename}"
        document_count=$((document_count + 1))
    done < <(list_documents "${stage_dir}/${lang}")
done

echo "Exported ${document_count} documents to ${doc_repo}/docs/{zh,en,ja,ko}/datakit"
