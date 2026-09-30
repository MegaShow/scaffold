import type { Request } from './client.ts';

/**
 * 将业务请求数据写入 HTTP 请求，组装失败时可抛出错误
 */
export interface RequestComposer {
  compose(request: Request, value: unknown): void;
}

/**
 * 根据请求方法和 Content-Type 选择请求组装器，未指定时使用 FormComposer 和 JsonComposer
 */
export class AdaptComposer implements RequestComposer {
  readonly form: RequestComposer;
  readonly json: RequestComposer;

  constructor(config: { form?: RequestComposer; json?: RequestComposer } = {}) {
    this.form = config.form ?? new FormComposer();
    this.json = config.json ?? new JsonComposer();
  }

  compose(request: Request, value: unknown): void {
    const contentType = getHeader(request, 'Content-Type')?.split(';', 1)[0].trim().toLowerCase();

    if (request.method === 'GET' || contentType === 'application/x-www-form-urlencoded') {
      this.form.compose(request, value);
      return;
    }
    if (contentType === 'application/json') {
      this.json.compose(request, value);
      return;
    }
    throw new TypeError('Unsupported media type');
  }
}

/**
 * 将业务请求数据编码为 URL 查询参数或表单请求体
 */
export class FormComposer implements RequestComposer {
  compose(request: Request, value: unknown): void {
    const form = new URLSearchParams();
    appendFormValue(form, '', value);

    // GET 请求
    if (request.method === 'GET') {
      const query = { ...request.query };
      for (const [name, item] of form) {
        const current = query[name];
        query[name] = current === undefined ? item : Array.isArray(current) ? [...current, item] : [current, item];
      }
      request.query = query;
      return;
    }

    // 非 GET 请求
    request.body = new TextEncoder().encode(form.toString());
  }
}

/**
 * 将业务请求数据编码为 JSON 请求体
 */
export class JsonComposer implements RequestComposer {
  compose(request: Request, value: unknown): void {
    const text = JSON.stringify(value, (key, value): unknown =>
      value instanceof Uint8Array ? value.toBase64() : value,
    );
    if (text === undefined) {
      throw new TypeError('Unable to serialize request data as JSON');
    }
    request.body = new TextEncoder().encode(text);
  }
}

function appendFormValue(form: URLSearchParams, name: string, value: unknown) {
  if (value === null || value === undefined) {
    return;
  }
  if (typeof value === 'boolean' || typeof value === 'number' || typeof value === 'string') {
    if (name === '') {
      throw new TypeError('Form request data must be an object');
    }
    form.append(name, String(value));
    return;
  }
  if (value instanceof Uint8Array) {
    if (name === '') {
      throw new TypeError('Form request data must be an object');
    }
    form.append(name, new TextDecoder('utf-8', { fatal: true }).decode(value));
    return;
  }
  if (Array.isArray(value)) {
    value.forEach((item, index) => {
      const itemName =
        typeof item === 'object' && item !== null && !(item instanceof Uint8Array) ? `${name}[${index}]` : name;
      appendFormValue(form, itemName, item);
    });
    return;
  }
  if (typeof value === 'object') {
    for (const [key, item] of Object.entries(value)) {
      appendFormValue(form, name === '' ? key : `${name}.${key}`, item);
    }
    return;
  }
  throw new TypeError(`Unsupported form value for field ${name}`);
}

function getHeader(request: Request, name: string): string | undefined {
  const currentName = Object.keys(request.headers ?? {}).find(item => item.toLowerCase() === name.toLowerCase());
  if (currentName === undefined) {
    return undefined;
  }
  const value = request.headers?.[currentName];
  return typeof value === 'string' ? value : value?.[0];
}
