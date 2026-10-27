#!/bin/sh
# page-history.sh — when, and by whom, every content page was created, from git.
#
# Hugo's GitInfo knows only a page's LAST commit. Its first one — the creation,
# followed across renames — is written here as JSON, for the theme's page
# history line (params.page.history). Run it from the site's root, before hugo:
#
#     sh themes/hextra/scripts/page-history.sh > data/pagehistory.json
#
# Keys are paths relative to the site root (content/en/docs/page.md); values
# hold the creation date and the author's NAME only — never the e-mail. It needs
# the full history: a shallow clone (CI default) gives every page the date of
# the oldest commit fetched.
set -eu
dir="${1:-content}"
if [ "$(git rev-parse --is-shallow-repository 2>/dev/null)" = "true" ]; then
  echo "page-history.sh: shallow clone, creation dates would be wrong (set GIT_DEPTH: 0)" >&2
  exit 1
fi
esc() { printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'; }
printf '{'
sep=''
git ls-files -z -- "$dir" | tr '\0' '\n' | grep -E '\.(md|markdown|html|adoc|org)$' | while IFS= read -r f; do
  line=$(git log --follow --diff-filter=A --format='%aI%x09%an' -- "$f" | tail -n 1)
  [ -n "$line" ] || continue
  date=$(printf '%s' "$line" | cut -f1)
  name=$(printf '%s' "$line" | cut -f2-)
  printf '%s\n  "%s": {"created": "%s", "author": "%s"}' "$sep" "$(esc "$f")" "$date" "$(esc "$name")"
  sep=','
done
printf '\n}\n'
