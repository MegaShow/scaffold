#!/usr/bin/env bash
set -euo pipefail

version=${1:?Missing release version}
output=${2:?Missing artifact directory}
stage=$(mktemp -d)
trap 'rm -rf -- "$stage"' EXIT
mkdir -p "$output/npm" "$output/github"
cp LICENSE "$stage/LICENSE"

# 固定归档时间和文件顺序，确保从同一提交重新构建时可核对产物
for target in linux/amd64 windows/amd64 darwin/arm64; do
  os=${target%/*}
  arch=${target#*/}
  binary=protoc-gen-scaffold
  if [[ "$os" == windows ]]; then binary+=.exe; fi
  asset="$output/github/scaffold-${version}-${os}-${arch}"
  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -buildvcs=false \
    -ldflags "-X go.icytown.com/scaffold/internal/generator.injectedVersion=v$version" \
    -o "$stage/$binary" ./cmd/protoc-gen-scaffold

  if [[ "$os" == windows ]]; then
    TZ=UTC touch -t 200001010000 "$stage/$binary" "$stage/LICENSE"
    (cd "$stage" && TZ=UTC zip -X -q "$asset.zip" "$binary" LICENSE)
  else
    tar --mtime='2000-01-01 UTC' --owner=0 --group=0 --numeric-owner \
      -C "$stage" -cf - "$binary" LICENSE | gzip -n > "$asset.tar.gz"
  fi
done
(cd "$output/github" && sha256sum -- *.tar.gz *.zip | awk '{print $2, $1}' > checksums.txt)

# 先构建再打包，发布阶段只上传已验证的 tarball，不执行包脚本
pnpm --filter @icytown/scaffold-client run build
(cd typescript/client && npm pack --ignore-scripts --pack-destination "$output/npm")
package="$output/npm/icytown-scaffold-client-$version.tgz"
tar -xOzf "$package" package/package.json | jq -e --arg version "$version" \
  '.name == "@icytown/scaffold-client" and .version == $version' > /dev/null
tar -xOzf "$package" package/dist/index.js > /dev/null
tar -xOzf "$package" package/dist/index.d.ts > /dev/null
