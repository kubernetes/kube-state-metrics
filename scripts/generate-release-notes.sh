#!/usr/bin/env bash
set -euo pipefail

# Minimal release notes generator (gh-only)
# Usage:
#   ./scripts/generate-release-notes-min.sh --tag v2.17.0 --out release/test_release.md
#
# Requires: gh (GitHub CLI) authenticated in the environment.

OUT="release/release_notes.md"
TAG=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tag) TAG="$2"; shift 2 ;;
    --out) OUT="$2"; shift 2 ;;
    *) echo "Unknown arg: $1" >&2; exit 2 ;;
  esac
done

if [[ -z "$TAG" ]]; then
  echo "ERROR: --tag is required" >&2
  exit 2
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "ERROR: gh (GitHub CLI) is required and must be authenticated." >&2
  exit 2
fi

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$REPO_ROOT" ]]; then
  echo "ERROR: must be inside a git repository" >&2
  exit 2
fi
CHANGELOG="${REPO_ROOT}/CHANGELOG.md"
if [[ ! -f "$CHANGELOG" ]]; then
  echo "ERROR: CHANGELOG.md not found at $CHANGELOG" >&2
  exit 2
fi

if [[ "$OUT" = /* ]]; then
  OUT_PATH="$OUT"
else
  OUT_PATH="${REPO_ROOT%/}/${OUT#./}"
fi
mkdir -p "$(dirname "$OUT_PATH")"

CLEAN_TAG="${TAG#v}" 
VERSION_LINE="$(grep -nE "^##[[:space:]]+v?${CLEAN_TAG//./\\.}[[:space:]]*/" "$CHANGELOG" 2>/dev/null | head -n1 | cut -d: -f1 || true)"

if [[ -z "$VERSION_LINE" ]]; then
  echo "ERROR: Version $TAG not found in $CHANGELOG" >&2
  exit 2
fi

SELECT_LINE="$VERSION_LINE"

TOTAL_LINES="$(wc -l < "$CHANGELOG")"
NEXT_HEADING_LINE="$((TOTAL_LINES + 1))"
for ln in $(seq $((SELECT_LINE + 1)) "$TOTAL_LINES"); do
  if sed -n "${ln}p" "$CHANGELOG" | grep -q '^##'; then
    NEXT_HEADING_LINE="$ln"
    break
  fi
done

USER_SECTION="$(sed -n "${SELECT_LINE},$((NEXT_HEADING_LINE - 1))p" "$CHANGELOG" || true)"
USER_FACING="$(printf "%s\n" "$USER_SECTION" | grep -E '^\s*\*\s*\[' | sed '/^\s*$/d')"
if [[ -z "$USER_FACING" ]]; then
  USER_FACING="    No user-facing changes found for this version."
fi

readarray -t ALL_TAGS < <(git -C "$REPO_ROOT" for-each-ref --sort=-creatordate --format '%(refname:short)' refs/tags || true)
PREV_TAG=""
found=0
for t in "${ALL_TAGS[@]}"; do
  if [[ "$found" -eq 1 ]]; then
    PREV_TAG="$t"
    break
  fi
  if [[ "$t" == "$TAG" ]]; then
    found=1
  fi
done
if [[ -z "$PREV_TAG" ]]; then
  for t in "${ALL_TAGS[@]}"; do
    if [[ "$t" != "$TAG" ]]; then
      PREV_TAG="$t"
      break
    fi
  done
fi

if [[ -n "$PREV_TAG" ]]; then
  RANGE="${PREV_TAG}..${TAG}"
  PREV_TAG_TEXT="${PREV_TAG}"
else
  PREV_REF="$(git -C "$REPO_ROOT" rev-list --max-parents=0 HEAD)"
  RANGE="${PREV_REF}..${TAG}"
  PREV_TAG_TEXT="(initial commit)"
fi

if [[ -n "$PREV_TAG" ]]; then
  PREV_TAG_DATE="$(git -C "$REPO_ROOT" show -s --format=%cI "$PREV_TAG" 2>/dev/null || true)"
else
  PREV_TAG_DATE="1970-01-01T00:00:00Z"
fi
TAG_DATE="$(git -C "$REPO_ROOT" show -s --format=%cI "$TAG" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")"
PREV_DATE_SHORT="$(date -d "$PREV_TAG_DATE" +%Y-%m-%d 2>/dev/null || echo "1970-01-01")"
TAG_DATE_SHORT="$(date -d "$TAG_DATE" +%Y-%m-%d 2>/dev/null || date -u +%Y-%m-%d)"

REPO_FULL="$(gh repo view --json nameWithOwner --template '{{.nameWithOwner}}' 2>/dev/null || true)"
if [[ -z "$REPO_FULL" ]]; then
  origin="$(git -C "$REPO_ROOT" remote get-url origin 2>/dev/null || true)"
  if [[ "$origin" =~ github.com[:/]+([^/]+)/([^/.]+) ]]; then
    REPO_FULL="${BASH_REMATCH[1]}/${BASH_REMATCH[2]}"
  fi
fi
COMPARE_URL=""
if [[ -n "$REPO_FULL" && "$PREV_TAG_TEXT" != "(initial commit)" ]]; then
  COMPARE_URL="https://github.com/${REPO_FULL}/compare/${PREV_TAG_TEXT}...${TAG}"
fi

PR_LINES="$(gh pr list --state merged --search "merged:${PREV_DATE_SHORT}..${TAG_DATE_SHORT}" --json number,title,author --limit 1000 --template '{{range .}}{{.number}}|{{.title}}|{{.author.login}}{{"\n"}}{{end}}')" || true

FULL_CHANGELOG_LINES=()
AUTHORS_IN_ORDER=()
if [[ -n "$PR_LINES" ]]; then
  while IFS= read -r line; do
    prnum="${line%%|*}"
    rest="${line#*|}"
    title="${rest%%|*}"
    login="${rest##*|}"
    FULL_CHANGELOG_LINES+=("- ${title} by @${login} in #${prnum}")
    if [[ -n "$login" && "$login" != "null" && "${AUTHORS_IN_ORDER[*]}" != *"$login"* ]]; then
      AUTHORS_IN_ORDER+=("$login")
    fi
  done <<< "$PR_LINES"
fi

NEW_CONTRIBUTORS=()
declare -A SEEN_AUTHORS
declare -A CONTRIBUTOR_PRS
if command -v gh >/dev/null 2>&1; then
  while IFS= read -r line; do
    prnum="$(echo "$line" | grep -oE '#[0-9]+' | tr -d '#')"
    if [[ -n "$prnum" ]]; then
      author="$(gh pr view "$prnum" --json author --jq '.author.login' 2>/dev/null || true)"
      if [[ -n "$author" && "$author" != "null" ]]; then
        older="$(gh pr list --state merged --author "$author" --search "merged:<${PREV_DATE_SHORT}" --json number --limit 1 --jq '.[0].number' 2>/dev/null || true)"
        if [[ -z "$older" || "$older" == "null" ]]; then
          if [[ -z "${CONTRIBUTOR_PRS[$author]:-}" ]]; then
            CONTRIBUTOR_PRS[$author]="$prnum"
          else
            if (( prnum < ${CONTRIBUTOR_PRS[$author]} )); then
              CONTRIBUTOR_PRS[$author]="$prnum"
            fi
          fi
        fi
      fi
    fi
  done < <(git log --pretty=format:'%s' ${RANGE} --merges | grep -E 'Merge pull request')
  
  for author in "${!CONTRIBUTOR_PRS[@]}"; do
    NEW_CONTRIBUTORS+=("$author#${CONTRIBUTOR_PRS[$author]}")
  done
fi

if [[ ${#NEW_CONTRIBUTORS[@]} -gt 0 ]]; then
  NEW_UNIQ=("${NEW_CONTRIBUTORS[@]}")
else
  NEW_UNIQ=()
fi

{
  echo "Changelog"
  echo
  if [[ -n "$USER_FACING" ]]; then
    printf "%s\n" "$USER_FACING" | sed 's/^/    /'
  else
    echo "    (no user-facing entries found in the selected CHANGELOG section)"
  fi
  echo
  echo "Full Changelog"
  echo
  if [[ ${#FULL_CHANGELOG_LINES[@]} -gt 0 ]]; then
    for l in "${FULL_CHANGELOG_LINES[@]}"; do
      printf "    %s\n" "$l"
    done
  else
    echo "    (no merged PRs found between ${PREV_TAG_TEXT} and ${TAG})"
  fi
  echo
  echo "New Contributors"
  echo
  if [[ ${#NEW_CONTRIBUTORS[@]} -eq 0 ]]; then
    echo "    No new contributors in this release."
  else
    for entry in "${NEW_UNIQ[@]}"; do
      user="${entry%%#*}"
      prnum="${entry##*#}"
      echo "    - @${user} made their first contribution in #${prnum}"
    done
  fi
  echo
  if [[ -n "$COMPARE_URL" ]]; then
    echo "Full Changelog: ${COMPARE_URL}"
  else
    echo "Full Changelog: ${PREV_TAG_TEXT}...${TAG}"
  fi
} > "$OUT_PATH"

echo "WROTE: $OUT_PATH"
echo "  user-facing from CHANGELOG.md at line ${SELECT_LINE}"
echo "  full changelog range: ${RANGE}"
if [[ -n "$COMPARE_URL" ]]; then
  echo "  compare link: ${COMPARE_URL}"
fi
