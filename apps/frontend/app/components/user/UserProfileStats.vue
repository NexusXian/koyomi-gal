<script setup lang="ts">
import type { DtoPublicUserProfile } from '~/api/generated/models'

defineProps<{ profile: DtoPublicUserProfile }>()
</script>

<template>
  <div class="profile-stats">
    <div v-if="profile.access?.can_view_posts"><strong>{{ profile.post_count ?? 0 }}</strong><span>帖子</span></div>
    <div v-if="profile.access?.can_view_comments"><strong>{{ profile.comment_count ?? 0 }}</strong><span>评论</span></div>
    <div v-if="profile.access?.can_view_ratings"><strong>{{ profile.rating_count ?? 0 }}</strong><span>评分</span></div>
    <div v-if="profile.access?.can_view_favorites"><strong>{{ profile.favorite_count ?? 0 }}</strong><span>收藏</span></div>
  </div>
</template>

<style scoped>
.profile-stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(110px, 1fr)); gap: 10px; }
.profile-stats div { display: grid; gap: 3px; padding: 18px; border: 1px solid var(--app-glass-border); border-radius: var(--radius-kun-lg); background: var(--app-glass-background); text-align: center; backdrop-filter: var(--app-glass-filter); }
.profile-stats strong { font-size: 24px; }
.profile-stats span { color: var(--color-default-500); font-size: 13px; }
</style>
