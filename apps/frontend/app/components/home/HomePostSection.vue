<script setup lang="ts">
import type { HomePost } from '~/types/home'

withDefaults(
  defineProps<{
    title: string
    description?: string
    icon?: string
    items: HomePost[]
  }>(),
  {
    description: '',
    icon: 'lucide:messages-square'
  }
)
</script>

<template>
  <section>
    <HomeSectionHeader
      :title="title"
      :description="description"
      :icon="icon"
      to="/posts"
    />
    <div v-if="items.length" class="post-list">
      <HomePostItem v-for="post in items" :key="post.id" :post="post" />
    </div>
    <KunCard v-else padding="md">
      <p class="empty-text">这里还很安静，期待第一篇社区帖子。</p>
    </KunCard>
  </section>
</template>

<style scoped>
.post-list {
  display: flex;
  flex-direction: column;
  gap: 9px;
}

.empty-text {
  margin: 0;
  color: var(--color-default-400);
  font-size: 14px;
  text-align: center;
}
</style>
