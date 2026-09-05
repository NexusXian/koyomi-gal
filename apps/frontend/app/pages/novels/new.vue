<script setup lang="ts">
useSeoMeta({
  title: '新建小说 - Koyomi',
  description: '创建一个新的小说条目'
})
</script>

<template>
  <AppPageContainer
    title="新建小说"
    description="填写信息创建小说条目，提交后需要审核通过才会公开。"
  >
    <NovelForm />
  </AppPageContainer>
</template>
