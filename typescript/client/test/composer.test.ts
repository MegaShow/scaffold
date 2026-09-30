import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { AdaptComposer, FormComposer, JsonComposer, type Request } from '../src/index.ts';

describe('AdaptComposer', () => {
  describe('constructor', () => {
    it('uses_defaults_when_options_are_omitted_or_empty', () => {
      for (const composer of [new AdaptComposer(), new AdaptComposer({})]) {
        assert.ok(composer.form instanceof FormComposer);
        assert.ok(composer.json instanceof JsonComposer);
      }
    });

    it('preserves_overrides_and_defaults_each_missing_composer', () => {
      const custom = { compose: () => undefined };
      const formOnly = new AdaptComposer({ form: custom });
      assert.equal(formOnly.form, custom);
      assert.ok(formOnly.json instanceof JsonComposer);
      const jsonOnly = new AdaptComposer({ json: custom });
      assert.ok(jsonOnly.form instanceof FormComposer);
      assert.equal(jsonOnly.json, custom);
      const both = new AdaptComposer({ form: custom, json: custom });
      assert.equal(both.form, custom);
      assert.equal(both.json, custom);
    });
  });

  describe('compose', () => {
    // GET 始终使用表单，其余请求按 Content-Type 选择编码方式
    const testCases: {
      name: string;
      method: Request['method'];
      headers?: Request['headers'];
      want: 'form' | 'json' | undefined;
    }[] = [
      { name: 'get_without_headers', method: 'GET', want: 'form' },
      { name: 'get_with_json', method: 'GET', headers: { 'Content-Type': 'application/json' }, want: 'form' },
      { name: 'get_with_unsupported_type', method: 'GET', headers: { 'Content-Type': 'text/plain' }, want: 'form' },
      {
        name: 'post_form',
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        want: 'form',
      },
      { name: 'post_json', method: 'POST', headers: { 'Content-Type': 'application/json' }, want: 'json' },
      { name: 'put_json', method: 'PUT', headers: { 'Content-Type': 'application/json' }, want: 'json' },
      {
        name: 'put_form',
        method: 'PUT',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        want: 'form',
      },
      { name: 'delete_unsupported', method: 'DELETE', headers: { 'Content-Type': 'text/plain' }, want: undefined },
      { name: 'patch_json', method: 'PATCH', headers: { 'Content-Type': 'application/json' }, want: 'json' },
      {
        name: 'delete_form',
        method: 'DELETE',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        want: 'form',
      },
      {
        name: 'json_case_and_parameters',
        method: 'POST',
        headers: { 'cOnTeNt-TyPe': ' APPLICATION/JSON ; charset=utf-8' },
        want: 'json',
      },
      {
        name: 'form_case_and_parameters',
        method: 'POST',
        headers: { 'content-type': ' APPLICATION/X-WWW-FORM-URLENCODED ; charset=utf-8' },
        want: 'form',
      },
      {
        name: 'header_array_first_value',
        method: 'POST',
        headers: { 'Content-Type': ['application/json', 'text/plain'] },
        want: 'json',
      },
      { name: 'missing_headers', method: 'POST', want: undefined },
      { name: 'missing_content_type', method: 'POST', headers: { Accept: 'application/json' }, want: undefined },
      { name: 'empty_content_type', method: 'POST', headers: { 'Content-Type': '' }, want: undefined },
      { name: 'empty_header_array', method: 'POST', headers: { 'Content-Type': [] }, want: undefined },
      { name: 'unsupported_content_type', method: 'POST', headers: { 'Content-Type': 'text/plain' }, want: undefined },
      {
        name: 'unsupported_json_suffix',
        method: 'POST',
        headers: { 'Content-Type': 'application/problem+json' },
        want: undefined,
      },
      {
        name: 'ignores_later_header_values',
        method: 'POST',
        headers: { 'Content-Type': ['text/plain', 'application/json'] },
        want: undefined,
      },
    ];
    for (const { name, method, headers, want } of testCases) {
      it(name, () => {
        const request: Request = { method, path: '/', headers };
        const value = { name: 'Alice', age: 42, tags: ['a', 'b'] };
        const composer = new AdaptComposer({ form: new FormComposer(), json: new JsonComposer() });
        if (want === undefined) {
          assert.throws(() => composer.compose(request, value), TypeError);
          assert.equal(request.query, undefined);
          assert.equal(request.body, undefined);
        } else {
          composer.compose(request, value);
          if (method === 'GET') {
            assert.deepEqual(request.query, { name: 'Alice', age: '42', tags: ['a', 'b'] });
            assert.equal(request.body, undefined);
          } else {
            const body =
              want === 'form' ? 'name=Alice&age=42&tags=a&tags=b' : '{"name":"Alice","age":42,"tags":["a","b"]}';
            assert.deepEqual(request.body, new TextEncoder().encode(body));
            assert.equal(request.query, undefined);
          }
        }
      });
    }

    // 底层组装器抛出的错误应原样透传
    const errorCases: { name: string; method: Request['method']; contentType?: string }[] = [
      { name: 'get_form_error_passthrough', method: 'GET' },
      { name: 'post_form_error_passthrough', method: 'POST', contentType: 'application/x-www-form-urlencoded' },
      { name: 'post_json_error_passthrough', method: 'POST', contentType: 'application/json' },
    ];
    for (const { name, method, contentType } of errorCases) {
      it(name, () => {
        const error = new Error('Composition failed');
        const failingComposer = {
          compose: () => {
            throw error;
          },
        };
        const composer = new AdaptComposer({ form: failingComposer, json: failingComposer });
        const request: Request = {
          method,
          path: '/',
          headers: contentType === undefined ? {} : { 'Content-Type': contentType },
        };
        assert.throws(
          () => composer.compose(request, {}),
          (caught: unknown) => caught === error,
        );
      });
    }
  });
});

