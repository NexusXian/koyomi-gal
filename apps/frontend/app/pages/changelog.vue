<script setup lang="ts">
useSeoMeta({ title: '更新日志 - Koyomi' })

interface ChangelogEntry {
  date: string
  version: string
  title: string
  items: { type: 'new' | 'improve' | 'fix'; text: string }[]
}

const entries: ChangelogEntry[] = [
  {
    date: '2026-09-03',
    version: 'v0.4.1',
    title: '用户身份展示与管理端优化',
    items: [
      { type: 'new', text: '管理端用户列表与详情展示用户身份标签，新建用户即时回显默认身份' },
      { type: 'improve', text: '新用户 ID 从 1001 起始' },
      { type: 'fix', text: '修复反馈处理角色缺失与顶部栏图标显示' },
      { type: 'fix', text: '调整管理端侧边管理栏宽度' }
    ]
  },
  {
    date: '2026-09-02',
    version: 'v0.4.0',
    title: '背景预设管理、站点页脚与反馈系统',
    items: [
      { type: 'new', text: '新增站点页脚与更新日志、意见反馈、用户协议、隐私政策、内容规范、版权投诉页面' },
      { type: 'new', text: '新增意见反馈与版权投诉提交（匿名、IP 限流），管理端可查看与处理' },
      { type: 'new', text: '背景预设改为后台管理：超级管理员可增删改查预设图片，用户侧动态拉取' },
      { type: 'improve', text: '验证码邮件改为 HTML 模板，Banner 图片改用 R2 统一域名' }
    ]
  },
  {
    date: '2026-08-31',
    version: 'v0.3.0',
    title: '通知、文章与图片资源',
    items: [
      { type: 'new', text: '站内通知系统：互动与审核消息、未读计数' },
      { type: 'new', text: '资讯文章模块与管理端发布流程' },
      { type: 'new', text: '图片直传 R2（预签名上传），头像、帖子、资源封面统一管理' }
    ]
  },
  {
    date: '2026-08-24',
    version: 'v0.2.0',
    title: '个性化背景与偏好同步',
    items: [
      { type: 'new', text: '用户个性化背景：预置图片、自定义上传、透明度与显示模式' },
      { type: 'new', text: '背景偏好云端同步，登录后多端一致' }
    ]
  },
  {
    date: '2026-08-18',
    version: 'v0.1.0',
    title: '站点初版上线',
    items: [
      { type: 'new', text: 'Galgame 目录、资源索引与评分收藏' },
      { type: 'new', text: '社区帖子与评论' },
      { type: 'new', text: '注册登录、邮箱验证码、RBAC 权限体系' }
    ]
  }
]

const typeLabels: Record<ChangelogEntry['items'][number]['type'], string> = {
  new: '新增',
  improve: '改进',
  fix: '修复'
}
</script>

<template>
  <AppPageContainer
    title="更新日志"
    description="记录站点每个版本的功能变化。"
  >
    <div class="changelog-list">
      <article v-for="entry in entries" :key="entry.version" class="changelog-entry">
        <header class="entry-header">
          <span class="entry-version">{{ entry.version }}</span>
          <h2 class="entry-title">{{ entry.title }}</h2>
          <time class="entry-date" :datetime="entry.date">{{ entry.date }}</time>
        </header>
        <ul class="entry-items">
          <li v-for="(item, index) in entry.items" :key="index">
            <span class="item-tag" :class="`item-${item.type}`">{{ typeLabels[item.type] }}</span>
            {{ item.text }}
          </li>
        </ul>
      </article>
    </div>
  </AppPageContainer>
</template>

<style scoped>
.changelog-list {
  max-width: 820px;
  display: flex;
  flex-direction: column;
  gap: 22px;
}

.changelog-entry {
  padding: 20px 22px;
  border: 1px solid var(--app-glass-border);
  border-radius: var(--radius-kun-md, 12px);
  background: var(--color-content2, transparent);
}

.entry-header {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 12px;
}

.entry-version {
  color: var(--color-primary);
  font-weight: 700;
  font-size: 15px;
}

.entry-title {
  margin: 0;
  font-size: 15px;
  font-weight: 650;
  color: var(--color-foreground);
}

.entry-date {
  margin-left: auto;
  color: var(--color-default-500, inherit);
  font-size: 13px;
}

.entry-items {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: var(--color-foreground);
  font-size: 14px;
  line-height: 1.8;
}

.item-tag {
  display: inline-block;
  margin-right: 8px;
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
}

.item-new {
  background: color-mix(in srgb, var(--color-primary) 14%, transparent);
  color: var(--color-primary);
}

.item-improve {
  background: color-mix(in srgb, #38bdf8 14%, transparent);
  color: #0284c7;
}

.item-fix {
  background: color-mix(in srgb, #f87171 14%, transparent);
  color: #dc2626;
}
</style>
