# Releasing

## Normal Flow

Releases are created **manually**. This repository has no GitHub Actions release workflow.

The intended procedure is:

1. Move the release notes for the target version into `CHANGELOG.md` under a `## [x.y.z]` section (and leave a fresh empty `## [Unreleased]` heading).
2. Update `VERSION` to the matching bare semantic version, for example `0.1.1`.
3. Update `.github/badges/version.svg` so the badge matches `VERSION`.
4. Commit those changes along with any release-ready code and push to `master`.
5. Build Linux and Windows binaries locally:

   ```bash
   VERSION="v$(tr -d '[:space:]' < VERSION)"
   mkdir -p dist/linux dist/windows release

   GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
     go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" \
     -o "dist/linux/terraform-provider-logscale_${VERSION}" .

   GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
     go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" \
     -o "dist/windows/terraform-provider-logscale_${VERSION}.exe" .

   (cd dist/linux && zip -9 "../../release/terraform-provider-logscale_${VERSION}_linux_amd64.zip" "terraform-provider-logscale_${VERSION}")
   (cd dist/windows && zip -9 "../../release/terraform-provider-logscale_${VERSION}_windows_amd64.zip" "terraform-provider-logscale_${VERSION}.exe")
   sha256sum release/*.zip > "release/terraform-provider-logscale_${VERSION}_SHA256SUMS"
   ```

6. Create a git tag and a GitHub Release from the curated changelog section, attaching the zip artifacts and SHA256SUMS:

   ```bash
   TAG="v$(tr -d '[:space:]' < VERSION)"
   git tag "${TAG}"
   git push origin "${TAG}"
   gh release create "${TAG}" \
     --title "${TAG}" \
     --notes-file <(awk -v version="$(tr -d '[:space:]' < VERSION)" '
       BEGIN { in_section=0; found=0 }
       $0 ~ "^## \\[" version "\\]" { in_section=1; found=1; next }
       $0 ~ "^## \\[" && in_section==1 { exit }
       in_section==1 { print }
       END { if (found==0) exit 2 }
     ' CHANGELOG.md) \
     release/*.zip release/*SHA256SUMS
   ```

## Rules

- `VERSION` must contain a bare semantic version such as `0.1.0`
- `CHANGELOG.md` should contain a matching `## [x.y.z]` section for the version being released
- if a tag like `v0.1.0` already exists, do not reuse it — bump `VERSION` first
- normal pushes that do not change `VERSION` do not create a release

## Examples

- `VERSION = 0.1.0`, commit, tag `v0.1.0`, publish GitHub Release: creates release `v0.1.0`
- later bump `VERSION` to `0.1.1`, tag, and publish: creates release `v0.1.1`
- push code changes without changing `VERSION`: no release is created
