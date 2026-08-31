<script setup lang="ts">
const route = useRoute()

const presetGalgameId = computed(() =>
  route.query.galgame_id ? Number(route.query.galgame_id) : undefined
)

useSeoMeta({
  title: '发帖 - Koyomi',
  description: '发表新的帖子'
})
</script>

<template>
  <AppPageContainer title="发帖" description="分享你的想法、攻略或讨论。">
    <PostForm :preset-galgame-id="presetGalgameId" />
  </AppPageContainer>
</template>
