<script setup lang="ts">
const route = useRoute()
const { has } = usePermissions()

const galgameId = computed(() => Number(route.params.id))

useSeoMeta({
  title: '编辑 Galgame - Koyomi',
  description: '编辑 Galgame 条目信息'
})
</script>

<template>
  <AppPageContainer
    title="编辑 Galgame"
    description="更新 Galgame 条目信息，更新后可能需要重新审核。"
  >
    <GalgameForm :galgame-id="galgameId" />
    <GalgameGalleryManager v-if="has('galgame_gallery:manage')" :galgame-id="galgameId" />
  </AppPageContainer>
</template>
