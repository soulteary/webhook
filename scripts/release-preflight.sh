#!/usr/bin/env bash

set -Eeuo pipefail

REPOSITORY="soulteary/webhook"
REPOSITORY_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RELEASE_VERSION="${1:-}"

fail() {
  echo "release preflight failed: $*" >&2
  exit 1
}

for command_name in git go goreleaser gh; do
  command -v "$command_name" >/dev/null 2>&1 || fail "missing required command: $command_name"
done
gh auth status >/dev/null 2>&1 || fail "GitHub CLI is not authenticated"

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

if gh release view "$RELEASE_VERSION" --repo "$REPOSITORY" >/dev/null 2>&1; then
  fail "GitHub Release $RELEASE_VERSION already exists"
fi

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

go mod tidy
git diff --exit-code -- go.mod go.sum
go test -race ./...
goreleaser check

documentation_output="$(mktemp -d)"
documentation_environment=""
cleanup() {
  if [[ -n "$documentation_environment" && -d "$documentation_environment" ]]; then
    rm -rf -- "$documentation_environment"
  fi
  if [[ -d "$documentation_output" ]]; then
    rm -rf -- "$documentation_output"
  fi
}
trap cleanup EXIT

if command -v mkdocs >/dev/null 2>&1; then
  mkdocs build --strict --site-dir "$documentation_output"
else
  command -v python3 >/dev/null 2>&1 || \
    fail "missing mkdocs and python3; install one of them to build documentation"
  documentation_environment="$(mktemp -d)"
  python3 -m venv "$documentation_environment"
  "$documentation_environment/bin/python" -m pip install \
    --disable-pip-version-check \
    -r requirements-docs.txt
  "$documentation_environment/bin/mkdocs" build \
    --strict \
    --site-dir "$documentation_output"
fi

[[ -z "$(git status --porcelain)" ]] || fail "preflight changed the working tree"

echo "release preflight passed for $RELEASE_VERSION at $(git rev-parse HEAD)"
echo "next: create one annotated tag and push only refs/tags/$RELEASE_VERSION"
