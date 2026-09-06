#!/usr/bin/env bash

set -Eeuo pipefail

REPOSITORY_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RELEASE_VERSION="${1:-}"

fail() {
  echo "release preflight failed: $*" >&2
  exit 1
}

command -v git >/dev/null 2>&1 || fail "missing required command: git"

[[ "$RELEASE_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || \
  fail "usage: $0 MAJOR.MINOR.PATCH (without a v prefix)"

cd "$REPOSITORY_ROOT"

[[ "$(git branch --show-current)" == "main" ]] || fail "run from the main branch"
[[ -z "$(git status --porcelain)" ]] || fail "the working tree is not clean"

git fetch --prune origin main --tags
[[ "$(git rev-parse HEAD)" == "$(git rev-parse origin/main)" ]] || \
  fail "local main does not exactly match origin/main"

if git show-ref --verify --quiet "refs/tags/$RELEASE_VERSION"; then
  fail "local tag $RELEASE_VERSION already exists"
fi

set +e
git ls-remote --exit-code --tags origin "refs/tags/$RELEASE_VERSION" >/dev/null 2>&1
remote_tag_status=$?
set -e
case "$remote_tag_status" in
  0) fail "remote tag $RELEASE_VERSION already exists" ;;
  2) ;;
  *) fail "could not confirm that remote tag $RELEASE_VERSION is unused" ;;
esac

for versioned_file in \
  README.md \
  docs/en-US/Container-Images.md \
  docs/zh-CN/Container-Images.md \
  docs/en-US/Migration-Guide.md \
  docs/zh-CN/Migration-Guide.md \
  example/quickstart/compose.yaml; do
  grep -F "$RELEASE_VERSION" "$versioned_file" >/dev/null || \
    fail "$versioned_file does not reference $RELEASE_VERSION"
done

echo "release preflight passed for $RELEASE_VERSION at $(git rev-parse HEAD)"
echo "build, test, documentation, and GoReleaser checks will run in Release CI before publishing"
