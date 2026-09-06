# Release guide

This is the maintainer runbook for publishing a stable WebHook release. A
release is created only by pushing one immutable semantic-version tag. Do not
create a GitHub Release manually and do not run the Release workflow from the
Actions page.

## 7.3.0 scope

The 7.3.0 release includes the following changes after 7.2.0:

- remove the duplicate `webhook-config-ui` release binary; use
  `webhook -config-ui` with the single published binary;
- add constrained Docker Secret template support through `getSecret` and
  `trimSpace`, without exposing arbitrary filesystem reads;
- update examples, container tags, integrity commands, and migration downloads
  to 7.3.0;
- make Release CI tag-only and split validation, publishing, attestation, and
  verification into retry-safe jobs.

## Release invariants

- The release tag is exactly `MAJOR.MINOR.PATCH`, without a `v` prefix.
- The tagged commit is already on `main`, and all required checks passed on it.
- A version is tagged once. Never move, delete, or reuse a published tag.
- The Release workflow creates the GitHub Release and both registries' images.
- `preflight` performs all build checks before credentials or publishing are
  used. `publish` runs once. `attest`, `verify`, or Documentation may be
  retried independently without running GoReleaser again.

## Prerequisites

Before preparing a tag:

1. Merge every release PR, including the version/documentation preparation PR.
2. Wait for Tests, container smoke tests, Security Scan, CodeQL, Documentation,
   and Benchmarks on the resulting `main` commit.
3. Confirm the Docker Hub credentials and GitHub repository/package permissions
   are available to Actions.
4. Install Git. Go, Python, MkDocs, GoReleaser, Syft, Cosign, and GitHub CLI are
   provided by GitHub Actions and are not local release dependencies.

For 7.3.0, PR #177 and PR #178 must both be present in the selected `main`
commit.

## Exact publishing sequence

From an up-to-date, clean `main` checkout, run one command:

```bash
./scripts/release.sh 7.3.0
```

The command runs a fast Git-only preflight, displays the exact commit, and asks
you to type `7.3.0` before it creates and pushes one annotated tag. It refuses a
dirty checkout, a non-`main` branch, a checkout that differs from `origin/main`,
an invalid version, stale version references, or an existing local/remote tag.

Do not create a draft or empty GitHub Release first; GoReleaser owns that
operation. The script pushes only `refs/tags/7.3.0`, never all local tags.

The tag push starts these paths:

| Order | Workflow/job | Side effects | Retry rule |
|---|---|---|---|
| 1 | Release / `preflight` | Validates the tag and `main` ancestry, runs race tests, builds strict documentation, and checks GoReleaser configuration | No publishing side effects; safe to retry |
| 2 | Release / `publish` | Builds the GitHub Release, archives, SBOMs, and signed container manifests | Run once only |
| 3 | Release / `attest` | Downloads published assets and creates GitHub provenance | Retry this job only |
| 4 | Release / `verify` | Verifies assets, Sigstore bundle, provenance, image UID, and image signatures | Retry this job only |
| Parallel | Documentation | Builds `7.3.0`, updates `latest`, and sets the documentation default | Retry independently |

Wait for both workflows; a visible GitHub Release is not completion by itself.

```bash
RELEASE_VERSION=7.3.0
RELEASE_RUN_ID="$(gh run list --workflow build.yml --branch "$RELEASE_VERSION" --limit 1 --json databaseId --jq '.[0].databaseId')"
gh run watch "$RELEASE_RUN_ID" --exit-status

DOCS_RUN_ID="$(gh run list --workflow docs.yml --branch "$RELEASE_VERSION" --limit 1 --json databaseId --jq '.[0].databaseId')"
gh run watch "$DOCS_RUN_ID" --exit-status
```

## Post-release verification

After both workflows pass:

```bash
gh release view "$RELEASE_VERSION" --repo soulteary/webhook \
  --json tagName,isDraft,isPrerelease,url

mkdir -p "release-$RELEASE_VERSION"
gh release download "$RELEASE_VERSION" --repo soulteary/webhook \
  --dir "release-$RELEASE_VERSION"
find "release-$RELEASE_VERSION" -maxdepth 1 -type f -printf '%f\n' | sort

test -z "$(find "release-$RELEASE_VERSION" -name '*webhook-config-ui*' -print -quit)"

for variant in core runner extended; do
  image="ghcr.io/soulteary/webhook:${variant}-${RELEASE_VERSION}"
  docker pull "$image"
  test "$(docker image inspect --format '{{ .Config.User }}' "$image")" = "65532:65532"
  docker run --rm "$image" -version | grep "$RELEASE_VERSION"
done
```

Also confirm that both
`https://soulteary.github.io/webhook/7.3.0/` and the `latest` documentation URL
resolve correctly. The Release workflow already verifies the checksum Sigstore
bundle, GitHub attestation, image signatures, and all three runtime variants.

## Failure handling

Do not immediately press **Re-run all jobs**.

| Failure point | Safe action |
|---|---|
| Local preflight or confirmation | Fix/update the checkout and rerun the command; no release exists yet |
| CI `preflight` | Fix `main`, prepare a new version, and push a new tag; publishing did not start |
| `publish`, before **Run GoReleaser** starts | Re-run the failed `publish` job |
| `publish`, after **Run GoReleaser** starts | Do not rerun; inspect the GitHub Release and both registries for partial immutable artifacts |
| `attest` | Re-run failed jobs; `publish` remains completed |
| `verify` | Re-run failed jobs or the `verify` job only |
| Documentation | Re-run only the Documentation workflow |

If GoReleaser started and any versioned release asset or image exists, do not
delete the tag and reuse the version. Preserve the evidence, fix the cause on
`main`, and publish the next patch version (for example, 7.3.1). This avoids
serving different bytes under the same immutable version.

If the local tag was created but the push failed, inspect it with
`git show 7.3.0`, then retry only `git push origin refs/tags/7.3.0`. Do not rerun
`release.sh`, because its duplicate-tag guard will stop.
