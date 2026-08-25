---
name: translate-operator-docs
description: Translate DataKit Operator Markdown documentation from export/zh into export/en, export/ja, and export/ko. Use when Chinese source documents are added, changed, renamed, or removed, or when translations need review or synchronization.
---

# Translate Operator Docs

Treat `export/zh` as the source of truth. Keep `export/en`, `export/ja`, and `export/ko` synchronized without using an external translation CLI or API key.

## Workflow

1. Inspect the Chinese changes and the related implementation or configuration before translating technical claims.
2. Create, rename, update, or remove the matching document in all three target directories.
3. Translate prose naturally for each language. Do not mechanically transliterate product or Kubernetes terminology.
4. Preserve filenames and the order of headings, lists, tables, admonitions, and examples.
5. Keep these values byte-for-byte unchanged:
   - executable/configuration code and inline code;
   - link and image destinations;
   - `{{.TemplateVariables}}`;
   - `<<<custom_key...>>>` brand variables;
   - Markdown anchors and attributes such as `{#install}` and `{:target="_blank"}`.
   In Mermaid blocks, preserve syntax and identifiers, but translate human-visible labels and messages.
6. Review the complete diff for factual accuracy, omissions, duplicated prose, and accidental source-language text.
7. Compute SHA-256 for every translated Chinese source and update each target language's `.translation/metadata.json` only after its translation is complete.
8. Run `make docs_lint`. Fix every failure before finishing.

Do not stage or commit changes unless the user explicitly asks.