describe('FormComposer', () => {
  describe('compose', () => {
    // 标量数组使用重复字段，嵌套对象使用点号，对象数组使用下标
    const successCases: { name: string; value: unknown; want: Record<string, string | string[]>; body: string }[] = [
      {
        name: 'parser_request',
        value: { name: 'Alice Smith', age: 42, tags: ['a', 'b'] },
        want: { name: 'Alice Smith', age: '42', tags: ['a', 'b'] },
        body: 'name=Alice+Smith&age=42&tags=a&tags=b',
      },
      { name: 'object_value', value: { name: 'Alice' }, want: { name: 'Alice' }, body: 'name=Alice' },
      {
        name: 'scalar_fields',
        value: { active: false, count: 0, ratio: -1.5, text: '' },
        want: { active: 'false', count: '0', ratio: '-1.5', text: '' },
        body: 'active=false&count=0&ratio=-1.5&text=',
      },
      { name: 'true_value', value: { active: true }, want: { active: 'true' }, body: 'active=true' },
      { name: 'null_value', value: null, want: {}, body: '' },
      { name: 'undefined_value', value: undefined, want: {}, body: '' },
      { name: 'empty_object', value: {}, want: {}, body: '' },
      { name: 'omitted_fields', value: { a: null, b: undefined, c: [], d: {} }, want: {}, body: '' },
      {
        name: 'scalar_array',
        value: { tags: ['a', 'b', null, undefined, false, 0] },
        want: { tags: ['a', 'b', 'false', '0'] },
        body: 'tags=a&tags=b&tags=false&tags=0',
      },
      {
        name: 'nested_object',
        value: { user: { name: 'Alice', address: { city: 'Paris' } } },
        want: { 'user.name': 'Alice', 'user.address.city': 'Paris' },
        body: 'user.name=Alice&user.address.city=Paris',
      },
      {
        name: 'object_array',
        value: { users: [{ name: 'Alice' }, { name: 'Bob' }] },
        want: { 'users[0].name': 'Alice', 'users[1].name': 'Bob' },
        body: 'users%5B0%5D.name=Alice&users%5B1%5D.name=Bob',
      },
      {
        name: 'nested_array',
        value: { matrix: [[1, 2], [3]] },
        want: { 'matrix[0]': ['1', '2'], 'matrix[1]': '3' },
        body: 'matrix%5B0%5D=1&matrix%5B0%5D=2&matrix%5B1%5D=3',
      },
      {
        name: 'escaped_keys_and_values',
        value: { 'a b&=': 'x y+&=/' },
        want: { 'a b&=': 'x y+&=/' },
        body: 'a+b%26%3D=x+y%2B%26%3D%2F',
      },
      {
        name: 'utf8_value',
        value: { text: '你好🌍' },
        want: { text: '你好🌍' },
        body: 'text=%E4%BD%A0%E5%A5%BD%F0%9F%8C%8D',
      },
      {
        name: 'bytes_value',
        value: { text: new TextEncoder().encode('你好') },
        want: { text: '你好' },
        body: 'text=%E4%BD%A0%E5%A5%BD',
      },
      { name: 'empty_bytes', value: { text: new Uint8Array() }, want: { text: '' }, body: 'text=' },
      {
        name: 'bytes_array',
        value: { text: [new Uint8Array([65]), new Uint8Array([66])] },
        want: { text: ['A', 'B'] },
        body: 'text=A&text=B',
      },
    ];
    for (const { name, value, want, body } of successCases) {
      // 同一组数据分别验证 GET 查询参数和 POST 表单请求体
      for (const method of ['GET', 'POST'] as const) {
        it(`${name}_${method}`, () => {
          const request: Request = { method, path: '/' };
          new FormComposer().compose(request, value);
          if (method === 'GET') {
            assert.deepEqual(request.query, want);
            assert.equal(request.body, undefined);
          } else {
            assert.deepEqual(request.body, new TextEncoder().encode(body));
            assert.equal(request.query, undefined);
          }
        });
      }
    }

    // 同名参数追加为数组，同时保留调用方持有的原查询对象及数组
    it('merges_query_values_without_mutating_existing_query', () => {
      const query = { preserved: 'value', scalar: false, array: ['old'], missing: undefined };
      const body = new Uint8Array([1]);
      const request: Request = { method: 'GET', path: '/', query, body };
      new FormComposer().compose(request, { scalar: [0, 1], array: ['a', 'b'], missing: 'new' });
      assert.deepEqual(request.query, {
        preserved: 'value',
        scalar: [false, '0', '1'],
        array: ['old', 'a', 'b'],
        missing: 'new',
      });
      assert.deepEqual(query, { preserved: 'value', scalar: false, array: ['old'], missing: undefined });
      assert.notEqual(request.query, query);
      assert.equal(request.body, body);
    });

    for (const method of ['POST', 'PUT', 'PATCH', 'DELETE'] as const) {
      it(`replaces_body_and_preserves_query_${method}`, () => {
        const query = { preserved: 'value' };
        const request: Request = { method, path: '/', query, body: new Uint8Array([1]) };
        new FormComposer().compose(request, { name: 'Alice' });
        assert.deepEqual(request.body, new TextEncoder().encode('name=Alice'));
        assert.equal(request.query, query);
      });
    }

    // 非法字段或无效 UTF-8 导致编码失败时，不应改写已有请求数据
    const invalidCases = [
      { name: 'root_string', value: 'Alice' },
      { name: 'root_number', value: 1 },
      { name: 'root_boolean', value: false },
      { name: 'root_bytes', value: new Uint8Array([65]) },
      { name: 'root_scalar_array', value: [1, 2] },
      { name: 'bigint_field', value: { valid: 'value', invalid: 1n } },
      { name: 'symbol_field', value: { invalid: Symbol('value') } },
      { name: 'function_field', value: { invalid: () => 1 } },
      { name: 'invalid_utf8', value: { invalid: new Uint8Array([0xff]) } },
    ];
    for (const { name, value } of invalidCases) {
      for (const method of ['GET', 'POST'] as const) {
        it(`${name}_${method}`, () => {
          const query = { preserved: 'value' };
          const body = new Uint8Array([1]);
          const request: Request = { method, path: '/', query, body };
          assert.throws(() => new FormComposer().compose(request, value), TypeError);
          assert.equal(request.query, query);
          assert.equal(request.body, body);
          assert.deepEqual(query, { preserved: 'value' });
          assert.deepEqual(body, new Uint8Array([1]));
        });
      }
    }
  });
});

