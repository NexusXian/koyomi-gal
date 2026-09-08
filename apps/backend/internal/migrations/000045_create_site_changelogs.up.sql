CREATE TABLE site_changelogs (
    id BIGSERIAL PRIMARY KEY,
    version VARCHAR(32) NOT NULL CHECK (BTRIM(version) <> ''),
    title VARCHAR(255) NOT NULL CHECK (BTRIM(title) <> ''),
    items JSONB NOT NULL CHECK (JSONB_TYPEOF(items) = 'array' AND JSONB_ARRAY_LENGTH(items) > 0),
    published_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_site_changelogs_version UNIQUE (version)
);

CREATE INDEX idx_site_changelogs_published_at ON site_changelogs (published_at DESC, id DESC);

-- Preserve the changelog history previously hardcoded in the web frontend.
INSERT INTO site_changelogs (version, title, items, published_at) VALUES
('v0.4.1', '用户身份展示与管理端优化', '[
  {"type": "new", "text": "管理端用户列表与详情展示用户身份标签，新建用户即时回显默认身份"},
  {"type": "improve", "text": "新用户 ID 从 1001 起始"},
  {"type": "fix", "text": "修复反馈处理角色缺失与顶部栏图标显示"},
  {"type": "fix", "text": "调整管理端侧边管理栏宽度"}
]'::jsonb, '2026-09-03 12:00:00+00'),
('v0.4.0', '背景预设管理、站点页脚与反馈系统', '[
  {"type": "new", "text": "新增站点页脚与更新日志、意见反馈、用户协议、隐私政策、内容规范、版权投诉页面"},
  {"type": "new", "text": "新增意见反馈与版权投诉提交（匿名、IP 限流），管理端可查看与处理"},
  {"type": "new", "text": "背景预设改为后台管理：超级管理员可增删改查预设图片，用户侧动态拉取"},
  {"type": "improve", "text": "验证码邮件改为 HTML 模板，Banner 图片改用 R2 统一域名"}
]'::jsonb, '2026-09-02 12:00:00+00'),
('v0.3.0', '通知、文章与图片资源', '[
  {"type": "new", "text": "站内通知系统：互动与审核消息、未读计数"},
  {"type": "new", "text": "资讯文章模块与管理端发布流程"},
  {"type": "new", "text": "图片直传 R2（预签名上传），头像、帖子、资源封面统一管理"}
]'::jsonb, '2026-08-31 12:00:00+00'),
('v0.2.0', '个性化背景与偏好同步', '[
  {"type": "new", "text": "用户个性化背景：预置图片、自定义上传、透明度与显示模式"},
  {"type": "new", "text": "背景偏好云端同步，登录后多端一致"}
]'::jsonb, '2026-08-24 12:00:00+00'),
('v0.1.0', '站点初版上线', '[
  {"type": "new", "text": "Galgame 目录、资源索引与评分收藏"},
  {"type": "new", "text": "社区帖子与评论"},
  {"type": "new", "text": "注册登录、邮箱验证码、RBAC 权限体系"}
]'::jsonb, '2026-08-18 12:00:00+00');
