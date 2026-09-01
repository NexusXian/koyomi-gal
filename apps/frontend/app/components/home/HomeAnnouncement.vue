<script setup lang="ts">
import { formatDate } from '~/constants/domain'
import type { HomeArticle } from '~/types/home'

defineProps<{
  items: HomeArticle[]
}>()
</script>

<template>
  <section>
    <HomeSectionHeader
      title="公告 / 资讯"
      description="社区动态、业界资讯与重要通知"
      icon="lucide:megaphone"
      to="/articles"
    />
    <KunCard padding="none" class-name="announcement-card">
      <div v-if="items.length" class="announcement-list">
        <NuxtLink
          v-for="article in items"
          :key="article.id"
          :to="`/articles/${article.id}`"
          class="announcement-item"
        >
          <div class="announcement-title">
            <KunChip v-if="article.is_pinned" color="primary" size="sm" variant="flat">
              置顶
            </KunChip>
            <h3>{{ article.title }}</h3>
          </div>
          <p v-if="article.summary">{{ article.summary }}</p>
          <time v-if="article.published_at" :datetime="article.published_at">
            {{ formatDate(article.published_at) }}
          </time>
        </NuxtLink>
      </div>
      <p v-else class="empty-text">暂无公告或资讯。</p>
    </KunCard>
  </section>
</template>

<style scoped>
.announcement-card {
  overflow: hidden;
}

.announcement-list {
  display: flex;
  flex-direction: column;
}

.announcement-item {
  display: block;
  padding: 15px 17px;
  border-bottom: 1px solid var(--color-default-100);
  transition: background var(--kun-dur-fast) var(--ease-kun-standard);
}

.announcement-item:last-child {
  border-bottom: 0;
}

.announcement-item:hover {
  background: color-mix(in srgb, var(--color-primary) 5%, transparent);
}

.announcement-title {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 7px;
}

h3 {
  overflow: hidden;
  margin: 0;
  font-size: 14px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

p {
  display: -webkit-box;
  overflow: hidden;
  margin: 7px 0 0;
  color: var(--color-default-500);
  font-size: 12px;
  line-height: 1.6;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

time {
  display: block;
  margin-top: 7px;
  color: var(--color-default-400);
  font-size: 11px;
}

.empty-text {
  margin: 0;
  padding: 34px 18px;
  text-align: center;
}
</style>
