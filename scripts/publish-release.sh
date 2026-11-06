#!/usr/bin/env bash
set -euo pipefail

version=${1:?Missing release version}
channel=${2:?Missing npm channel}
artifacts=${3:?Missing artifact directory}
package=@icytown/scaffold-client
tarball="$artifacts/npm/icytown-scaffold-client-$version.tgz"
tag="v$version"
sha=$(git rev-parse HEAD)
(cd "$artifacts/github" && sha256sum -c checksums.txt)
integrity="sha512-$(openssl dgst -sha512 -binary "$tarball" | openssl base64 -A)"

# 在客户端目录运行 npm，避免触发仓库根目录的 pnpm 开发工具约束
# 只把不存在的版本视为待发布，认证或网络错误需要明确失败
npm_view() {
  local result
  if result=$(cd typescript/client && npm view --json "$@"); then
    printf '%s' "$result"
  elif jq -e '.error.code == "E404"' <<< "$result" > /dev/null; then
    printf 'null'
  else
    printf '%s\n' "$result" >&2
    return 1
  fi
}

published=$(npm_view "$package@$version" dist.integrity | jq -r '. // empty')
if [[ -n "$published" && "$published" != "$integrity" ]]; then
  echo "npm version $package@$version exists with different package contents" >&2
  exit 1
fi
if [[ -z "$published" ]]; then
  current=$(npm_view "$package@$channel" version | jq -r '. // empty')
  if [[ -n "$current" ]]; then
    # 避免旧发布的重试把 npm 渠道从较新版本移回旧版本
    node --input-type=module -e '
      import { compareVersions } from "./scripts/release.mjs"
      const [version, current] = process.argv.slice(1)
      if (current !== "0.0.0-stage" && compareVersions(version, current) <= 0) {
        throw new Error(`Publishing ${version} would move the npm channel backward from ${current}`)
      }
    ' "$version" "$current"
  fi
fi

# 已有标签可以复用，但必须仍指向最初通过检查的提交
refs=$(git ls-remote --tags origin "refs/tags/$tag" "refs/tags/$tag^{}" \
  "refs/tags/go/mux/$tag" "refs/tags/go/mux/$tag^{}")
missing=()
for ref in "refs/tags/$tag" "refs/tags/go/mux/$tag"; do
  target=$(awk -v ref="$ref" '$2 == ref { direct = $1 } $2 == ref "^{}" { peeled = $1 } END { print peeled ? peeled : direct }' <<< "$refs")
  if [[ -z "$target" ]]; then
    missing+=("$sha:$ref")
  elif [[ "$target" != "$sha" ]]; then
    echo "Tag $ref points to $target instead of $sha" >&2
    exit 1
  fi
done

# 同一次推送创建两个 Go 标签，避免只创建其中一个标签
if ((${#missing[@]})); then git push --atomic origin "${missing[@]}"; fi

if [[ -z "$published" ]]; then
  (cd typescript/client && npm publish "$tarball" --ignore-scripts --access public --tag "$channel")
fi

# 先传附件再公开 Release，兼容启用了不可变发布的仓库
release=$(gh api --paginate --slurp "repos/$GITHUB_REPOSITORY/releases?per_page=100" |
  jq -c --arg tag "$tag" '[.[][] | select(.tag_name == $tag)] | first // empty')
if [[ -z "$release" ]]; then
  flags=()
  if [[ "$channel" != latest ]]; then flags+=(--prerelease --latest=false); fi
  gh release create "$tag" --verify-tag --draft --title "$tag" --generate-notes "${flags[@]}"
  release=$(gh api --paginate --slurp "repos/$GITHUB_REPOSITORY/releases?per_page=100" |
    jq -ce --arg tag "$tag" '[.[][] | select(.tag_name == $tag)] | first')
fi
release_id=$(jq -er '.id' <<< "$release")
assets=$(gh api --paginate --slurp "repos/$GITHUB_REPOSITORY/releases/$release_id/assets?per_page=100")
for asset in "$artifacts/github/"*; do
  name=${asset##*/}
  existing=$(jq -c --arg name "$name" '[.[][] | select(.name == $name)] | first // empty' <<< "$assets")
  if [[ -z "$existing" ]]; then
    gh release upload "$tag" "$asset"
    continue
  fi
  expected="sha256:$(sha256sum "$asset" | cut -d ' ' -f 1)"
  if [[ $(jq -r '.digest' <<< "$existing") != "$expected" ]]; then
    echo "GitHub Release asset $name exists with different contents or has no digest" >&2
    exit 1
  fi
done
if [[ $(jq -r '.draft' <<< "$release") == true ]]; then
  gh release edit "$tag" --draft=false
fi
