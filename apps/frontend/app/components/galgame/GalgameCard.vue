<script setup lang="ts">
import type { DtoGalgameListItem } from '~/api/generated/models'
import { GALGAME_STATUS, domainLabel } from '~/constants/domain'

const props = defineProps<{
  galgame: DtoGalgameListItem
  showStatus?: boolean
  to?: string
}>()

const fallbackCover = `data:image/svg+xml;utf8,${encodeURIComponent(
  '<svg xmlns="http://www.w3.org/2000/svg" width="300" height="400"><rect width="100%" height="100%" fill="#e9e5f5"/><text x="50%" y="50%" text-anchor="middle" font-size="64" fill="#9b8ec4">G</text></svg>'
)}`

const ratingAverage = computed(() =>
  props.galgame.rating?.average !== undefined
    ? Number(props.galgame.rating.average.toFixed(1))
    : null
)

const statusColor = computed(() => {
  switch (props.galgame.status) {
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
</script>

<template>
  <KunCard
    :is-hoverable="true"
    padding="none"
    class-name="galgame-card"
    content-class="galgame-card-content"
  >
    <NuxtLink :to="to || `/galgames/${galgame.id}`" class="galgame-link">
      <div class="galgame-cover">
        <img
          :src="galgame.cover_url || fallbackCover"
          :alt="galgame.title || 'Galgame 封面'"
          loading="lazy"
        />
        <KunChip
          v-if="showStatus"
          class-name="status-badge"
          :color="statusColor"
          variant="solid"
          size="sm"
        >
          {{ domainLabel(GALGAME_STATUS, galgame.status) }}
        </KunChip>
        <span v-if="galgame.age_rating === 3" class="age-badge">R18</span>
      </div>

      <div class="galgame-body">
        <h3 class="galgame-title" :title="galgame.title">
          {{ galgame.title || '未命名 Galgame' }}
        </h3>
        <p v-if="galgame.original_title" class="galgame-subtitle">
          {{ galgame.original_title }}
        </p>

        <p v-if="galgame.developer?.name" class="galgame-developer">
          <KunIcon name="lucide:building-2" />
          {{ galgame.developer.name }}
        </p>

        <div v-if="galgame.tags?.length" class="galgame-tags">
          <KunChip
            v-for="tag in galgame.tags.slice(0, 3)"
            :key="tag.id"
            size="sm"
            variant="flat"
          >
            {{ tag.name }}
          </KunChip>
          <span v-if="galgame.tags.length > 3" class="tags-more">
            +{{ galgame.tags.length - 3 }}
          </span>
        </div>

        <div class="galgame-meta">
          <span v-if="ratingAverage !== null" class="meta-rating">
            <KunIcon name="lucide:star" />
            {{ ratingAverage }}
            <em>({{ galgame.rating?.count ?? 0 }})</em>
          </span>
          <span class="meta-item">
            <KunIcon name="lucide:heart" />
            {{ galgame.statistics?.favorite_count ?? 0 }}
          </span>
          <span v-if="galgame.release_date" class="meta-item">
            {{ galgame.release_date.slice(0, 4) }}
          </span>
        </div>
      </div>
    </NuxtLink>
  </KunCard>
</template>

<style scoped>
.galgame-card {
  overflow: hidden;
  transition:
    transform var(--kun-dur-fast) var(--ease-kun-standard),
    box-shadow var(--kun-dur-fast) var(--ease-kun-standard);
}

.galgame-card:hover {
  transform: translateY(-3px);
}

:deep(.galgame-card-content) {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 0;
}

.galgame-link {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.galgame-cover {
  position: relative;
  aspect-ratio: 3 / 4;
  overflow: hidden;
  background: var(--color-content2);
}

.galgame-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform var(--kun-dur-base) var(--ease-kun-standard);
}

.galgame-card:hover .galgame-cover img {
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

.galgame-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 6px;
  padding: 12px;
}

.galgame-title {
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

.galgame-subtitle {
  display: -webkit-box;
  overflow: hidden;
  margin: 0;
  color: var(--color-default-500);
  font-size: 12px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 1;
}

.galgame-developer {
  display: flex;
  align-items: center;
  gap: 5px;
  margin: 0;
  color: var(--color-default-500);
  font-size: 13px;
}

.galgame-tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
}

.tags-more {
  color: var(--color-default-400);
  font-size: 12px;
}

.galgame-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: auto;
  padding-top: 8px;
  color: var(--color-default-500);
  font-size: 13px;
}

.meta-rating,
.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.meta-rating {
  color: var(--color-warning-500);
  font-weight: 600;
}

.meta-rating em {
  color: var(--color-default-400);
  font-style: normal;
  font-weight: 400;
  font-size: 12px;
}
</style>
