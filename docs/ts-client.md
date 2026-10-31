# TypeScript Client 使用文档

本文档介绍如何使用 Scaffold 生成和使用 TypeScript 客户端代码。

## 快速开始

### 声明 IDL 和生成客户端代码

创建一个 proto 文件，用于定义 Web 应用所支持的 HTTP 接口。

```proto
// user.proto
edition = "2024";

package com.example.user;

option go_package = "example.com/user";

import "scaffold/api.proto";

message GetUserRequest {
  string user_id = 1;
}

message GetUserResponse {
  string user_id = 1;
  string name = 2;
}

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (api.http) = {
      get: "/users/{user_id}"
    };
  }
}
```

运行 Scaffold 生成 TypeScript 代码。

```sh
protoc-gen-scaffold -lang=typescript -kind=client -format=prettier user.proto
```

与 Golang 代码生成不同，TypeScript 需要指定代码格式化工具，目前仅支持 prettier，请确保 prettier 可访问执行。如果需要指定 prettier 的地址，可通过 `-prettier=<your_prettier_path>` 参数来配置。

生成的 `user.d.ts` 包含请求和响应类型，`user.service.ts` 包含可调用的 `UserService`。

### 安装依赖

安装 Scaffold 的 TypeScript 客户端依赖。

```sh
pnpm add @icytown/scaffold-client
```

如果使用 Axios，还需要安装 Axios。

```sh
pnpm add axios
```

### 创建客户端并调用服务

Scaffold 支持实现 Client 接口的 HTTP 客户端，也提供了 Fetch 和 Axios 的适配实现。

```ts
import type { Request, Response } from '@icytown/scaffold-client';

export interface Client {
  request(request: Request): Promise<Response>;
}
```

Client 负责发送请求并返回原始响应。

#### Fetch

使用 FetchClient 接入运行环境提供的 Fetch API。

```ts
import { AdaptComposer, FetchClient, JsonResolver } from '@icytown/scaffold-client';

import { UserService } from './gen/user.service.ts';

const client = new FetchClient({ baseURL: 'http://localhost:3000' });
const composer = new AdaptComposer();
const resolver = new JsonResolver();
const service = new UserService(client, { composer, resolver });

const user = await service.getUser({ user_id: '1' });
```

上述调用发送 `GET /users/1`。成功响应示例如下。

```json
{ "user_id": "1", "name": "User 1" }
```

#### Axios

使用 AxiosClient 包装调用方创建的 Axios 实例。服务地址、超时和默认请求头等配置由 Axios 实例管理。

```ts
import { AdaptComposer, AxiosClient, JsonResolver } from '@icytown/scaffold-client';
import { create } from 'axios';

import { UserService } from './gen/user.service.ts';

const client = new AxiosClient(create({ baseURL: 'http://localhost:3000', timeout: 5000 }));
const composer = new AdaptComposer();
const resolver = new JsonResolver();
const service = new UserService(client, { composer, resolver });

const user = await service.getUser({ user_id: '1' });
```

## Scaffold Client

Scaffold 提供了 `@icytown/scaffold-client` 包，定义与具体 HTTP 实现解耦的 Client、RequestComposer、ResponseResolver 等接口抽象和部分实现。

在客户端包中，有以下概念。

- Client：发送 HTTP 请求并接收原始响应的抽象
- RequestComposer：将业务请求数据编码为查询参数或请求体的抽象
- ResponseResolver：检查 HTTP 响应并解析业务数据的抽象

生成的服务负责组装请求方法、路径和请求头，然后依次调用 composer、client 和 resolver。构造服务时必须显式传入这三个组件，生成代码不提供默认实现。

客户端包提供了以下常用实现，也可以自行实现对应接口。

| 接口             | 实现              | 用途                                       |
| ---------------- | ----------------- | ------------------------------------------ |
| Client           | FetchClient       | 使用 Fetch API 发送请求                    |
| Client           | AxiosClient       | 使用调用方提供的 Axios 实例发送请求        |
| RequestComposer  | AdaptComposer     | 根据请求方法和 Content-Type 选择组装器     |
| RequestComposer  | FormComposer      | GET 编码为查询参数，其他方法编码为表单请求体 |
| RequestComposer  | JsonComposer      | 编码为 JSON 请求体                         |
| RequestComposer  | MultipartComposer | 使用原生 FormData 编码普通字段和文件       |
| ResponseResolver | JsonResolver      | 检查状态码并解析 JSON 响应                 |

