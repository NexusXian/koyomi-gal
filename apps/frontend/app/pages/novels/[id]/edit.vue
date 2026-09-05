<script setup lang="ts">
const route = useRoute()
const { has } = usePermissions()
const novelId = computed(() => Number(route.params.id))
const volumeManagerKey = ref(0)

useSeoMeta({
  title: '编辑小说 - Koyomi',
  description: '编辑小说条目信息'
})

// 卷册 / 关联变化时重挂卷册管理器，重新拉取最新数据
function onChanged(): void {
  volumeManagerKey.value += 1
}
</script>

<template>
  <AppPageContainer
    title="编辑小说"
    description="更新小说条目信息，编辑已发布内容会记录为贡献。"
  >
    <NovelForm :novel-id="novelId" />
    <template v-if="has('novel:update')">
      <NovelVolumeManager
        :key="volumeManagerKey"
        :novel-id="novelId"
        @changed="onChanged"
      />
      <NovelRelations :novel-id="novelId" @changed="onChanged" />
    </template>
  </AppPageContainer>
</template>
