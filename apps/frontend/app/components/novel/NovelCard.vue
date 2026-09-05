<script setup lang="ts">
import type { DtoNovelListItem } from '~/api/generated/models'
import {
  GALGAME_STATUS,
  NOVEL_RELEASE_STATUS,
  NOVEL_STATUS,
  domainLabel,
  domainSlug
} from '~/constants/domain'

const props = defineProps<{
  novel: DtoNovelListItem
  showStatus?: boolean
  to?: string
}>()

const fallbackCover =
  'data:image/svg+xml;utf8,' +
  encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" width="300" height="400"><rect width="100%" height="100%" fill="#e9e5f5"/><text x="50%" y="50%" text-anchor="middle" font-size="64" fill="#9b8ec4">N</text></svg>'
  )

const statusColor = computed(() => {
  switch (props.novel.status) {
    case 1:
      return 'success'
    case 2:
      return 'danger'
    case 3:
      return 'warning'
    default:
      return 'default'
  }
})

const releaseStatusLabel = computed(() => {
  const option = domainSlug(NOVEL_RELEASE_STATUS, props.novel.release_status ?? '')
  return option?.label ?? '未知'
})

const novelHref = computed(
  () => props.to || '/novels/' + String(props.novel.id || '')
)
</script>

<template>
  <KunCard
    :is-hoverable="true"
    padding="none"
    class-name="novel-card"
    content-class="novel-card-content"
  >
    <NuxtLink :to="novelHref" class="novel-link">
      <div class="novel-cover">
        <SensitiveImage
          :src="novel.cover_url || fallbackCover"
          :alt="novel.title || '小说封面'"
          :sensitive="novel.is_cover_sensitive"
        />
        <KunChip
          v-if="showStatus"
          class-name="status-badge"
          :color="statusColor"
          variant="solid"
          size="sm"
        >
          {{ domainLabel(NOVEL_STATUS, novel.status) }}
        </KunChip>
        <span v-if="novel.age_rating === 3" class="age-badge">R18</span>
      </div>

      <div class="novel-body">
        <h3 class="novel-title" :title="novel.title">
          {{ novel.title || '未命名小说' }}
        </h3>
        <p v-if="novel.original_title" class="novel-subtitle">
          {{ novel.original_title }}
        </p>

        <p v-if="novel.author" class="novel-author">
          <KunIcon name="lucide:pen-line" />
          {{ novel.author }}
        </p>

        <p v-if="novel.publisher || novel.label" class="novel-publisher">
          <KunIcon name="lucide:book-marked" />
          {{ [novel.publisher, novel.label].filter(Boolean).join(' / ') }}
        </p>

        <div v-if="novel.tags?.length" class="novel-tags">
          <KunChip
            v-for="tag in novel.tags.slice(0, 3)"
            :key="tag.id"
            size="sm"
            variant="flat"
          >
            {{ tag.name }}
          </KunChip>
          <span v-if="novel.tags.length > 3" class="tags-more">
            +{{ novel.tags.length - 3 }}
          </span>
        </div>

        <div class="novel-meta">
          <span class="meta-item">
            <KunIcon name="lucide:library" />
            {{ novel.statistics?.volume_count ?? 0 }} 卷
          </span>
          <span class="meta-item">{{ releaseStatusLabel }}</span>
          <span v-if="novel.first_release_date" class="meta-item">
            {{ novel.first_release_date.slice(0, 4) }}
          </span>
        </div>
      </div>
    </NuxtLink>
  </KunCard>
</template>

<style scoped>
.novel-card {
  overflow: hidden;
  transition:
    transform var(--kun-dur-fast) var(--ease-kun-standard),
    box-shadow var(--kun-dur-fast) var(--ease-kun-standard);
}

.novel-card:hover {
  transform: translateY(-3px);
}

:deep(.novel-card-content) {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 0;
}

.novel-link {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.novel-cover {
  position: relative;
  aspect-ratio: 3 / 4;
  overflow: hidden;
  background: var(--color-content2);
}

.novel-card:hover .novel-cover :deep(img) {
  transform: scale(1.05);
}

.age-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 2px 8px;
  border-radius: var(--radius-kun-sm);
  background: color-mix(in srgb, var(--color-danger) 88%, transparent);
  color: #fff;
  font-size: 12px;
  font-weight: 700;
}

:deep(.status-badge) {
  position: absolute;
  top: 8px;
  left: 8px;
}

.novel-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 6px;
  padding: 12px;
}

.novel-title {
  display: -webkit-box;
  overflow: hidden;
  margin: 0;
  color: var(--color-foreground);
  font-size: 15px;
  font-weight: 700;
  line-height: 1.4;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.novel-subtitle {
  display: -webkit-box;
  overflow: hidden;
  margin: 0;
  color: var(--color-default-500);
  font-size: 12px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 1;
}

.novel-author,
.novel-publisher {
  display: flex;
  align-items: center;
  gap: 5px;
  margin: 0;
  color: var(--color-default-500);
  font-size: 13px;
}

.novel-tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
}

.tags-more {
  color: var(--color-default-400);
  font-size: 12px;
}

.novel-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: auto;
  padding-top: 8px;
  color: var(--color-default-500);
  font-size: 13px;
}

.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
</style>
