#!/usr/bin/env bash
set -euo pipefail

version=${1:?Missing release version}
channel=${2:?Missing npm channel}
artifacts=${3:?Missing artifact directory}
package=@icytown/scaffold-client
tarball="$artifacts/npm/icytown-scaffold-client-$version.tgz"
tag="v$version"
phase='Resolve release commit'
# 所有非零退出都标记当前发布阶段，包括显式退出和命令执行失败
trap 'status=$?; if ((status != 0)); then printf "::error title=Release %s failed::%s (exit %s)\n" "$tag" "$phase" "$status" >&2; fi' EXIT
sha=$(git rev-parse HEAD)
phase='Verify release artifacts'
printf '%s\n' "$phase"
# 将文件名在前的清单转回 sha256sum 的校验格式
(cd "$artifacts/github" && awk 'NF != 2 {exit 1} {print $2 "  " $1}' checksums.txt | sha256sum --strict -c -)
digest=$(openssl dgst -sha512 -binary "$tarball" | openssl base64 -A)
integrity="sha512-$digest"

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

phase='Check npm version and channel'
printf '%s\n' "$phase"
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
phase='Check and push Git tags'
printf '%s\n' "$phase"
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
  phase='Publish npm package'
  printf '%s\n' "$phase"
  (cd typescript/client && npm publish "$tarball" --ignore-scripts --access public --tag "$channel")
else
  printf 'npm version %s@%s already matches the artifact\n' "$package" "$version"
fi

# 查找已有 Release，以便重跑时补齐缺失附件
phase='Look up GitHub Release'
printf '%s\n' "$phase"
release=$(gh api --paginate --slurp "repos/$GITHUB_REPOSITORY/releases?per_page=100" |
  jq -c --arg tag "$tag" '[.[][] | select(.tag_name == $tag)] | first // empty')
if [[ -z "$release" ]]; then
  flags=()
  if [[ "$channel" != latest ]]; then flags+=(--prerelease --latest=false); fi
  # 由 gh 使用刚创建的草稿直接上传附件，再公开 Release，避免立即按 tag 查找草稿
  phase='Create GitHub Release and upload assets'
  printf '%s\n' "$phase"
  gh release create "$tag" "$artifacts/github/"* --verify-tag --title "$tag" --generate-notes "${flags[@]}"
  exit 0
fi
phase='List existing GitHub Release assets'
printf '%s\n' "$phase"
release_id=$(jq -er '.id' <<< "$release")
assets=$(gh api --paginate --slurp "repos/$GITHUB_REPOSITORY/releases/$release_id/assets?per_page=100")
for asset in "$artifacts/github/"*; do
  name=${asset##*/}
  phase="Check or upload GitHub Release asset $name"
  printf '%s\n' "$phase"
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
phase='Publish existing GitHub Release draft'
draft=$(jq -r '.draft' <<< "$release")
if [[ "$draft" == true ]]; then
  printf '%s\n' "$phase"
  gh release edit "$tag" --draft=false
fi
