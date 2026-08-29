# API 接口约定

## 版本与路径

- 所有业务接口路径以 `/api` 开头。
- 当前接口版本为 `v1`。
- 完整接口前缀为 `/api/v1`。
- 破坏性变更必须升级版本，例如 `/api/v2`。
- 临时 OpenAPI 地址为 `http://localhost:8080/swagger/doc.json`。
- 前端通过 `NUXT_PUBLIC_API_BASE` 配置 API Origin，该配置不包含 `/api/v1`。

示例：

```text
GET    /api/v1/articles
GET    /api/v1/articles/{id}
POST   /api/v1/articles
PATCH  /api/v1/articles/{id}
DELETE /api/v1/articles/{id}
```

## 认证接口

```text
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
```

登录请求体：

```json
{
  "email": "user@example.com",
  "password": "password"
}
```

登录和刷新接口返回当前用户以及短期有效的 Access Token：

```json
{
  "code": 0,
  "data": {
    "token": "access-token",
    "user": {
      "id": 1,
      "username": "example",
      "email": "user@example.com",
      "avatar": ""
    }
  },
  "msg": "success"
}
```

Access Token 只保存在 Pinia 内存中，请求时通过以下 Header 发送：

```http
Authorization: Bearer <token>
```

Refresh Token 不得出现在 JSON 响应中。后端必须通过 Cookie 写入，并至少设置：

```http
HttpOnly; Secure; SameSite=Lax; Path=/api/v1/auth
```

刷新接口必须轮换 Refresh Token，并返回新的 Access Token 和用户信息。退出接口必须撤销刷新会话并清除 Cookie。

## 响应结构

包含数据的成功响应：

```json
{
  "code": 0,
  "data": {},
  "msg": "success"
}
```

不包含数据的成功响应：

```json
{
  "code": 0,
  "msg": "success"
}
```

错误响应：

```json
{
  "code": 205,
  "msg": "用户登录失效"
}
```

HTTP 状态码必须与错误类型一致。登录失效返回 `401`，权限不足返回 `403`，参数错误返回 `400`，资源不存在返回 `404`，未预期的服务端错误返回 `500`。

## 命名规则

- 多单词 JSON 字段使用 `snake_case`。
- 日期时间使用 RFC 3339 字符串，例如 `2026-08-30T12:00:00Z`。
- 资源路径使用复数名词。
- OpenAPI `operationId` 必须稳定且唯一。
- OpenAPI `tag` 按业务领域划分生成的 Service。

## OpenAPI 代码生成

Orval 读取 OpenAPI 文档，并将生成代码写入 `app/api/generated/`：

```bash
pnpm api:generate
```

禁止手动修改生成文件。需要变更接口时，应修改后端 OpenAPI 定义并重新生成。生成的请求通过 `app/api/mutator.ts` 接入现有 `$api`，因此会复用 Token 注入、`401` 刷新锁和请求重试逻辑。

认证接口需要按场景传入 `skipAuth: true` 和 `skipRefresh: true`。登录、刷新和退出仍优先使用手写的 `useAuth` 与认证 Service。

## 跨域要求

前端和 API 使用不同 Origin 时，前端请求会携带凭据。后端必须返回明确的允许来源和凭据 Header：

```http
Access-Control-Allow-Origin: https://frontend.example.com
Access-Control-Allow-Credentials: true
```

使用 Cookie 时，`Access-Control-Allow-Origin` 不得设置为 `*`。
