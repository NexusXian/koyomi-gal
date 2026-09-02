# Koyomi Gal

[**English**](README.md) | [**简体中文**](README.zh-CN.md)

一个 Galgame 社区平台：作品目录、评分、资源、帖子、文章与后台管理，由两个独立应用组成，位于同一仓库。

## 仓库结构

```
koyomi-gal/
├── apps/
│   ├── frontend/   # Nuxt 4（Vue 3、TypeScript、Tailwind CSS 4、Pinia）
│   └── backend/    # Go 1.27（Gin、GORM/PostgreSQL、Redis、Asynq、Swagger）
└── .github/
```

没有根级 workspace 清单或任务运行器，所有命令请在对应应用目录下执行。

## 技术栈

### 前端（`apps/frontend`）

- Nuxt 4 / Vue 3 / TypeScript
- Tailwind CSS 4、`@kungal/ui` 组件库、Ant Design Vue
- Pinia 状态管理
- 通过 Orval 从后端 OpenAPI 规范生成的 API 客户端（`app/api/generated/`）

### 后端（`apps/backend`）

- Gin HTTP 框架，GORM + pgx，PostgreSQL
- Redis + Asynq（异步邮件 worker，独立进程）
- JWT 认证（access + refresh 双令牌），RBAC 权限中间件
- Cloudflare R2（S3 兼容）图片存储
- Swagger 文档位于 `/swagger/index.html`
- SQL 迁移嵌入二进制，服务启动时自动执行（golang-migrate）

## 快速开始

### 前置条件

- Go 1.27+
- Node.js 20+ 与 pnpm
- PostgreSQL 与 Redis
- SMTP 邮箱账号（验证邮件）
- Cloudflare R2 存储桶（图片资源）

### 后端

```bash
cd apps/backend
cp .env.development.example .env.development   # 填入真实配置
go run ./cmd/server   # HTTP API
go run ./cmd/worker   # 异步邮件 worker（发送验证邮件必须运行）
```

- 两个进程都会加载 `.env.<APP_ENV>`（默认 `APP_ENV=development`）；已存在的环境变量优先于 dotenv 值。
- 服务启动前会先连通 PostgreSQL 与 Redis，并在启动时执行嵌入的迁移。
- server 与 worker 必须使用相同的 Redis 数据库和 `VERIFICATION_SECRET`（标准 base64，解码后至少 32 字节）；`CHANGE_ME` 占位符会在启动时报错。

### 前端

```bash
cd apps/frontend
pnpm install --frozen-lockfile
cp .env.development.example .env.development   # 将 NUXT_PUBLIC_API_BASE 指向后端源
pnpm dev
```

- `NUXT_PUBLIC_API_BASE` 只填源（如 `http://localhost:8080`）；接口代码会自动拼接 `/api/v1`。
- 后端接口变更后重新生成 API 客户端：`pnpm api:generate`（需要后端运行并提供 `/swagger/doc.json`）。

## 常用命令

| 应用     | 命令                             | 用途                    |
| -------- | -------------------------------- | ----------------------- |
| Frontend | `pnpm dev`                       | 开发服务器              |
| Frontend | `pnpm build`                     | 生产构建                |
| Frontend | `pnpm exec nuxt typecheck`       | 类型检查                |
| Frontend | `pnpm api:generate`              | 通过 Orval 生成 API 客户端 |
| Backend  | `go run ./cmd/server`            | 运行 HTTP 服务          |
| Backend  | `go run ./cmd/worker`            | 运行邮件 worker         |
| Backend  | `go test ./...`                  | 运行测试                |
| Backend  | `go vet ./...`                   | 静态检查                |
| Backend  | `bash scripts/gen-swagger.sh`    | 重新生成 Swagger 文档    |

## API 约定

所有 API 响应遵循 `code` / `data` / `msg` 包装结构。API 路径、请求体、认证 Cookie、CORS 及 OpenAPI 定义详见后端 Swagger 文档（`/swagger/index.html`）。
