#!/usr/bin/env bash
set -euo pipefail

add_or_fix_front_matter() {
  local f="$1"
  # Only process regular files that end with .md
  [[ -f "$f" ]] || return 0
  [[ "$f" == *.md ]] || return 0

  local first_line
  first_line=$(head -n1 "$f" || true)

  if [[ "$first_line" == "---" ]]; then
    # Has front matter. Ensure render_with_liquid: false exists before closing '---'.
    if awk 'BEGIN{fm=0; found=0} NR==1 && /^---$/ {fm=1; next} fm && /^render_with_liquid:/ {found=1} fm && /^---$/ {fm=0} END{exit found?0:1}' "$f"; then
      # already present
      return 0
    else
      # insert before the closing --- of the first front matter block
      awk 'BEGIN{fm=0; done=0}
        NR==1 && /^---$/ {fm=1; print; next}
        fm && /^---$/ && !done {print "render_with_liquid: false"; print "---"; fm=0; done=1; next}
        {print}
      ' "$f" > "$f.tmp" && mv "$f.tmp" "$f"
    fi
  else
    # No front matter. Prepend minimal front matter disabling Liquid
    tmp=$(mktemp)
    {
      echo '---'
      echo 'render_with_liquid: false'
      echo '---'
      echo
      cat "$f"
    } > "$tmp"
    mv "$tmp" "$f"
  fi
}

fix_inline_code_examples() {
  local f="$1"
  [[ -f "$f" ]] || return 0
  [[ "$f" == *.md ]] || return 0

  # Escape {{ and }} in Go code blocks to prevent Liquid parsing
  # Replace {{ with {% raw %}{{{% endraw %} and }} with {% raw %}}}{% endraw %}
  # This preserves the content while preventing Jekyll Liquid from interpreting it
  perl -i -pe 's/\{\{/{% raw %}{{{% endraw %}/g; s/\}\}/{% raw %}}}{% endraw %}/g' "$f"
}

# If no args, operate on default paths
if [[ $# -eq 0 ]]; then
  set -- docs/api-reference/*.md docs/generated-examples.md
fi

for path in "$@"; do
  # Expand globs
  for f in $path; do
    add_or_fix_front_matter "$f"
    fix_inline_code_examples "$f"
  done
done
