<script setup lang="ts">
const route = useRoute()
const router = useRouter()
const { has, hasAny, load } = usePermissions()
const { initialize } = useAuth()
const userStore = useUserStore()
const ready = ref(false)

const galgameId = computed(() => Number(route.params.id))

onMounted(async () => {
  await initialize()
  if (!userStore.isAuthenticated) {
    void router.replace('/login')
    return
  }
  await load()
  ready.value = true
})

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
    <a-spin v-if="!ready" tip="正在加载编辑权限..." />
    <template v-else>
      <a-alert v-if="!hasAny(['galgame:update', 'character:manage', 'galgame_gallery:manage'])" type="warning" show-icon message="没有编辑此条目的权限" />
      <GalgameForm v-if="has('galgame:update')" :galgame-id="galgameId" />
      <GalgameCharacterManager v-if="has('character:manage')" :key="galgameId" :galgame-id="galgameId" />
      <GalgameGalleryManager v-if="has('galgame_gallery:manage')" :galgame-id="galgameId" />
    </template>
  </AppPageContainer>
</template>
