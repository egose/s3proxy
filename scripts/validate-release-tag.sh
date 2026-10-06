#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=scripts/semver.sh
source "$(dirname "${BASH_SOURCE[0]}")/semver.sh"

tag=${1:-}

if [[ $tag != v* ]] || ! semver_valid "${tag#v}"; then
  printf 'release tag must be an anchored SemVer with a v prefix: %s\n' "$tag" >&2
  exit 1
fi

expected=$(git rev-parse "$GITHUB_SHA^{commit}")
actual=$(git rev-list -n 1 "$tag")
head=$(git rev-parse HEAD)
if [ "$actual" != "$expected" ] || [ "$head" != "$expected" ]; then
  printf 'tag, event source commit, and checkout must match exactly\n' >&2
  exit 1
fi
if [ "$(tr -d '\r\n' < VERSION)" != "${tag#v}" ]; then
  printf 'VERSION does not match release tag %s\n' "$tag" >&2
  exit 1
fi
