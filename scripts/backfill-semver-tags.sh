#!/usr/bin/env bash
# Backfill Go-modules mirror tags (v1.YYMMDD.N) for every historical CalVer
# tag (vYY.M.D, YY >= 20). Run once after release-go-mirror.yml lands; from
# then on the workflow handles new tags automatically.
#
# Usage:
#   ./scripts/backfill-semver-tags.sh           # local — creates tags only
#   ./scripts/backfill-semver-tags.sh --push    # also push them to origin
#
# Idempotent: skips any v1.YYMMDD.N that already exists.

set -euo pipefail

PUSH=false
if [ "${1:-}" = "--push" ]; then
  PUSH=true
fi

git fetch --tags --quiet

mapped=0
skipped=0

while IFS= read -r CAL; do
  [ -z "$CAL" ] && continue
  case "$CAL" in
    v1.*) skipped=$((skipped+1)); continue ;;
  esac

  IFS=. read -r Y M D <<<"${CAL#v}"
  if ! [[ "$Y" =~ ^[0-9]+$ && "$M" =~ ^[0-9]+$ && "$D" =~ ^[0-9]+$ ]]; then
    skipped=$((skipped+1)); continue
  fi
  if [ "$Y" -lt 20 ] || [ "$M" -lt 1 ] || [ "$M" -gt 12 ] || [ "$D" -lt 1 ] || [ "$D" -gt 31 ]; then
    skipped=$((skipped+1)); continue
  fi

  MMDD=$(printf '%02d%02d' "$M" "$D")
  BASE="v1.${Y}${MMDD}"
  PATCH=0
  while git rev-parse -q --verify "refs/tags/${BASE}.${PATCH}" >/dev/null; do
    PATCH=$((PATCH+1))
  done
  SEMVER="${BASE}.${PATCH}"

  COMMIT=$(git rev-list -n1 "$CAL")
  git tag -a "$SEMVER" "$COMMIT" -m "Go-modules mirror of $CAL"
  echo "mapped $CAL → $SEMVER ($COMMIT)"
  mapped=$((mapped+1))
done < <(git tag --list 'v*.*.*' | sort -V)

echo
echo "summary: mapped=$mapped skipped=$skipped"

if $PUSH && [ "$mapped" -gt 0 ]; then
  echo "pushing v1.* tags to origin..."
  git push origin --tags
fi
