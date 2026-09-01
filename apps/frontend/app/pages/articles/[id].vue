<script setup lang="ts">
import { formatDate } from '~/constants/domain'
import { createContentService } from '~/services/content'

const route = useRoute()
const id = Number(route.params.id)
if (!Number.isInteger(id) || id <= 0) {
  throw createError({ statusCode: 404, statusMessage: '文章不存在' })
}

const contentService = createContentService(useNuxtApp().$api)
const { data: article, pending, error, refresh } = await useAsyncData(
  `article-${id}`,
  () => contentService.getArticle(id)
)

if (error.value) {
  const statusCode = (error.value as { statusCode?: number }).statusCode
  if (statusCode === 404) {
    throw createError({
      statusCode: 404,
      statusMessage: '文章不存在或尚未发布',
      fatal: true
    })
  }
  setResponseStatus(statusCode && statusCode >= 400 ? statusCode : 500)
} else if (!article.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '文章不存在或尚未发布',
    fatal: true
  })
}

useSeoMeta({
  title: () => article.value ? `${article.value.title} - Koyomi Gal` : '文章 - Koyomi Gal',
  description: () => article.value?.summary || '阅读 Koyomi Gal 社区文章。',
  ogTitle: () => article.value?.title,
  ogDescription: () => article.value?.summary || undefined,
  ogImage: () => article.value?.cover_url || undefined,
  ogType: 'article'
})
</script>

<template>
  <AppPageContainer>
    <KunSkeleton v-if="pending" class="detail-skeleton" />

    <KunCard v-else-if="error" padding="lg">
      <div class="detail-state">
        <KunIcon name="lucide:file-warning" />
        <h1>文章无法加载</h1>
        <p>{{ getApiErrorMessage(error, '文章可能不存在或暂时无法访问。') }}</p>
        <KunButton color="primary" @click="() => refresh()">重新加载</KunButton>
      </div>
    </KunCard>

    <article v-else-if="article" class="article-detail">
      <header class="article-header">
        <div class="article-type">
          <KunChip v-if="article.is_pinned" color="primary" size="sm" variant="flat">置顶</KunChip>
          <span>{{ article.type === 'announcement' ? '公告' : article.type === 'news' ? '资讯' : article.type }}</span>
        </div>
        <h1>{{ article.title }}</h1>
        <p v-if="article.summary" class="article-summary">{{ article.summary }}</p>
        <div class="article-meta">
          <time v-if="article.published_at" :datetime="article.published_at">
            <KunIcon name="lucide:calendar-days" /> {{ formatDate(article.published_at) }}
          </time>
          <span v-if="article.view_count != null"><KunIcon name="lucide:eye" /> {{ article.view_count }}</span>
        </div>
      </header>

      <img v-if="article.cover_url" class="detail-cover" :src="article.cover_url" :alt="article.title" />

      <KunCard padding="lg" class-name="content-card">
        <div class="article-content">{{ article.content || '暂无正文内容。' }}</div>
      </KunCard>

      <NuxtLink to="/articles" class="back-link">
        <KunIcon name="lucide:arrow-left" /> 返回资讯与公告
      </NuxtLink>
    </article>
  </AppPageContainer>
</template>

<style scoped>
.detail-skeleton { min-height: 560px; }
.detail-state { display: flex; min-height: 260px; align-items: center; justify-content: center; flex-direction: column; color: var(--color-default-500); text-align: center; }
.detail-state > :deep(svg) { color: var(--color-warning); font-size: 32px; }
.detail-state h1 { margin: 12px 0 0; color: var(--color-foreground); font-size: 22px; }
.detail-state p { margin: 8px 0 18px; }
.article-header { max-width: 820px; margin: 12px auto 28px; text-align: center; }
.article-type { display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--color-primary-600); font-size: 13px; font-weight: 650; }
.article-header h1 { margin: 13px 0 0; font-size: clamp(29px, 5vw, 45px); line-height: 1.22; letter-spacing: -0.035em; }
.article-summary { max-width: 680px; margin: 15px auto 0; color: var(--color-default-500); font-size: 15px; line-height: 1.75; }
.article-meta { display: flex; align-items: center; justify-content: center; gap: 18px; margin-top: 17px; color: var(--color-default-400); font-size: 12px; }
.article-meta span, .article-meta time { display: inline-flex; align-items: center; gap: 5px; }
.detail-cover { display: block; width: 100%; max-height: 460px; margin-bottom: 20px; border-radius: 18px; object-fit: cover; }
.article-content { min-height: 220px; color: var(--color-foreground); font-size: 16px; line-height: 1.9; overflow-wrap: anywhere; white-space: pre-wrap; }
.back-link { display: inline-flex; align-items: center; gap: 6px; margin-top: 20px; color: var(--color-primary-600); font-size: 14px; }
</style>
