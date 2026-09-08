<script setup lang="ts">
import { createSiteChangelogService } from '~/services/siteChangelog'
import type { ChangelogItem } from '~/types/siteChangelog'

useSeoMeta({ title: '更新日志 - Koyomi' })

const changelogService = createSiteChangelogService(useNuxtApp().$api)

const { data, pending, error, refresh } = await useAsyncData(
  'site-changelog',
  () => changelogService.list()
)

const typeLabels: Record<ChangelogItem['type'], string> = {
  new: '新增',
  improve: '改进',
  fix: '修复'
}

function entryDate(value: string): string {
  return value.slice(0, 10)
}
</script>

<template>
  <AppPageContainer
    title="更新日志"
    description="记录站点每个版本的功能变化。"
  >
    <div v-if="pending" class="changelog-list">
      <KunSkeleton v-for="item in 3" :key="item" class="changelog-skeleton" />
    </div>

    <KunCard v-else-if="error" padding="lg">
      <div class="list-state">
        <p>{{ getApiErrorMessage(error, '更新日志加载失败') }}</p>
        <KunButton color="primary" @click="() => refresh()">重新加载</KunButton>
      </div>
    </KunCard>

    <KunCard v-else-if="!data?.length" padding="lg">
      <div class="list-state">
        <p>暂无更新日志</p>
      </div>
    </KunCard>

    <div v-else class="changelog-list">
      <article v-for="entry in data" :key="entry.id" class="changelog-entry">
        <header class="entry-header">
          <span class="entry-version">{{ entry.version }}</span>
          <h2 class="entry-title">{{ entry.title }}</h2>
          <time class="entry-date" :datetime="entry.publishedAt">{{ entryDate(entry.publishedAt) }}</time>
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

.changelog-skeleton {
  height: 160px;
}

.list-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  color: var(--color-foreground);
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
