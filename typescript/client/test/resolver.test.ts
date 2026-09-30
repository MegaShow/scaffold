import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { HTTPError, JsonResolver, type Response } from '../src/index.ts';

describe('JsonResolver', () => {
  describe('resolve', () => {
    // 2xx 响应解析为 JSON 值，仅零字节响应体返回 undefined
    const successCases: { name: string; status: number; body: string; want: unknown }[] = [
      { name: 'object_value', status: 200, body: '{"name":"Alice"}', want: { name: 'Alice' } },
      { name: 'array_value', status: 200, body: '[1,2]', want: [1, 2] },
      { name: 'empty_object', status: 200, body: '{}', want: {} },
      { name: 'empty_array', status: 200, body: '[]', want: [] },
      { name: 'null_value', status: 200, body: 'null', want: null },
      { name: 'true_value', status: 200, body: 'true', want: true },
      { name: 'false_value', status: 200, body: 'false', want: false },
      { name: 'zero_value', status: 200, body: '0', want: 0 },
      { name: 'number_value', status: 200, body: '-1.25e2', want: -125 },
      { name: 'empty_string', status: 200, body: '""', want: '' },
      { name: 'string_with_escape', status: 200, body: '"line\\n\\"quoted\\""', want: 'line\n"quoted"' },
      { name: 'utf8_value', status: 200, body: '{"text":"你好🌍"}', want: { text: '你好🌍' } },
      {
        name: 'nested_value',
        status: 200,
        body: '{"items":[{"id":1},null],"active":true}',
        want: { items: [{ id: 1 }, null], active: true },
      },
      { name: 'surrounding_whitespace', status: 200, body: ' \n {"name":"Alice"}\t\r\n', want: { name: 'Alice' } },
      { name: 'created_status', status: 201, body: '{"id":1}', want: { id: 1 } },
      { name: 'last_success_status', status: 299, body: '[1,2]', want: [1, 2] },
      { name: 'empty_ok_response', status: 200, body: '', want: undefined },
      { name: 'empty_no_content_response', status: 204, body: '', want: undefined },
      { name: 'empty_last_success_response', status: 299, body: '', want: undefined },
    ];
    for (const { name, status, body, want } of successCases) {
      it(name, () => {
        assert.deepEqual(new JsonResolver().resolve(createResponse(status, body)), want);
      });
    }

    // 成功响应中的非法 JSON 直接抛出解析错误，纯空白也不视为空响应体
    const invalidCases = [
      { name: 'plain_text_success', body: 'not JSON' },
      { name: 'whitespace_only_success', body: ' \t\r\n' },
      { name: 'truncated_json_success', body: '{"name":' },
      { name: 'trailing_comma_success', body: '[1,2,]' },
      { name: 'trailing_content_success', body: '{"name":"Alice"} extra' },
    ];
    for (const { name, body } of invalidCases) {
      it(name, () => {
        assert.throws(() => new JsonResolver().resolve(createResponse(200, body)), SyntaxError);
      });
    }

    // 非 2xx 响应统一抛出 HTTPError，非法 JSON 回退为原文，并保留原始响应
    const errorCases: { name: string; status: number; body: string; want: unknown }[] = [
      { name: 'informational_status', status: 100, body: '{}', want: {} },
      { name: 'last_non_success_status', status: 199, body: '{}', want: {} },
      { name: 'first_redirect_status', status: 300, body: '{}', want: {} },
      { name: 'empty_redirect_response', status: 304, body: '', want: undefined },
      {
        name: 'plain_error_default_500',
        status: 500,
        body: '{"error":"internal failure"}',
        want: { error: 'internal failure' },
      },
      {
        name: 'bad_request_status',
        status: 400,
        body: '{"error":"400 bad request"}',
        want: { error: '400 bad request' },
      },
      {
        name: 'unsupported_media_type_status',
        status: 415,
        body: '{"error":"415 unsupported media type"}',
        want: { error: '415 unsupported media type' },
      },
      {
        name: 'wrapped_bad_request_status',
        status: 400,
        body: '{"error":"invalid payload: 400 bad request"}',
        want: { error: 'invalid payload: 400 bad request' },
      },
      {
        name: 'error_with_escape',
        status: 500,
        body: '{"error":"line\\n\\"quoted\\""}',
        want: { error: 'line\n"quoted"' },
      },
      { name: 'custom_error_shape', status: 404, body: '{"message":"not found"}', want: { message: 'not found' } },
      { name: 'array_error', status: 400, body: '["invalid","missing"]', want: ['invalid', 'missing'] },
      { name: 'null_error', status: 500, body: 'null', want: null },
      { name: 'false_error', status: 500, body: 'false', want: false },
      { name: 'zero_error', status: 500, body: '0', want: 0 },
      { name: 'string_error', status: 500, body: '"internal failure"', want: 'internal failure' },
      { name: 'empty_json_string_error', status: 500, body: '""', want: '' },
      { name: 'empty_error_response', status: 500, body: '', want: undefined },
      { name: 'plain_text_error', status: 500, body: 'internal failure', want: 'internal failure' },
      { name: 'html_error', status: 502, body: '<html>Bad Gateway</html>', want: '<html>Bad Gateway</html>' },
      { name: 'truncated_json_error', status: 500, body: '{"error":', want: '{"error":' },
      { name: 'whitespace_only_error', status: 500, body: ' \t\r\n', want: ' \t\r\n' },
      { name: 'utf8_text_error', status: 500, body: '请求失败🌍', want: '请求失败🌍' },
    ];
    for (const { name, status, body, want } of errorCases) {
      it(name, () => {
        const response = createResponse(status, body);
        assert.throws(
          () => new JsonResolver().resolve(response),
          (error: unknown) => {
            assert.ok(error instanceof HTTPError);
            assert.equal(error.response, response);
            assert.deepEqual(error.data, want);
            return true;
          },
        );
      });
    }
  });
});

function createResponse(status: number, body: string): Response {
  return {
    status,
    headers: {},
    body: new TextEncoder().encode(body),
  };
}
