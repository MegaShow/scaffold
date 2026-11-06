import assert from 'node:assert/strict';
import { test } from 'node:test';

import { compareVersions, detectVersion, parseVersion, resolveVersion } from './release.mjs';

test('resolveVersion', () => {
  assert.equal(resolveVersion('0.1.0', 'next'), '0.1.1');
  assert.equal(resolveVersion('0.1.0', 'prepatch'), '0.1.1-beta.1');
  assert.equal(resolveVersion('0.1.0', 'preminor'), '0.2.0-beta.1');
  assert.equal(resolveVersion('0.1.0', 'premajor'), '1.0.0-beta.1');
  assert.equal(resolveVersion('0.2.0-beta.1', 'next'), '0.2.0-beta.2');
  assert.equal(resolveVersion('0.2.0-rc.1', 'next'), '0.2.0-rc.2');
  assert.equal(resolveVersion('0.2.0-rc.2', 'minor'), '0.2.0');
  assert.equal(resolveVersion('0.2.0-beta.2', 'next', '0.2.0-rc.1'), '0.2.0-rc.1');
});

test('parseVersion', () => {
  for (const version of ['2.0.0', '0.01.0', '0.1.0-beta.01', '0.1.0\n', '0.1.0+build']) {
    assert.throws(() => parseVersion(version), /Invalid release version/);
  }
});

test('compareVersions', () => {
  assert.ok(compareVersions('0.2.0-beta.9', '0.2.0-beta.10') < 0);
  assert.ok(compareVersions('0.2.0-beta.99', '0.2.0-rc.1') < 0);
  assert.ok(compareVersions('0.2.0-rc.1', '0.2.0') < 0);
});

test('detectVersion', () => {
  assert.equal(detectVersion('0.1.0', '0.1.0'), undefined);
  assert.equal(detectVersion('0.1.0', '0.0.0'), undefined);
  assert.deepEqual(detectVersion('0.0.0', '0.1.0'), { version: '0.1.0', channel: 'latest' });
  assert.deepEqual(detectVersion('0.1.0', '0.2.0-beta.1'), { version: '0.2.0-beta.1', channel: 'beta' });
  assert.deepEqual(detectVersion('0.2.0-beta.1', '0.2.0-rc.1'), { version: '0.2.0-rc.1', channel: 'rc' });
  assert.throws(() => detectVersion('0.2.0', '0.1.0'), /must advance/);
});
