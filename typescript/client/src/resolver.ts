import { HTTPError, type Response } from './client.ts';

/**
 * 将 HTTP 响应解析为业务数据，解析失败或响应不满足要求时可抛出错误
 */
export interface ResponseResolver {
  resolve<T>(response: Response): T | undefined;
}

/**
 * 将 UTF-8 JSON 响应体解析为业务数据
 */
export class JsonResolver implements ResponseResolver {
  resolve<T>(response: Response): T | undefined {
    const text = new TextDecoder().decode(response.body);

    if (response.status < 200 || response.status >= 300) {
      let data: unknown;
      try {
        data = text === '' ? undefined : (JSON.parse(text) as unknown);
      } catch {
        data = text;
      }
      throw new HTTPError(response, data);
    }

    if (text === '') {
      return undefined;
    }

    return JSON.parse(text) as T;
  }
}
