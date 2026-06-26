#!/usr/bin/env sh
set -eu

VERSION=${1:-${NPANEL_VERSION:-v1.0.10}}
case "$VERSION" in
  v*) ;;
  *) VERSION="v$VERSION" ;;
esac

CLEAN_VERSION=${VERSION#v}
ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
DIST_DIR=${DIST_DIR:-"$ROOT_DIR/dist/release"}
TARGETS=${TARGETS:-"linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64"}

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

for target in $TARGETS; do
  GOOS=$(printf "%s" "$target" | cut -d / -f 1)
  GOARCH=$(printf "%s" "$target" | cut -d / -f 2)
  GOARM=$(printf "%s" "$target" | cut -d / -f 3)

  suffix="$GOOS-$GOARCH"
  if [ -n "$GOARM" ]; then
    suffix="$suffix-$GOARM"
  fi

  package="npanel-backend-$CLEAN_VERSION-$suffix"
  workdir="$DIST_DIR/$package"
  mkdir -p "$workdir/configs"

  binary="npanel"
  if [ "$GOOS" = "windows" ]; then
    binary="npanel.exe"
  fi

  echo "==> Building $package"
  if [ -n "$GOARM" ]; then
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" GOARM="${GOARM#v}" \
      go build -trimpath -ldflags "-s -w -X main.Version=$VERSION" \
      -o "$workdir/$binary" ./cmd/npanel
  else
    CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
      go build -trimpath -ldflags "-s -w -X main.Version=$VERSION" \
      -o "$workdir/$binary" ./cmd/npanel
  fi

  cp README.md README.zh-CN.md LICENSE openapi.yaml "$workdir/"
  cp configs/config.yaml configs/config.docker.yaml "$workdir/configs/"

  (
    cd "$DIST_DIR"
    tar -czf "$package.tar.gz" "$package"
    rm -rf "$package"
  )
done

(
  cd "$DIST_DIR"
  shasum -a 256 ./*.tar.gz > SHA256SUMS
)

echo "Release artifacts written to $DIST_DIR"
