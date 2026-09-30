import assert from 'node:assert/strict';
import { Buffer } from 'node:buffer';
import { describe, it } from 'node:test';

import { AxiosClient, type AxiosInstance, type Request } from '../src/index.ts';

describe('AxiosClient', () => {
  describe('request', () => {
    // 地址与默认配置由注入的实例负责，客户端只传递请求路径
    for (const method of ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'] as const) {
      it(`forwards_request_${method}`, async () => {
        const controller = new AbortController();
        const body = method === 'GET' ? undefined : new Uint8Array([9, 0, 255, 128, 9]).subarray(1, 4);
        let calls = 0;
        const axios: AxiosInstance = {
          request(config) {
            calls++;
            assert.equal(this, axios);
            assert.equal(config.url, '/users?existing=value');
            assert.equal(config.method, method);
            assert.equal(config.signal, controller.signal);
            assert.deepEqual(config.data === undefined ? undefined : new Uint8Array(config.data), body);
            assert.equal(config.responseType, 'arraybuffer');
            assert.equal('baseURL' in config, false);
            const data = new Uint8Array([255, 0]).buffer;
            return Promise.resolve({ status: 200, headers: {}, data });
          },
        };
        const client = new AxiosClient(axios);

        const response = await client.request({
          method,
          path: '/users?existing=value',
          body,
          signal: controller.signal,
        });

        assert.equal(calls, 1);
        assert.deepEqual(response.body, new Uint8Array([255, 0]));
        assert.deepEqual(body, method === 'GET' ? undefined : new Uint8Array([0, 255, 128]));
      });
    }

    const queryCases: { name: string; query?: Request['query']; want: string }[] = [
      { name: 'missing_query', want: '' },
      { name: 'empty_query', query: {}, want: '' },
      {
        name: 'scalar_query',
        query: { name: 'Alice Smith', age: 0, active: false },
        want: 'name=Alice+Smith&age=0&active=false',
      },
      { name: 'array_query', query: { tags: ['a', 'b'], flags: [true, 1] }, want: 'tags=a&tags=b&flags=true&flags=1' },
      { name: 'empty_query_values', query: { missing: undefined, empty: [], text: '' }, want: 'text=' },
      { name: 'escaped_query', query: { 'a&b': '你好 +/' }, want: 'a%26b=%E4%BD%A0%E5%A5%BD+%2B%2F' },
    ];
    for (const { name, query, want } of queryCases) {
      it(name, async () => {
        let calls = 0;
        const client = new AxiosClient({
          request: config => {
            calls++;
            assert.ok(config.params instanceof URLSearchParams);
            assert.equal(config.params.toString(), want);
            return Promise.resolve({ status: 204, headers: {}, data: new ArrayBuffer(0) });
          },
        });

        await client.request({ method: 'GET', path: '/', query });

        assert.equal(calls, 1);
      });
    }

    // 请求头不做媒体类型推断，也不拆分或合并重复值
    const headerCases: { name: string; headers?: Request['headers']; want: Record<string, string | string[]> }[] = [
      { name: 'missing_headers', want: {} },
      {
        name: 'string_header',
        headers: { 'Content-Type': 'application/octet-stream' },
        want: { 'Content-Type': 'application/octet-stream' },
      },
      { name: 'array_header', headers: { 'X-Value': ['a', 'b'] }, want: { 'X-Value': ['a', 'b'] } },
      { name: 'empty_array_header', headers: { 'X-Value': [] }, want: { 'X-Value': [] } },
    ];
    for (const { name, headers, want } of headerCases) {
      it(name, async () => {
        let calls = 0;
        const client = new AxiosClient({
          request: config => {
            calls++;
            assert.deepEqual(config.headers, want);
            assert.notEqual(config.headers, headers);
            return Promise.resolve({ status: 204, headers: {}, data: new ArrayBuffer(0) });
          },
        });

        await client.request({ method: 'GET', path: '/', headers });

        assert.equal(calls, 1);
      });
    }

    it('isolates_request_data_from_instance_mutation', async () => {
      const request: Request = {
        method: 'POST',
        path: '/',
        query: { tags: ['a', 'b'] },
        headers: { 'X-Value': ['a', 'b'] },
        body: new Uint8Array([1, 2]),
      };
      let calls = 0;
      const client = new AxiosClient({
        request: config => {
          calls++;
          config.params.append('tags', 'changed');
          const values = config.headers['X-Value'];
          assert.ok(Array.isArray(values));
          values.push('changed');
          assert.ok(config.data instanceof ArrayBuffer);
          new Uint8Array(config.data).fill(0);
          return Promise.resolve({ status: 204, headers: {}, data: new ArrayBuffer(0) });
        },
      });

      await client.request(request);

      assert.equal(calls, 1);
      assert.deepEqual(request.query, { tags: ['a', 'b'] });
      assert.deepEqual(request.headers, { 'X-Value': ['a', 'b'] });
      assert.deepEqual(request.body, new Uint8Array([1, 2]));
    });

    it('sends_empty_body', async () => {
      let calls = 0;
      const client = new AxiosClient({
        request: config => {
          calls++;
          assert.ok(config.data instanceof ArrayBuffer);
          assert.equal(config.data.byteLength, 0);
          return Promise.resolve({ status: 204, headers: {}, data: new ArrayBuffer(0) });
        },
      });

      await client.request({ method: 'POST', path: '/', body: new Uint8Array() });

      assert.equal(calls, 1);
    });

    // 所有 HTTP 状态都返回给 resolver，响应头直接保留 Axios 提供的值
    for (const status of [200, 201, 204, 302, 400, 404, 500]) {
      it(`returns_response_${status}`, async () => {
        const headers = { 'x-value': 'a, b', 'set-cookie': ['a=1', 'b=2'] };
        const data = status === 204 ? new ArrayBuffer(0) : new Uint8Array([0, 255]).buffer;
        let calls = 0;
        const client = new AxiosClient({
          request: config => {
            calls++;
            assert.equal(config.validateStatus(status), true);
            return Promise.resolve({ status, headers, data });
          },
        });

        const response = await client.request({ method: 'GET', path: '/' });

        assert.equal(calls, 1);
        assert.equal(response.status, status);
        assert.equal(response.headers, headers);
        assert.deepEqual(response.body, new Uint8Array(data));
      });
    }

    const bodyCases = [
      { name: 'array_buffer', data: new Uint8Array([0, 255, 128]).buffer },
      { name: 'byte_view', data: new Uint8Array([9, 0, 255, 128, 9]).subarray(1, 4) },
      { name: 'node_buffer', data: Buffer.from([9, 0, 255, 128, 9]).subarray(1, 4) },
    ];
    for (const { name, data } of bodyCases) {
      it(name, async () => {
        let calls = 0;
        const client = new AxiosClient({
          request: () => {
            calls++;
            return Promise.resolve({ status: 200, headers: {}, data });
          },
        });

        const response = await client.request({ method: 'GET', path: '/' });

        assert.equal(calls, 1);
        assert.deepEqual(response.body, new Uint8Array([0, 255, 128]));
      });
    }

    for (const name of ['network_error', 'timeout_error', 'cancelled_error']) {
      it(name, async () => {
        const error = new Error(name);
        let calls = 0;
        const client = new AxiosClient({
          request: () => {
            calls++;
            return Promise.reject(error);
          },
        });

        await assert.rejects(client.request({ method: 'GET', path: '/' }), (caught: unknown) => caught === error);

        assert.equal(calls, 1);
      });
    }
  });
});
