#!/bin/sh
# builds release tarballs into dist/ for every supported os/arch.
# usage: scripts/build-release.sh <version>, run from the repo root.
set -eu

if [ $# -ne 1 ] || [ -z "$1" ]; then
  echo "usage: $0 <version>" >&2
  exit 2
fi
ver="$1"

mkdir -p dist
for t in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
  os="${t%/*}"
  arch="${t#*/}"
  # CGO_ENABLED=0 because a native linux/amd64 build defaults to cgo and links glibc via net/os/user
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -ldflags "-s -w -X main.version=$ver" -o dist/mdo ./cmd/mdo
  tar -C dist -czf "dist/mdo_${ver}_${os}_${arch}.tar.gz" mdo
  # remove the bare binary so the next target does not collide and dist holds only tarballs
  rm dist/mdo
done
