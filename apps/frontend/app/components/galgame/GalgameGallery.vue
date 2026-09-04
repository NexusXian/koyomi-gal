<script setup lang="ts">
import type { DtoGalleryImageData } from '~/api/generated/models'

const props = defineProps<{
  galgameId: number
  gameTitle?: string
}>()

const { items, pending, error, refresh } = useGameGallery(() => props.galgameId)

const revealedSpoilers = ref(new Set<number>())
const lightboxOpen = ref(false)
const lightboxIndex = ref(0)

const lightboxImages = computed(() =>
  items.value.map((item, index) => ({
    src: item.url ?? '',
    alt: itemAlt(item, index)
  }))
)

function itemAlt(item: DtoGalleryImageData, index: number): string {
  return item.title || `${props.gameTitle || '游戏'} 游戏画面 ${index + 1}`
}

function aspectRatio(item: DtoGalleryImageData): string {
  if (item.width && item.height && item.width > 0 && item.height > 0) {
    return `${item.width} / ${item.height}`
  }
  return '16 / 9'
}

function isSpoilerHidden(item: DtoGalleryImageData): boolean {
  return Boolean(item.is_spoiler) && !revealedSpoilers.value.has(item.id ?? 0)
}

function handleItemClick(item: DtoGalleryImageData, index: number): void {
  if (isSpoilerHidden(item)) {
    const next = new Set(revealedSpoilers.value)
    next.add(item.id ?? 0)
    revealedSpoilers.value = next
    return
  }
  lightboxIndex.value = index
  lightboxOpen.value = true
}
</script>

<template>
  <KunCard padding="lg" class-name="gallery-card">
    <div class="section-head-row">
      <KunHeader
        name="游戏画面"
        :description="`共 ${items.length} 张`"
        scale="h3"
        class="section-heading"
      />
    </div>

    <div v-if="error" class="gallery-state">
      <p>游戏画面加载失败</p>
      <KunButton color="primary" variant="bordered" size="sm" @click="() => refresh()">
        重新加载
      </KunButton>
    </div>

    <div v-else-if="pending" class="gallery-grid" aria-label="游戏画面加载中">
      <div v-for="i in 4" :key="i" class="gallery-skeleton" />
    </div>

    <KunNull v-else-if="items.length === 0" message="暂无游戏画面" />

    <div v-else class="gallery-grid">
      <button
        v-for="(item, index) in items"
        :key="item.id"
        type="button"
        class="gallery-item"
        :class="{ 'is-spoiler-hidden': isSpoilerHidden(item) }"
        @click="handleItemClick(item, index)"
      >
        <img
          :src="item.url"
          :alt="itemAlt(item, index)"
          :style="{ aspectRatio: aspectRatio(item) }"
          :loading="index === 0 ? 'eager' : 'lazy'"
          decoding="async"
          referrerpolicy="no-referrer"
        />
        <span v-if="item.is_spoiler && !isSpoilerHidden(item)" class="spoiler-badge">
          <KunIcon name="lucide:eye-off" />
          剧透
        </span>
        <span v-if="isSpoilerHidden(item)" class="spoiler-cover">
          <KunIcon name="lucide:eye-off" />
          可能包含剧透
          <small>点击查看图片</small>
        </span>
      </button>
    </div>

    <KunLightbox
      :images="lightboxImages"
      :is-open="lightboxOpen"
      :initial-index="lightboxIndex"
      @update:is-open="lightboxOpen = $event"
    />
  </KunCard>
</template>

<style scoped>
.gallery-card {
  margin-bottom: 18px;
}

.section-heading {
  margin-bottom: 4px;
}

.section-head-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.gallery-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 26px 0 10px;
  color: var(--color-default-500);
  font-size: 14px;
}

.gallery-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 14px;
}

.gallery-item {
  position: relative;
  overflow: hidden;
  padding: 0;
  border: 0;
  border-radius: var(--radius-kun-lg);
  background: var(--color-content2);
  cursor: pointer;
}

.gallery-item img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.25s var(--ease-kun-standard);
}

.gallery-item:hover img {
  transform: scale(1.03);
}

.spoiler-badge {
  position: absolute;
  top: 8px;
  left: 8px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-default-900) 65%, transparent);
  color: #fff;
  font-size: 12px;
}

.spoiler-cover {
  position: absolute;
  inset: 0;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background: var(--color-content3);
  color: var(--color-default-500);
  font-size: 14px;
}

.spoiler-cover small {
  color: var(--color-default-400);
  font-size: 12px;
}

.is-spoiler-hidden img {
  filter: blur(18px);
  transform: scale(1.1);
}

.gallery-skeleton {
  aspect-ratio: 16 / 9;
  border-radius: var(--radius-kun-lg);
  background: var(--color-content2);
  animation: gallery-skeleton-pulse 1.4s ease-in-out infinite;
}

@keyframes gallery-skeleton-pulse {
  0%,
  100% {
    opacity: 1;
  }

  50% {
    opacity: 0.55;
  }
}

@media (min-width: 768px) {
  .gallery-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .gallery-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
</style>
