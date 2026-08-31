<script setup lang="ts">
useSeoMeta({
  title: '管理 - Koyomi',
  description: 'Koyomi 管理端'
})

const router = useRouter()
const { load, hasAny } = usePermissions()
const { isAuthenticated } = storeToRefs(useUserStore())

onMounted(async () => {
  if (!isAuthenticated.value) {
    void router.replace('/login')
    return
  }

  await load()

  if (
    !hasAny([
      'galgame:review',
      'resource_report:list',
      'resource:review',
      'galgame:update',
      'role:list',
      'permission:list',
      'role:assign'
    ])
  ) {
    throw createError({
      statusCode: 403,
      statusMessage: '没有管理权限',
      fatal: true
    })
  }
})
</script>

<template>
  <AppPageContainer title="管理" description="审核内容与管理系统数据。" width="wide">
    <AdminNav />
    <NuxtPage />
  </AppPageContainer>
</template>
