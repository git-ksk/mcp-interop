#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <version> <commit> <build-date>" >&2
  exit 2
fi

version="$1"
commit="$2"
build_date="$3"
archive_version="${version#v}"
workdirs=()

cleanup() {
  local dir
  for dir in "${workdirs[@]}"; do
    [[ -n "$dir" ]] && rm -rf "$dir"
  done
}
trap cleanup EXIT

if [[ -z "$archive_version" || -z "$commit" || -z "$build_date" ]]; then
  echo "version, commit, and build-date must be non-empty" >&2
  exit 2
fi

rm -rf dist
mkdir -p dist

targets=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
  "windows arm64"
)

for target in "${targets[@]}"; do
  read -r goos goarch <<<"$target"
  name="mcp-interop_${archive_version}_${goos}_${goarch}"
  workdir="$(mktemp -d)"
  workdirs+=("$workdir")
  binary="mcp-interop"
  if [[ "$goos" == "windows" ]]; then
    binary="mcp-interop.exe"
  fi

  env CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build \
      -trimpath \
      -ldflags "-s -w -X main.version=${version} -X main.commit=${commit} -X main.buildDate=${build_date}" \
      -o "$workdir/$binary" \
      ./cmd/mcp-interop

  cp LICENSE "$workdir/LICENSE"
  if [[ "$goos" == "windows" ]]; then
    (
      cd "$workdir"
      zip -q "$OLDPWD/dist/${name}.zip" "$binary" LICENSE
    )
  else
    tar -C "$workdir" -czf "dist/${name}.tar.gz" "$binary" LICENSE
  fi
  rm -rf "$workdir"
  workdir=""
done

(
  cd dist
  sha256sum ./*.tar.gz ./*.zip > checksums.txt
)
