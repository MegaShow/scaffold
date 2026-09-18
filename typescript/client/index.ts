export type HTTPMethod = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';

export type QueryValue = boolean | number | string;

export interface Client {
  request<T>(request: Request): Promise<Response<T>>;
}

export interface Request {
  method: HTTPMethod;
  path: string;
  query?: Readonly<Record<string, QueryValue | readonly QueryValue[] | undefined>>;
  headers?: Readonly<Record<string, string>>;
  body?: unknown;
  signal?: AbortSignal;
}

export interface Response<T> {
  status: number;
  headers: Readonly<Record<string, string>>;
  body: T;
}
