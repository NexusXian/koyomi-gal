# Koyomi Gal 后端待办与设计方案

## 实施原则

- 保持现有领域模块结构，不重构 `user`、`rbac`、`app`。
- 继续使用 Gin、GORM、PostgreSQL、Redis、现有 Auth、RBAC、Logger、Response、Error 和 Migration。
- SQL Migration 是数据库 Schema 的唯一来源，禁止使用 `AutoMigrate`。
- Repository 负责数据访问，Service 负责业务与事务，Handler 负责 HTTP，`app.go` 负责组装，`router.go` 负责路由。
- 用户 ID 只从认证 Middleware Context 获取，不允许客户端提交。
- 暂不引入缓存、搜索引擎、消息总线、微服务或通用 CRUD 框架。

## 第一阶段遗留事项

### Catalog 管理查询

- [x] 确认是否补充 `GET /api/v1/developers/:id`。（已实现）
- [x] 确认是否补充 `GET /api/v1/tags/:id`。（已实现）
- [x] 设计管理员读取 `pending/rejected/hidden` Galgame 的方式。公共列表和详情必须继续只返回 `published`。（已实现 `GET /api/v1/admin/galgames` 与 `GET /api/v1/admin/galgames/:id`，权限 `galgame:review`；公共接口语义未变）

建议增加独立的 RBAC 管理查询，避免改变公共接口语义：

```text
GET /api/v1/admin/galgames
GET /api/v1/admin/galgames/:id
```

权限建议：

```text
galgame:review
```

### Developer / Tag 删除策略

- [x] 确认是否需要 Developer 和 Tag 删除接口。（决策：暂不提供硬删除，保留现有数据）

当前不提供硬删除，避免破坏已使用的目录数据。若以后实现：

- Developer 被使用时返回冲突，或先迁移关联 Galgame；不要静默清空大量关联。
- Tag 被使用时默认返回冲突；如需删除，必须明确是否同时删除全部 `galgame_tags` 关联。
- 未确认保留审核记录前，不增加 `DeletedAt`。

### Rating 精度

- [x] 第二阶段评分上线前，将 `galgames.rating_average` 从 `NUMERIC(3,2)` 调整为 `NUMERIC(4,2)`。（已在 `000005` Migration 中完成）

原因：`NUMERIC(3,2)` 最大只能保存 `9.99`，无法保存满分平均值 `10.00`。

## 第二阶段：Galgame 用户关系

### Migration

- [x] 新增 `000005_create_galgame_user_relations.up.sql`。
- [x] 新增 `000005_create_galgame_user_relations.down.sql`。
- [x] 创建 `galgame_ratings`、`galgame_favorites`、`user_galgames`。
- [x] 为评分增加 `1 <= score <= 10` 数据库约束。
- [x] 为用户与 Galgame 关系建立唯一索引和真实外键。
- [x] 同一 Migration 中修正 `rating_average` 为 `NUMERIC(4,2)`。

### 代码结构

继续放在 `internal/galgame`：

```text
internal/galgame/model/rating.go
internal/galgame/model/favorite.go
internal/galgame/model/user_state.go
internal/galgame/dto/rating.go
internal/galgame/dto/user_state.go
internal/galgame/repository/user_relation_repository.go
internal/galgame/service/rating_service.go
internal/galgame/service/favorite_service.go
internal/galgame/service/user_state_service.go
internal/galgame/handler/user_relation_handler.go
```

> 已实现，并额外增加 `internal/galgame/service/user_relation_service.go` 聚合读取 `GET /galgames/:id/me`。

### API

- [x] `PUT /api/v1/galgames/:id/rating`
- [x] `DELETE /api/v1/galgames/:id/rating`
- [x] `POST /api/v1/galgames/:id/favorite`
- [x] `DELETE /api/v1/galgames/:id/favorite`
- [x] `PUT /api/v1/galgames/:id/state`
- [x] `DELETE /api/v1/galgames/:id/state`
- [x] `GET /api/v1/galgames/:id/me`

全部接口必须登录。用户 ID 从 `middleware.CurrentUserID` 获取。

### 事务设计

评分创建、修改、删除必须在一个事务中完成：

```text
写入或删除 galgame_ratings
→ SELECT AVG(score), COUNT(*)
→ 更新 galgames.rating_average 和 rating_count
```

收藏必须在一个事务中完成：

```text
新增或删除 galgame_favorites
→ 使用数据库原子表达式更新 favorite_count
```

删除收藏时使用 `GREATEST(favorite_count - 1, 0)`，禁止读取后在 Go 中递减。

游玩状态：

```text
1 wish
2 playing
3 completed
4 paused
5 dropped
```

Service 必须校验状态值和 `play_time_minutes >= 0`。

### 测试

- [x] 首次评分、修改评分、删除评分。
- [x] 评分平均值和数量重新计算。
- [x] 重复收藏幂等或返回明确冲突。
- [x] 收藏计数并发更新不丢失。
- [x] 游玩状态创建、修改、删除。
- [x] `GET /galgames/:id/me` 只返回当前用户关系。
- [x] 未登录请求返回 401。
- [x] 事务失败时关系和统计字段全部回滚。

（集成测试位于 `internal/galgame/service/user_relation_service_test.go` 与 `internal/galgame/handler/user_relation_handler_test.go`，使用 `RBAC_TEST_DSN` 独立 PostgreSQL。）

## 第三阶段：Resource 模块

### Migration

