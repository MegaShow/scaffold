import type { Client, Request, Response } from './client.ts';

/**
 * Fetch 客户端配置
 */
export interface FetchClientOptions {
  /**
   * 服务地址，浏览器中支持相对于当前页面的地址，请求路径会追加到此地址的路径前缀下
   */
  baseURL: string | URL;
  /**
   * 自定义 Fetch 实现，默认使用全局 fetch
   */
  fetch?: typeof globalThis.fetch;
  /**
   * 默认请求头，同名请求头由单次请求覆盖
   */
  headers?: HeadersInit;
}

/**
 * 使用 Fetch API 发送原始 HTTP 请求
 */
export class FetchClient implements Client {
  readonly #baseURL: URL;
  readonly #fetch: typeof globalThis.fetch;
  readonly #headers: Headers;

  constructor(options: FetchClientOptions) {
    this.#baseURL = new URL(options.baseURL, globalThis.location?.href);
    if (!this.#baseURL.pathname.endsWith('/')) {
      this.#baseURL.pathname += '/';
    }
    this.#fetch = options.fetch ?? ((input, init) => globalThis.fetch(input, init));
    this.#headers = new Headers(options.headers);
  }

  async request(request: Request): Promise<Response> {
    // 去掉请求路径的前导斜杠，保留服务地址中的路径前缀
    const url = new URL(request.path.replace(/^\/+/, ''), this.#baseURL);
    // 追加查询参数，跳过 undefined，数组编码为同名的多个参数
    for (const [name, value] of Object.entries(request.query ?? {})) {
      if (value === undefined) {
        continue;
      }
      for (const item of Array.isArray(value) ? value : [value]) {
        url.searchParams.append(name, String(item));
      }
    }

    // 复制默认请求头后按名称覆盖，避免单次请求修改后续请求的默认值
    const headers = new Headers(this.#headers);
    for (const [name, value] of Object.entries(request.headers ?? {})) {
      headers.delete(name);
      for (const item of typeof value === 'string' ? [value] : value) {
        headers.append(name, item);
      }
    }

    // 发送原始请求体字节，并将取消信号传给 Fetch
    const response = await this.#fetch(url, {
      method: request.method,
      headers,
      body: request.body === undefined ? undefined : new Uint8Array(request.body),
      signal: request.signal,
    });
    // 返回 Fetch 提供的状态码、响应头和响应体字节，HTTP 错误状态交由 resolver 处理
    return {
      status: response.status,
      headers: Object.fromEntries(response.headers),
      body: new Uint8Array(await response.arrayBuffer()),
    };
  }
}
