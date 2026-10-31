import type { Client, Request, Response } from './client.ts';

/**
 * Axios 客户端所需的实例类型，仅声明请求时使用的配置及响应字段
 */
export interface AxiosInstance {
  request(config: {
    url: string;
    method: Request['method'];
    params: URLSearchParams;
    headers: Record<string, string | string[]>;
    data?: ArrayBuffer | FormData;
    signal?: AbortSignal;
    responseType: 'arraybuffer';
    validateStatus: (status: number) => boolean;
  }): Promise<{
    status: number;
    headers: Response['headers'];
    data: ArrayBuffer | Uint8Array;
  }>;
}

/**
 * 使用调用方提供的 Axios 实例发送原始 HTTP 请求
 */
export class AxiosClient implements Client {
  readonly #axios: AxiosInstance;

  constructor(axios: AxiosInstance) {
    this.#axios = axios;
  }

  async request(request: Request): Promise<Response> {
    // 使用重复参数编码数组，避免 Axios 默认添加数组下标或方括号
    const params = new URLSearchParams();
    for (const [name, value] of Object.entries(request.query ?? {})) {
      if (value === undefined) {
        continue;
      }
      for (const item of Array.isArray(value) ? value : [value]) {
        params.append(name, String(item));
      }
    }

    // 复制请求头及数组，默认请求头由 Axios 实例负责合并
    const headers: Record<string, string | string[]> = {};
    for (const [name, value] of Object.entries(request.headers ?? {})) {
      headers[name] = typeof value === 'string' ? value : [...value];
    }

    if (request.body instanceof FormData) {
      for (const name of Object.keys(headers)) {
        if (name.toLowerCase() === 'content-type') {
          delete headers[name];
        }
      }
    }

    // 仅发送视图中的字节，以二进制接收响应，编解码交由 composer 和 resolver 处理
    const response = await this.#axios.request({
      url: request.path,
      method: request.method,
      params,
      headers,
      data:
        request.body instanceof FormData
          ? request.body
          : request.body === undefined
            ? undefined
            : new Uint8Array(request.body).buffer,
      signal: request.signal,
      responseType: 'arraybuffer',
      validateStatus: () => true,
    });

    // 浏览器返回 ArrayBuffer，Node.js 返回 Buffer，统一为 Uint8Array
    return {
      status: response.status,
      headers: response.headers,
      body: new Uint8Array(response.data),
    };
  }
}