describe('JsonComposer', () => {
  describe('compose', () => {
    // 验证 JSON 序列化语义，以及各层级字节数据的 Base64 编码
    const successCases: { name: string; value: unknown; want: string }[] = [
      {
        name: 'parser_request',
        value: { name: 'Alice', age: 42, tags: ['a', 'b'] },
        want: '{"name":"Alice","age":42,"tags":["a","b"]}',
      },
      { name: 'object_value', value: { name: 'Alice' }, want: '{"name":"Alice"}' },
      { name: 'array_value', value: [1, false, null], want: '[1,false,null]' },
      { name: 'empty_object', value: {}, want: '{}' },
      { name: 'empty_array', value: [], want: '[]' },
      { name: 'null_value', value: null, want: 'null' },
      { name: 'false_value', value: false, want: 'false' },
      { name: 'zero_value', value: 0, want: '0' },
      { name: 'empty_string', value: '', want: '""' },
      { name: 'string_with_escape', value: 'line\n"quoted"', want: '"line\\n\\"quoted\\""' },
      { name: 'utf8_value', value: { text: '你好🌍' }, want: '{"text":"你好🌍"}' },
      { name: 'nested_value', value: { items: [{ id: 1 }] }, want: '{"items":[{"id":1}]}' },
      { name: 'omitted_undefined_field', value: { name: 'Alice', missing: undefined }, want: '{"name":"Alice"}' },
      { name: 'undefined_array_item', value: [undefined], want: '[null]' },
      { name: 'non_finite_numbers', value: [NaN, Infinity, -Infinity], want: '[null,null,null]' },
      { name: 'root_bytes', value: new Uint8Array([0, 255, 128]), want: '"AP+A"' },
      { name: 'empty_bytes', value: new Uint8Array(), want: '""' },
      { name: 'nested_bytes', value: { data: new Uint8Array([65]) }, want: '{"data":"QQ=="}' },
      { name: 'bytes_array', value: [new Uint8Array([65]), new Uint8Array([66])], want: '["QQ==","Qg=="]' },
      { name: 'to_json_value', value: { toJSON: () => ({ id: 1 }) }, want: '{"id":1}' },
    ];
    for (const { name, value, want } of successCases) {
      it(name, () => {
        const query = { preserved: 'value' };
        const request: Request = { method: 'POST', path: '/', query, body: new Uint8Array([1]) };
        new JsonComposer().compose(request, value);
        assert.deepEqual(request.body, new TextEncoder().encode(want));
        assert.equal(request.query, query);
      });
    }

    // 无法序列化的数据应抛错，并保留已有请求体
    const circular: { self?: unknown } = {};
    circular.self = circular;
    const invalidCases = [
      { name: 'undefined_value', value: undefined },
      { name: 'symbol_value', value: Symbol('value') },
      { name: 'function_value', value: () => 1 },
      { name: 'bigint_value', value: 1n },
      { name: 'nested_bigint', value: { id: 1n } },
      { name: 'circular_value', value: circular },
      { name: 'undefined_to_json', value: { toJSON: () => undefined } },
    ];
    for (const { name, value } of invalidCases) {
      it(name, () => {
        const body = new Uint8Array([1]);
        const request: Request = { method: 'POST', path: '/', body };
        assert.throws(() => new JsonComposer().compose(request, value), TypeError);
        assert.equal(request.body, body);
        assert.deepEqual(body, new Uint8Array([1]));
      });
    }
  });
});
