<script setup lang="ts">
useSeoMeta({
  title: '新建 Galgame - Koyomi',
  description: '创建一个新的 Galgame 条目'
})
</script>

<template>
  <AppPageContainer
    title="新建 Galgame"
    description="填写信息创建 Galgame 条目，提交后需要审核通过才会公开。"
  >
    <GalgameForm />
  </AppPageContainer>
</template>
