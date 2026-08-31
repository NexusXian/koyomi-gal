<script setup lang="ts">
const route = useRoute()

const postId = computed(() => Number(route.params.id))

useSeoMeta({
  title: '编辑帖子 - Koyomi',
  description: '编辑帖子内容'
})
</script>

<template>
  <AppPageContainer title="编辑帖子" description="更新帖子标题或内容。">
    <PostForm :post-id="postId" />
  </AppPageContainer>
</template>
