#!/usr/bin/env bash

set -euo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly EXPORT_DIR="${SCRIPT_DIR}/export"
readonly CONFIG_FILE="${EXPORT_DIR}/config.env"

doc_repo="${HOME}/git/dataflux-doc"

usage() {
    cat <<'EOF'
Export DataKit Operator documents to dataflux-doc.

Usage:
  ./export.sh [-D dataflux-doc-dir]

Options:
  -D DIR  dataflux-doc repository (default: ~/git/dataflux-doc)
  -h      show this help
EOF
}

fail() {
    echo "error: $*" >&2
    exit 1
}

while getopts ":D:h" opt; do
    case "${opt}" in
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
[[ -d "${EXPORT_DIR}/zh" && -d "${EXPORT_DIR}/en" ]] || fail "missing zh/en document sources"
[[ -d "${doc_repo}" ]] || fail "dataflux-doc directory does not exist: ${doc_repo}"

doc_repo="$(cd -- "${doc_repo}" && pwd)"
[[ -f "${doc_repo}/mkdocs.zh.yml" && -f "${doc_repo}/mkdocs.en.yml" ]] || \
    fail "target is not a dataflux-doc repository: ${doc_repo}"

list_documents() {
    find "$1" -maxdepth 1 -type f -name '*.md' -printf '%f\n' | LC_ALL=C sort
}

list_placeholders() {
    { grep -rhoE --include='*.md' '\{\{\.[A-Za-z][A-Za-z0-9]*\}\}' "$1" || true; } | LC_ALL=C sort -u
}

list_brand_tokens() {
    { grep -oE '<<<[^>]*>>>' "$1" || true; } | LC_ALL=C sort
}

if ! diff -u <(list_documents "${EXPORT_DIR}/zh") <(list_documents "${EXPORT_DIR}/en"); then
    fail "zh/en document filenames differ"
fi

if ! diff -u <(list_placeholders "${EXPORT_DIR}/zh") <(list_placeholders "${EXPORT_DIR}/en"); then
    fail "zh/en template placeholders differ"
fi

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
    if ! grep -RFq --include='*.md' -- "${placeholder}" "${EXPORT_DIR}/zh" "${EXPORT_DIR}/en"; then
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
mkdir -p "${stage_dir}/zh" "${stage_dir}/en"

for lang in zh en; do
    while IFS= read -r filename; do
        source_file="${EXPORT_DIR}/${lang}/${filename}"
        rendered_file="${stage_dir}/${lang}/${filename}"
        sed "${sed_expressions[@]}" "${source_file}" >"${rendered_file}"

        # Full image values can add brand variables, but existing variables must remain.
        missing_brand_tokens="$(comm -23 \
            <(list_brand_tokens "${source_file}") \
            <(list_brand_tokens "${rendered_file}"))"
        [[ -z "${missing_brand_tokens}" ]] || fail "brand variables changed in ${lang}/${filename}"
    done < <(list_documents "${EXPORT_DIR}/${lang}")
done

unresolved="$({ grep -RnoE '\{\{\.[A-Za-z][A-Za-z0-9]*\}\}' "${stage_dir}" || true; })"
[[ -z "${unresolved}" ]] || fail "unresolved template placeholders:\n${unresolved}"

document_count=0
for lang in zh en; do
    target_dir="${doc_repo}/docs/${lang}/datakit"
    [[ -d "${doc_repo}/docs/${lang}" ]] || fail "missing dataflux-doc language directory: docs/${lang}"
    mkdir -p "${target_dir}"

    while IFS= read -r filename; do
        install -m 0644 "${stage_dir}/${lang}/${filename}" "${target_dir}/${filename}"
        document_count=$((document_count + 1))
    done < <(list_documents "${stage_dir}/${lang}")
done

echo "Exported ${document_count} documents to ${doc_repo}/docs/{zh,en}/datakit"
