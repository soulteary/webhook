#!/usr/bin/env bash

set -Eeuo pipefail

REPOSITORY_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RELEASE_VERSION="${1:-}"

fail() {
  echo "release failed: $*" >&2
  exit 1
}

[[ "$RELEASE_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || \
  fail "usage: $0 MAJOR.MINOR.PATCH (without a v prefix)"

cd "$REPOSITORY_ROOT"
"$REPOSITORY_ROOT/scripts/release-preflight.sh" "$RELEASE_VERSION"

RELEASE_COMMIT="$(git rev-parse HEAD)"
echo
echo "Ready to release $RELEASE_VERSION from $RELEASE_COMMIT"
echo "Release CI will run preflight -> publish -> attest -> verify -> documentation"
printf "Type %s to create and push the release tag: " "$RELEASE_VERSION"
read -r confirmation
[[ "$confirmation" == "$RELEASE_VERSION" ]] || fail "confirmation did not match; nothing was published"

git tag -a "$RELEASE_VERSION" "$RELEASE_COMMIT" -m "Release $RELEASE_VERSION"
if ! git push origin "refs/tags/$RELEASE_VERSION"; then
  echo "the annotated tag exists locally but was not published" >&2
  echo "inspect it with: git show $RELEASE_VERSION" >&2
  echo "after resolving the connection, retry only:" >&2
  echo "git push origin refs/tags/$RELEASE_VERSION" >&2
  exit 1
fi

echo
echo "Tag $RELEASE_VERSION was pushed successfully."
echo "Workflow: https://github.com/soulteary/webhook/actions/workflows/build.yml"
