import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { FetchClient, type Request } from '../src/index.ts';

describe('FetchClient', () => {
  describe('constructor', () => {
    it('rejects_relative_base_url_without_location', () => {
      assert.throws(() => new FetchClient({ baseURL: '/api' }), TypeError);
    });
  });

  describe('request', () => {
    // 浏览器中的相对服务地址按页面 URL 解析，再追加请求路径
    const relativeCases = [
      { name: 'root_base_url', baseURL: '/', want: 'https://example.com/users' },
      { name: 'root_relative_base_url', baseURL: '/api', want: 'https://example.com/api/users' },
      { name: 'page_relative_base_url', baseURL: 'api', want: 'https://example.com/app/api/users' },
      { name: 'current_directory_base_url', baseURL: './api/', want: 'https://example.com/app/api/users' },
      { name: 'parent_directory_base_url', baseURL: '../api', want: 'https://example.com/api/users' },
      { name: 'protocol_relative_base_url', baseURL: '//api.example.com/v1', want: 'https://api.example.com/v1/users' },
      {
        name: 'absolute_base_url_with_location',
        baseURL: 'https://other.example.com/api',
        want: 'https://other.example.com/api/users',
      },
    ];
    for (const { name, baseURL, want } of relativeCases) {
      it(name, async context => {
        const descriptor = Object.getOwnPropertyDescriptor(globalThis, 'location');
        context.after(() => {
          if (descriptor === undefined) {
            Reflect.deleteProperty(globalThis, 'location');
          } else {
            Object.defineProperty(globalThis, 'location', descriptor);
          }
        });
        Object.defineProperty(globalThis, 'location', {
          configurable: true,
          value: { href: 'https://example.com/app/index.html?view=list#top' },
        });
        let calls = 0;
        const client = new FetchClient({
          baseURL,
          fetch: input => {
            calls++;
            assert.ok(input instanceof URL);
            assert.equal(input.href, want);
            return Promise.resolve(new Response(null, { status: 204 }));
          },
        });

        await client.request({ method: 'GET', path: '/users' });

        assert.equal(calls, 1);
      });
    }

    // 路径前缀不受首尾斜杠影响，路径已有的查询参数与新增参数共存
    const urlCases: { name: string; baseURL: string | URL; path: string; query?: Request['query']; want: string }[] = [
      { name: 'origin', baseURL: 'https://example.com', path: '/users', want: 'https://example.com/users' },
      {
        name: 'prefix_without_slash',
        baseURL: 'https://example.com/api',
        path: '/users',
        want: 'https://example.com/api/users',
      },
      {
        name: 'prefix_with_slash',
        baseURL: 'https://example.com/api/',
        path: 'users',
        want: 'https://example.com/api/users',
      },
      {
        name: 'multiple_leading_slashes',
        baseURL: 'https://example.com/api',
        path: '//users',
        want: 'https://example.com/api/users',
      },
      {
        name: 'url_object',
        baseURL: new URL('https://example.com/api'),
        path: '/users',
        want: 'https://example.com/api/users',
      },
      { name: 'empty_path', baseURL: 'https://example.com/api', path: '', want: 'https://example.com/api/' },
      {
        name: 'scalar_query',
        baseURL: 'https://example.com',
        path: '/users',
        query: { name: 'Alice Smith', active: false, age: 0 },
        want: 'https://example.com/users?name=Alice+Smith&active=false&age=0',
      },
      {
        name: 'array_query',
        baseURL: 'https://example.com',
        path: '/users',
        query: { tags: ['a', 'b'], flags: [true, 1] },
        want: 'https://example.com/users?tags=a&tags=b&flags=true&flags=1',
      },
      {
        name: 'empty_query_values',
        baseURL: 'https://example.com',
        path: '/users',
        query: { omitted: undefined, empty: [], text: '' },
        want: 'https://example.com/users?text=',
      },
      {
        name: 'existing_query',
        baseURL: 'https://example.com',
        path: '/users?tags=old&keep=yes',
        query: { tags: ['a', 'b'] },
        want: 'https://example.com/users?tags=old&keep=yes&tags=a&tags=b',
      },
      {
        name: 'escaped_query',
        baseURL: 'https://example.com',
        path: '/users',
        query: { 'a&b': '你好 +/' },
        want: 'https://example.com/users?a%26b=%E4%BD%A0%E5%A5%BD+%2B%2F',
      },
    ];
    for (const { name, baseURL, path, query, want } of urlCases) {
      it(name, async () => {
        const originalBaseURL = String(baseURL);
        let calls = 0;
        const client = new FetchClient({
          baseURL,
          fetch: (input, init) => {
            calls++;
            assert.ok(input instanceof URL);
            assert.equal(input.href, want);
            assert.equal(init?.method, 'GET');
            return Promise.resolve(new Response(null, { status: 204 }));
          },
        });
        await client.request({ method: 'GET', path, query });
        assert.equal(calls, 1);
        assert.equal(String(baseURL), originalBaseURL);
      });
    }

    // 请求头按名称忽略大小写覆盖，数组交由 Headers 合并
    const headerCases: {
      name: string;
      defaults?: HeadersInit;
      headers?: Request['headers'];
      want: Record<string, string>;
    }[] = [
      { name: 'no_headers', want: {} },
      { name: 'default_headers', defaults: { Authorization: 'Bearer token' }, want: { authorization: 'Bearer token' } },
      {
        name: 'request_headers',
        headers: { 'Content-Type': 'application/octet-stream' },
        want: { 'content-type': 'application/octet-stream' },
      },
      {
        name: 'override_default',
        defaults: { 'X-Value': 'old', 'X-Keep': 'kept' },
        headers: { 'x-value': 'new' },
        want: { 'x-keep': 'kept', 'x-value': 'new' },
      },
      {
        name: 'array_header',
        defaults: { 'X-Value': 'old' },
        headers: { 'x-value': ['a', 'b'] },
        want: { 'x-value': 'a, b' },
      },
      { name: 'empty_array_header', defaults: { 'X-Value': 'old' }, headers: { 'x-value': [] }, want: {} },
    ];
    for (const { name, defaults, headers, want } of headerCases) {
      it(name, async () => {
        let calls = 0;
        const client = new FetchClient({
          baseURL: 'https://example.com',
          headers: defaults,
          fetch: (input, init) => {
            calls++;
            assert.deepEqual(Object.fromEntries(new Headers(init?.headers)), want);
            return Promise.resolve(new Response(null, { status: 204 }));
          },
        });
        await client.request({ method: 'GET', path: '/', headers });
        assert.equal(calls, 1);
      });
    }

    it('isolates_default_headers_and_query_between_requests', async () => {
      const headers = new Headers({ 'X-Value': 'original' });
      const baseURL = new URL('https://example.com/api');
      const urls: string[] = [];
      const values: (string | null)[] = [];
      const client = new FetchClient({
        baseURL,
        headers,
        fetch: (input, init) => {
          assert.ok(input instanceof URL);
          urls.push(input.href);
          values.push(new Headers(init?.headers).get('X-Value'));
          return Promise.resolve(new Response(null, { status: 204 }));
        },
      });
      headers.set('X-Value', 'mutated');
      baseURL.pathname = '/changed';
      const request: Request = {
        method: 'GET',
        path: '/users',
        query: { tags: ['a', 'b'] },
        headers: { 'X-Value': 'override' },
      };
      await client.request(request);
      assert.equal(urls.length, 1);
      await client.request({ method: 'GET', path: '/users' });
      assert.equal(urls.length, 2);
      assert.deepEqual(urls, ['https://example.com/api/users?tags=a&tags=b', 'https://example.com/api/users']);
      assert.deepEqual(values, ['override', 'original']);
      assert.deepEqual(request.query, { tags: ['a', 'b'] });
      assert.deepEqual(request.headers, { 'X-Value': 'override' });
    });

    // 仅传输实际字节，不执行 JSON 编码，也不隐式添加媒体类型
    for (const method of ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'] as const) {
      it(`forwards_method_body_and_signal_${method}`, async () => {
        const body = method === 'GET' ? undefined : new Uint8Array([9, 0, 255, 128, 9]).subarray(1, 4);
        const controller = new AbortController();
        let calls = 0;
        const client = new FetchClient({
          baseURL: 'https://example.com',
          fetch: (input, init) => {
            calls++;
            assert.equal(init?.method, method);
            assert.deepEqual(init?.body, body);
            assert.equal(init?.signal, controller.signal);
            assert.deepEqual(Object.fromEntries(new Headers(init?.headers)), {});
            return Promise.resolve(new Response(null, { status: 204 }));
          },
        });
        await client.request({ method, path: '/', body, signal: controller.signal });
        assert.equal(calls, 1);
        assert.deepEqual(body, method === 'GET' ? undefined : new Uint8Array([0, 255, 128]));
      });
    }

    it('sends_empty_body', async () => {
      let calls = 0;
      const client = new FetchClient({
        baseURL: 'https://example.com',
        fetch: (input, init) => {
          calls++;
          assert.deepEqual(init?.body, new Uint8Array());
          return Promise.resolve(new Response(null, { status: 204 }));
        },
      });
      await client.request({ method: 'POST', path: '/', body: new Uint8Array() });
      assert.equal(calls, 1);
    });

    // HTTP 错误状态由 resolver 处理，客户端保留状态码、响应头和原始字节
    for (const status of [200, 201, 204, 302, 400, 404, 500]) {
      it(`returns_response_${status}`, async () => {
        const body = status === 204 ? null : new Uint8Array([0, 255, 128]);
        let calls = 0;
        const client = new FetchClient({
          baseURL: 'https://example.com',
          fetch: () => {
            calls++;
            return Promise.resolve(new Response(body, { status, headers: { 'X-Value': 'value' } }));
          },
        });
        const response = await client.request({ method: 'GET', path: '/' });
        assert.equal(calls, 1);
        assert.equal(response.status, status);
        assert.deepEqual(response.headers, { 'x-value': 'value' });
        assert.deepEqual(response.body, body ?? new Uint8Array());
      });
    }

    it('returns_fetch_header_entries_without_special_handling', async () => {
      const cookies = ['session=abc; Expires=Wed, 21 Oct 2030 07:28:00 GMT', 'theme=dark; Path=/'];
      const headers = new Headers();
      for (const cookie of cookies) {
        headers.append('Set-Cookie', cookie);
      }
      headers.append('X-Value', 'a');
      headers.append('X-Value', 'b');
      const fetchResponse = new Response(null, { headers });
      let calls = 0;
      const client = new FetchClient({
        baseURL: 'https://example.com',
        fetch: () => {
          calls++;
          return Promise.resolve(fetchResponse);
        },
      });
      const response = await client.request({ method: 'GET', path: '/' });
      assert.equal(calls, 1);
      assert.deepEqual(response.headers, Object.fromEntries(fetchResponse.headers));
    });

    it('propagates_transport_error', async () => {
      const error = new TypeError('Network failure');
      let calls = 0;
      const client = new FetchClient({
        baseURL: 'https://example.com',
        fetch: () => {
          calls++;
          return Promise.reject(error);
        },
      });
      await assert.rejects(client.request({ method: 'GET', path: '/' }), (caught: unknown) => caught === error);
      assert.equal(calls, 1);
    });

    it('propagates_body_read_error', async () => {
      const error = new Error('Response stream failed');
      let calls = 0;
      const stream = new ReadableStream<Uint8Array>({
        start: controller => {
          controller.error(error);
        },
      });
      const client = new FetchClient({
        baseURL: 'https://example.com',
        fetch: () => {
          calls++;
          return Promise.resolve(new Response(stream));
        },
      });
      await assert.rejects(client.request({ method: 'GET', path: '/' }), (caught: unknown) => caught === error);
      assert.equal(calls, 1);
    });

    it('propagates_abort_reason', async () => {
      const controller = new AbortController();
      const error = new Error('Request cancelled');
      controller.abort(error);
      let calls = 0;
      const client = new FetchClient({
        baseURL: 'https://example.com',
        fetch: (input, init) => {
          calls++;
          assert.equal(init?.signal, controller.signal);
          init?.signal?.throwIfAborted();
          return Promise.resolve(new Response(null));
        },
      });
      await assert.rejects(
        client.request({ method: 'GET', path: '/', signal: controller.signal }),
        (caught: unknown) => caught === error,
      );
      assert.equal(calls, 1);
    });

    it('uses_global_fetch_by_default', async context => {
      const fetchMock = context.mock.method(globalThis, 'fetch', () =>
        Promise.resolve(new Response(null, { status: 204 })),
      );
      const client = new FetchClient({ baseURL: 'https://example.com' });
      const response = await client.request({ method: 'GET', path: '/' });
      assert.equal(response.status, 204);
      assert.equal(fetchMock.mock.callCount(), 1);
      const input = fetchMock.mock.calls[0].arguments[0];
      assert.ok(input instanceof URL);
      assert.equal(input.href, 'https://example.com/');
    });
  });
});