`new AdaptComposer()` 默认使用 FormComposer、JsonComposer 和 MultipartComposer。GET 请求使用表单组装器，其余请求根据 Content-Type 选择；不支持的媒体类型会抛出错误。也可以通过构造参数中的 `form`、`json`、`multipart` 单独替换对应实现。

## 常见用法

### 设置默认请求头

通过 FetchClient 的 headers 配置默认请求头。

```ts
const client = new FetchClient({
  baseURL: 'http://localhost:3000',
  headers: { Authorization: `Bearer ${token}` },
});
```

使用 Axios 时，将默认请求头配置到 Axios 实例。

```ts
const client = new AxiosClient(
  create({
    baseURL: 'http://localhost:3000',
    headers: { Authorization: `Bearer ${token}` },
  }),
);
```

### 处理 HTTP 错误

JsonResolver 在响应状态码不属于 2xx 时抛出 HTTPError。错误中的 response 字段保留状态码、响应头和原始响应体，data 是解析后的错误数据；如果错误响应不是有效 JSON，则保留为文本。

```ts
import { HTTPError } from '@icytown/scaffold-client';

try {
  const user = await service.getUser({ user_id: '1' });
} catch (error) {
  if (error instanceof HTTPError && error.response.status === 404) {
    // 处理用户不存在的情况
  } else {
    throw error;
  }
}
```

请求组装错误、传输错误及响应解析错误都会直接传递给调用方。

### 通用 Response 解包

如果服务端按照 Go Server 文档中的示例返回统一 JSON 结构，可以实现自定义 ResponseResolver，提取其中的 data。

```ts
import { JsonResolver } from '@icytown/scaffold-client';
import type { Response, ResponseResolver } from '@icytown/scaffold-client';

interface ResponseData<T> {
  code: number;
  message: string;
  data?: T;
}

class DataResolver implements ResponseResolver {
  readonly #json = new JsonResolver();

  resolve<T>(response: Response): T | undefined {
    return this.#json.resolve<ResponseData<T>>(response)?.data;
  }
}
```

这里复用 JsonResolver 检查 HTTP 状态码，非 2xx 响应仍抛出 HTTPError。如果服务端使用 HTTP 200 承载业务错误，需要在自定义 resolver 中另行检查 code。

将创建服务时使用的 resolver 替换为自定义实现即可。

```ts
const resolver = new DataResolver();
const service = new UserService(client, { composer, resolver });
```

### 发送表单请求

不包含文件类型字段的非 GET 方法默认设置 JSON 请求头。如果接口需要表单请求体，可以在自定义 composer 中修改 Content-Type 后调用 FormComposer。

```ts
import { FormComposer } from '@icytown/scaffold-client';
import type { Request, RequestComposer } from '@icytown/scaffold-client';

class FormRequestComposer implements RequestComposer {
  readonly #form = new FormComposer();

  compose(request: Request, value: unknown): void {
    if (request.method !== 'GET') {
      request.headers = {
        ...request.headers,
        'Content-Type': 'application/x-www-form-urlencoded',
      };
    }
    this.#form.compose(request, value);
  }
}

const composer = new FormRequestComposer();
const service = new UserService(client, { composer, resolver });
```

### 文件上传

通过 `api.File` 声明文件类型字段，使用 `repeated api.File` 声明文件数组，即可生成使用 multipart/form-data 的客户端方法。

```proto
import "scaffold/api.proto";

message UploadFileRequest {
  string name = 1;
  api.File file = 2;
  repeated api.File files = 3;
}

service FileService {
  rpc UploadFile(UploadFileRequest) returns (api.Empty) {
    option (api.http) = {
      post: "/files/upload"
    };
  }
}
```

生成的 TypeScript 类型中，`api.File` 映射为原生 `File`，`repeated api.File` 映射为 `File[]`。FetchClient 和 AxiosClient 都支持发送 FormData，发送时由底层生成包含 boundary 的 Content-Type，无需手动设置。
