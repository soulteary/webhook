# Manual Docker Builds

This directory contains Dockerfiles for **manual or local** builds, as opposed to the release images produced by GoReleaser from `docker/goreleaser/`.

- **Dockerfile** – standard manual build
- **Dockerfile.alpine** – Alpine-based manual build

Use these when you want to build and run the image locally without going through the GoReleaser pipeline.

Pass build metadata explicitly when the image will be published or shared:

```bash
docker build \
  -f docker/manual/Dockerfile \
  --build-arg VERSION="$(git describe --tags --always)" \
  --build-arg COMMIT="$(git rev-parse HEAD)" \
  --build-arg BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --build-arg BRANCH="$(git branch --show-current)" \
  -t webhook:local .
```

If these arguments are omitted, the image identifies itself as a development build.
