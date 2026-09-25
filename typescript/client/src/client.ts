/**
 * Scaffold 支持的 HTTP 请求方法
 */
export type Method = 'DELETE' | 'GET' | 'PATCH' | 'POST' | 'PUT';

/**
 * HTTP 头支持的值
 */
export type HeaderValue = string | readonly string[];

/**
 * URL 查询参数支持的标量值
 */
export type QueryValue = boolean | number | string;

/**
 * 发送 HTTP 请求的客户端
 */
export interface Client {
  /**
   * 发送 HTTP 请求
   *
   * 收到 HTTP 响应时无论状态码均返回 Response
   * 未收到响应时抛出传输错误
   */
  request(request: Request): Promise<Response>;
}

/**
 * 客户端发送的 HTTP 请求
 */
export interface Request {
  /** HTTP 请求方法 */
  method: Method;
  /** 请求路径 */
  path: string;
  /** URL 查询参数 */
  query?: Record<string, QueryValue | QueryValue[] | undefined>;
  /** HTTP 请求头，重复字段使用字符串数组表示 */
  headers?: Record<string, HeaderValue>;
  /** 原始请求体字节 */
  body?: Uint8Array;
  /** 用于取消请求的信号 */
  signal?: AbortSignal;
}

/**
 * 客户端接收的 HTTP 响应
 */
export interface Response {
  /** HTTP 响应状态码 */
  status: number;
  /** HTTP 响应头，重复字段使用字符串数组表示 */
  headers: Readonly<Record<string, HeaderValue>>;
  /** 原始响应体字节 */
  body: Uint8Array;
}

/**
 * HTTP 响应状态表示请求失败时抛出的错误
 */
export class HTTPError extends Error {
  readonly data: unknown;
  readonly response: Response;

  constructor(response: Response, data: unknown) {
    super(`HTTP request failed with status ${response.status}`);
    this.name = 'HTTPError';
    this.data = data;
    this.response = response;
  }
}
