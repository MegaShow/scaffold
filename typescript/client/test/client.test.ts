import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { HTTPError, type Response } from '../src/index.ts';

describe('HTTPError', () => {
  describe('constructor', () => {
    it('sets_error_identity_and_preserves_response_and_data', () => {
      const response: Response = {
        status: 404,
        headers: { 'Content-Type': 'application/json' },
        body: new TextEncoder().encode('{"message":"not found"}'),
      };
      const data = { message: 'not found' };
      const error = new HTTPError(response, data);
      assert.ok(error instanceof Error);
      assert.ok(error instanceof HTTPError);
      assert.equal(error.name, 'HTTPError');
      assert.equal(error.message, 'HTTP request failed with status 404');
      // 保留原始引用，便于调用方检查响应和业务错误数据
      assert.equal(error.response, response);
      assert.equal(error.data, data);
    });
  });
});
