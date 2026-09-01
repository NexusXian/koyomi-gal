<script setup lang="ts">
import type { HomeGalgame } from '~/types/home'

withDefaults(
  defineProps<{
    title: string
    description?: string
    icon?: string
    items: HomeGalgame[]
    compact?: boolean
  }>(),
  {
    description: '',
    icon: 'lucide:gamepad-2',
    compact: false
  }
)
</script>

<template>
  <section class="galgame-section">
    <HomeSectionHeader
      :title="title"
      :description="description"
      :icon="icon"
      to="/galgames"
    />
    <div v-if="items.length" class="galgame-grid" :class="{ compact }">
      <HomeGalgameCard
        v-for="galgame in items"
        :key="galgame.id"
        :galgame="galgame"
      />
    </div>
    <KunCard v-else padding="md" class-name="empty-card">
      <p class="empty-text">暂时没有可展示的 Galgame。</p>
    </KunCard>
  </section>
</template>

<style scoped>
.galgame-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 11px;
}

.empty-text {
  margin: 0;
  color: var(--color-default-400);
  font-size: 14px;
  text-align: center;
}

@media (min-width: 640px) {
  .galgame-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 14px;
  }
}

@media (min-width: 1024px) {
  .galgame-grid:not(.compact) {
    grid-template-columns: repeat(6, minmax(0, 1fr));
  }
}
</style>
