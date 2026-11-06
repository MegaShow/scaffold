import { execFileSync } from 'node:child_process';
import { appendFileSync, readFileSync, writeFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';

const manifest = 'typescript/client/package.json';

// 校验版本是否符合当前 Go 模块支持的格式，并拆出版本号、预发布渠道和序号
export function parseVersion(version) {
  const match = /^([01])\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-(beta|rc)\.(0|[1-9]\d*))?$/.exec(version);
  if (!match || match[0] !== version) {
    throw new Error(`Invalid release version: ${version} (expected Go-compatible v0/v1 with optional beta.N or rc.N)`);
  }
  const [, major, minor, patch, channel, sequence] = match;
  return { major: BigInt(major), minor: BigInt(minor), patch: BigInt(patch), channel, sequence: BigInt(sequence ?? 0) };
}

// 按版本优先级比较，左边较旧返回 -1，相等返回 0，较新返回 1
export function compareVersions(left, right) {
  const a = parseVersion(left);
  const b = parseVersion(right);
  const rank = { beta: 0, rc: 1, latest: 2 };
  for (const [x, y] of [
    [a.major, b.major],
    [a.minor, b.minor],
    [a.patch, b.patch],
    [rank[a.channel ?? 'latest'], rank[b.channel ?? 'latest']],
    [a.sequence, b.sequence],
  ]) {
    if (x !== y) {
      return x < y ? -1 : 1;
    }
  }
  return 0;
}

// 优先使用手填版本，否则按所选递增方式从当前版本计算目标版本
export function resolveVersion(current, release, explicit) {
  if (explicit) {
    parseVersion(explicit);
    return explicit;
  }
  const { major, minor, patch, channel, sequence } = parseVersion(current);
  const core = `${major}.${minor}.${patch}`;
  const preid = channel ?? 'beta';
  switch (release) {
    case 'next':
      return channel ? `${core}-${channel}.${sequence + 1n}` : `${major}.${minor}.${patch + 1n}`;
    // 按 SemVer 的对应层级转正，例如 0.2.0-beta.2 选择 minor 后得到 0.2.0
    case 'patch':
      return channel ? core : `${major}.${minor}.${patch + 1n}`;
    case 'minor':
      return channel && patch === 0n ? core : `${major}.${minor + 1n}.0`;
    case 'major':
      return channel && minor === 0n && patch === 0n ? core : `${major + 1n}.0.0`;
    case 'prepatch':
      return `${major}.${minor}.${patch + 1n}-${preid}.1`;
    case 'preminor':
      return `${major}.${minor + 1n}.0-${preid}.1`;
    case 'premajor':
      return `${major + 1n}.0.0-${preid}.1`;
    default:
      throw new Error(`Unknown release type: ${release}`);
  }
}

// 执行 Git 查询并返回去掉末尾空白的输出，命令失败时抛出错误
function git(...args) {
  return execFileSync('git', args, { encoding: 'utf8' }).trimEnd();
}

// 将键值追加到 GitHub Actions 的步骤输出文件，供后续步骤和任务读取
function output(values) {
  appendFileSync(
    process.env.GITHUB_OUTPUT,
    Object.entries(values)
      .map(([key, value]) => `${key}=${value}\n`)
      .join(''),
  );
}

// 校验目标版本未被占用，并更新 npm 清单以准备发布 PR
function prepare(release, explicit) {
  const source = JSON.parse(readFileSync(manifest, 'utf8'));
  const version = resolveVersion(source.version, release, explicit);
  const tags = git('tag', '--list', 'v' + version, 'go/mux/v' + version);
  if (compareVersions(version, source.version) <= 0 || tags) {
    throw new Error(`Version ${version} is already reserved or does not advance ${source.version}`);
  }

  source.version = version;
  writeFileSync(manifest, `${JSON.stringify(source, null, 2)}\n`);
  output({ version });
}

// 版本号递增时返回发布版本和 npm 渠道，保留 0.0.0 作为未发布的初始版本
export function detectVersion(previous, version) {
  if (version === previous || version === '0.0.0') {
    return;
  }
  const { channel } = parseVersion(version);
  if (compareVersions(version, previous) < 0) {
    throw new Error(`Release version must advance ${previous}`);
  }
  return { version, channel: channel ?? 'latest' };
}

// 对比当前提交与指定提交的 npm 清单，检测到新版本时将结果交给工作流
function detect(before) {
  if (!/^[0-9a-f]{40}$/.test(before ?? '') || /^0+$/.test(before)) {
    return;
  }
  const version = JSON.parse(readFileSync(manifest, 'utf8')).version;
  const previous = JSON.parse(git('show', `${before}:${manifest}`)).version;
  const release = detectVersion(previous, version);
  if (release) {
    output(release);
  }
}

// 只在直接运行脚本时分派子命令，测试或其他脚本导入时不执行发布流程
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const [, , command, first, second] = process.argv;
  switch (command) {
    case 'prepare':
      prepare(first?.split(' ')[0], second);
      break;
    case 'detect':
      detect(first);
      break;
    default:
      throw new Error(`Unknown release command: ${command}`);
  }
}
