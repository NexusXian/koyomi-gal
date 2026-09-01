<script setup lang="ts">
import { formatDate } from '~/constants/domain'
import { createContentService } from '~/services/content'

useSeoMeta({
  title: '资讯与公告 - Koyomi Gal',
  description: '浏览 Koyomi Gal 社区公告、Galgame 资讯与专题文章。'
})

const route = useRoute()
const router = useRouter()
const contentService = createContentService(useNuxtApp().$api)
const limit = 12

const articleTypes = new Set(['announcement', 'news', 'event', 'update'])
const page = computed(() => {
  const value = Number(route.query.page ?? 1)
  return Number.isFinite(value) ? Math.max(1, Math.trunc(value)) : 1
})
const articleType = computed(() => {
  const value = typeof route.query.type === 'string' ? route.query.type : ''
  return articleTypes.has(value) ? value : ''
})

const { data, pending, error, refresh } = await useAsyncData(
  'articles-list',
  () => contentService.listArticles({
    type: articleType.value || undefined,
    page: page.value,
    limit
  }),
  { watch: [page, articleType] }
)

const totalPage = computed(() =>
  Math.max(1, Math.ceil((data.value?.total ?? 0) / (data.value?.limit || limit)))
)

function setType(value: string): void {
  void router.push({ query: { type: value || undefined } })
}

function setPage(value: number): void {
  void router.push({
    query: {
      type: articleType.value || undefined,
      page: value > 1 ? value : undefined
    }
  })
}
</script>

<template>
  <AppPageContainer
    title="资讯与公告"
    description="社区动态、站内公告与 Galgame 相关资讯。"
    width="wide"
  >
    <div class="article-tabs" role="navigation" aria-label="文章类型">
      <button :class="{ active: !articleType }" type="button" @click="setType('')">全部</button>
      <button :class="{ active: articleType === 'announcement' }" type="button" @click="setType('announcement')">公告</button>
      <button :class="{ active: articleType === 'news' }" type="button" @click="setType('news')">资讯</button>
      <button :class="{ active: articleType === 'event' }" type="button" @click="setType('event')">活动</button>
      <button :class="{ active: articleType === 'update' }" type="button" @click="setType('update')">更新</button>
    </div>

    <div v-if="pending" class="article-grid">
      <KunSkeleton v-for="item in 6" :key="item" class="article-skeleton" />
    </div>

    <KunCard v-else-if="error" padding="lg">
      <div class="list-state">
        <p>{{ getApiErrorMessage(error, '文章列表加载失败') }}</p>
        <KunButton color="primary" @click="() => refresh()">重新加载</KunButton>
      </div>
    </KunCard>

    <div v-else-if="data?.items.length" class="article-grid">
      <NuxtLink
        v-for="article in data.items"
        :key="article.id"
        :to="`/articles/${article.id}`"
        class="article-card"
      >
        <div v-if="article.cover_url" class="article-cover">
          <img :src="article.cover_url" :alt="article.title" loading="lazy" />
        </div>
        <div class="article-body">
          <div class="article-labels">
            <KunChip v-if="article.is_pinned" color="primary" size="sm" variant="flat">置顶</KunChip>
            <span>{{ article.type === 'announcement' ? '公告' : article.type === 'news' ? '资讯' : article.type }}</span>
          </div>
          <h2>{{ article.title }}</h2>
          <p>{{ article.summary || '阅读全文了解详细内容。' }}</p>
          <time v-if="article.published_at" :datetime="article.published_at">
            {{ formatDate(article.published_at) }}
          </time>
        </div>
      </NuxtLink>
    </div>

    <KunCard v-else padding="lg">
      <div class="list-state"><p>当前分类下还没有文章。</p></div>
    </KunCard>

    <div v-if="totalPage > 1" class="pagination-row">
      <KunPagination
        :current-page="page"
        :total-page="totalPage"
        :is-loading="pending"
        @update:current-page="setPage"
      />
    </div>
  </AppPageContainer>
</template>

<style scoped>
.article-tabs {
  display: flex;
  overflow-x: auto;
  gap: 7px;
  margin-bottom: 20px;
}

.article-tabs button {
  padding: 7px 15px;
  border: 1px solid var(--app-glass-border);
  border-radius: 999px;
  background: var(--app-glass-background);
  color: var(--color-default-500);
  cursor: pointer;
  -webkit-backdrop-filter: var(--app-glass-filter);
  backdrop-filter: var(--app-glass-filter);
}

.article-tabs button.active {
  border-color: var(--color-primary);
  background: color-mix(in srgb, var(--color-primary) 11%, transparent);
  color: var(--color-primary-600);
  font-weight: 650;
}

.article-grid {
  display: grid;
  grid-template-columns: repeat(1, minmax(0, 1fr));
  gap: 16px;
}

.article-card {
  display: flex;
  min-width: 0;
  min-height: 180px;
  overflow: hidden;
  border: 1px solid var(--app-glass-border);
  border-radius: 16px;
  background: var(--app-glass-background);
  box-shadow: var(--shadow-kun-sm);
  -webkit-backdrop-filter: var(--app-glass-filter);
  backdrop-filter: var(--app-glass-filter);
  transition: transform var(--kun-dur-fast) var(--ease-kun-standard);
}

.article-card:hover { transform: translateY(-2px); }

.article-cover { width: 38%; flex: 0 0 auto; overflow: hidden; }
.article-cover img { width: 100%; height: 100%; object-fit: cover; transition: transform var(--kun-dur-base) var(--ease-kun-standard); }
.article-card:hover img { transform: scale(1.04); }
.article-body { display: flex; min-width: 0; flex: 1; flex-direction: column; padding: 17px; }
.article-labels { display: flex; align-items: center; gap: 7px; color: var(--color-primary-600); font-size: 12px; }
.article-body h2 { display: -webkit-box; overflow: hidden; margin: 9px 0 0; font-size: 17px; line-height: 1.45; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.article-body p { display: -webkit-box; overflow: hidden; margin: 8px 0; color: var(--color-default-500); font-size: 13px; line-height: 1.65; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.article-body time { margin-top: auto; color: var(--color-default-400); font-size: 11px; }
.article-skeleton { min-height: 180px; }
.list-state { display: flex; align-items: center; justify-content: center; flex-direction: column; gap: 12px; min-height: 130px; color: var(--color-default-500); text-align: center; }
.list-state p { margin: 0; }
.pagination-row { display: flex; justify-content: center; margin-top: 26px; }

@media (min-width: 760px) {
  .article-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 479px) {
  .article-card { min-height: 160px; }
  .article-cover { width: 34%; }
  .article-body { padding: 14px; }
}
</style>