- [x] 新增 `000006_create_resources.up.sql`。
- [x] 新增 `000006_create_resources.down.sql`。
- [x] 创建 `resources`、`resource_links`。
- [x] 在举报需求确认后创建 `resource_reports`，不要提前实现复杂工作流。（已随 `000008_create_resource_reports` 实现：每用户每资源一次举报、管理员按 `resource_report:list` / `resource_report:handle` 查询与结单，处理仅改状态不改动资源本身）

### 模块结构

```text
internal/resource/
├── dto/
├── handler/
├── model/
├── repository/
└── service/
```

### 核心规则

- `resource_type` 使用常量：`other/game/patch/save/soundtrack/cg/guide`。
- `status` 使用常量：`pending/published/rejected/hidden`。
- 一个 Resource 拥有多个 `resource_links`。
- `uploader_id` 只能来自登录态。
- 公开查询只返回 `published`。
- 上传者可以修改、删除自己的资源。
- 管理其他用户资源必须拥有 `resource:update` 或 `resource:delete`。

### API

- [x] `GET /api/v1/galgames/:id/resources`
- [x] `GET /api/v1/resources/:id`
- [x] `POST /api/v1/resources`
- [x] `PUT /api/v1/resources/:id`
- [x] `DELETE /api/v1/resources/:id`

资源创建事务：

```text
创建 Resource
→ 批量创建 Resource Links
→ galgames.resource_count = resource_count + 1
```

删除资源时原子减少计数，且不能产生负数。（已实现：`GREATEST(resource_count - 1, 0)`）

### RBAC

- [x] Seed `resource:update`。
- [x] Seed `resource:delete`。
- [x] Seed `resource:review`。
- [x] 举报管理启用后 Seed `resource_report:list`、`resource_report:handle`。（已实现；另 Seed `galgame:review` 供管理端 Galgame 查询）

普通登录用户上传自己的资源不应要求管理员权限；所有权与管理员权限组合判断必须放在 Service。

> 已实现：`ResourceService.ensureCanManage` 先判上传者所有权，再回退 `resource:update` / `resource:delete` 权限；`resource_type` / `status` 以 SMALLINT 常量实现（0-6 / 0-3），与 Galgame 状态风格一致。集成测试位于 `internal/resource/service/resource_service_test.go` 与 `internal/resource/handler/resource_handler_test.go`。

### 举报设计

`resource_reports` 建议字段：

```text
id, resource_id, user_id, reason, description, status,
handled_by, handled_at, created_at, updated_at
```

原因枚举：

```text
invalid_link
wrong_password
corrupted
malware
wrong_version
duplicate
other
```

## 第四阶段：Community 模块

### Migration

- [x] 新增 `000007_create_community.up.sql`。
- [x] 新增 `000007_create_community.down.sql`。
- [x] 创建 `posts`、`comments`、`post_likes`、`comment_likes`、`post_favorites`。

### 模块结构

```text
internal/community/
├── dto/
├── handler/
├── model/
├── repository/
└── service/
```

不要拆成 `internal/post`、`internal/comment`、`internal/like`。

> 已实现：`post_service` / `comment_service` / `interaction_service` 与对应 handler 均在 `internal/community` 内。创建评论时客户端传 `parent_id` + `reply_to_comment_id`，服务端将回复目标作者写入 `reply_to_user_id`（客户端不提交用户 ID）；删除一级评论按子树大小原子扣减 `posts.comment_count`。API：`GET/POST /posts`、`GET/PUT/DELETE /posts/:id`、`GET/POST /posts/:id/comments`、`PUT/DELETE /comments/:id`、`POST/DELETE /posts/:id/like|favorite`、`POST/DELETE /comments/:id/like`。集成测试位于 `internal/community/service/community_service_test.go` 与 `internal/community/handler/community_handler_test.go`。

### 核心规则

- Post 的 `galgame_id` 可为空，支持普通社区帖子和 Galgame 讨论。
- Post 作者和 Comment 作者只能来自登录态。
- 评论只使用两层展示结构。
- `parent_id` 永远指向一级评论。
- 回复另一条回复时，通过 `reply_to_user_id` 标识被回复用户。
- Like 使用独立的 `post_likes` 和 `comment_likes`，不使用多态 Target 表。
- 作者可以管理自己的内容；管理他人内容需要审核权限。

### 事务与计数

- [x] 关联 Galgame 的 Post 创建时原子增加 `galgames.post_count`。
- [x] Comment 创建、删除时原子更新 `posts.comment_count`。
- [x] Like 和 Favorite 使用唯一索引防止重复，并原子更新对应计数。
- [x] 所有减法使用 `GREATEST(count - 1, 0)`。

### RBAC

- [x] Seed `post:moderate`。
- [x] Seed `comment:moderate`。

## 第五阶段：数据规模明确后再评估

- [ ] Character、People、Staff 及其 Galgame 关系。
- [ ] 内容贡献历史和 Galgame Wiki Revision。
- [ ] Community 与 Notification 集成。
- [ ] 实际性能数据证明有必要后，再增加 Redis 缓存。
- [ ] PostgreSQL 搜索无法满足需求后，再评估 Meilisearch。

搜索服务启用后必须保持：

```text
PostgreSQL = Source of Truth
Search Engine = Index
```

## 每阶段完成标准

每个阶段完成后必须执行：

```bash
gofmt
./scripts/gen-swagger.sh
go vet ./...
go test ./...
go build ./...
```

数据库集成测试必须使用独立 PostgreSQL 测试库，并覆盖：

- Schema Migration。
- 正常业务流程。
- 唯一约束和外键错误。
- 权限与所有权判断。
- 事务回滚。
- 并发计数。
- 公共查询不泄露未发布内容。
- 列表查询不存在 N+1。
