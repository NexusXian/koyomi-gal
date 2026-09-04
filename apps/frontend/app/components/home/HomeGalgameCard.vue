<script setup lang="ts">
import type { HomeGalgame } from '~/types/home'

const props = defineProps<{
  galgame: HomeGalgame
}>()

const fallbackCover = `data:image/svg+xml;utf8,${encodeURIComponent(
  '<svg xmlns="http://www.w3.org/2000/svg" width="300" height="400"><rect width="100%" height="100%" fill="#ede9f7"/><text x="50%" y="52%" text-anchor="middle" font-size="70" fill="#9a8abf">G</text></svg>'
)}`

const developerName = computed(() => {
  if (typeof props.galgame.developer === 'string') {
    return props.galgame.developer
  }
  return props.galgame.developer?.name ?? ''
})
</script>

<template>
  <NuxtLink :to="`/galgames/${galgame.id}`" class="galgame-card">
    <div class="cover-wrap">
      <SensitiveImage
        :src="galgame.cover_url || fallbackCover"
        :alt="`${galgame.title} 封面`"
        :sensitive="galgame.cover_sensitive"
      />
      <span v-if="galgame.rating_average != null" class="rating-badge">
        <KunIcon name="lucide:star" />
        {{ Number(galgame.rating_average).toFixed(1) }}
      </span>
    </div>
    <div class="card-copy">
      <h3 :title="galgame.title">{{ galgame.title || '未命名 Galgame' }}</h3>
      <p v-if="developerName" :title="developerName">{{ developerName }}</p>
      <div class="card-meta">
        <span><KunIcon name="lucide:heart" /> {{ galgame.favorite_count ?? 0 }}</span>
        <span v-if="galgame.release_date">{{ galgame.release_date.slice(0, 4) }}</span>
      </div>
    </div>
  </NuxtLink>
</template>

<style scoped>
.galgame-card {
  display: flex;
  min-width: 0;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--app-glass-border);
  border-radius: 16px;
  background: var(--app-glass-background);
  box-shadow: var(--shadow-kun-sm);
  -webkit-backdrop-filter: var(--app-glass-filter);
  backdrop-filter: var(--app-glass-filter);
  transition:
    transform var(--kun-dur-fast) var(--ease-kun-standard),
    box-shadow var(--kun-dur-fast) var(--ease-kun-standard);
}

.galgame-card:hover {
  box-shadow: var(--shadow-kun-md);
  transform: translateY(-3px);
}

.cover-wrap {
  position: relative;
  aspect-ratio: 3 / 4;
  overflow: hidden;
  background: var(--color-content2);
}

.galgame-card:hover .cover-wrap :deep(img) {
  transform: scale(1.04);
}

.rating-badge {
  position: absolute;
  right: 7px;
  bottom: 7px;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 3px 7px;
  border-radius: 999px;
  background: rgb(15 13 22 / 76%);
  color: #ffd36a;
  font-size: 11px;
  font-weight: 700;
  backdrop-filter: blur(6px);
}

.card-copy {
  display: flex;
  min-height: 90px;
  flex: 1;
  flex-direction: column;
  padding: 10px 11px;
}

h3 {
  display: -webkit-box;
  overflow: hidden;
  margin: 0;
  font-size: 14px;
  font-weight: 700;
  line-height: 1.45;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

p {
  overflow: hidden;
  margin: 5px 0 0;
  color: var(--color-default-500);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: auto;
  padding-top: 8px;
  color: var(--color-default-400);
  font-size: 11px;
}

.card-meta span {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
</style>
