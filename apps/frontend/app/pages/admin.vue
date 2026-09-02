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
      'image:read',
      'image:manage',
      'resource:review',
      'post:moderate',
      'comment:moderate',
      'galgame:update',
      'role:list',
      'role:create',
      'role:update',
      'role:delete',
      'permission:list',
      'permission:create',
      'permission:update',
      'permission:delete',
      'permission:assign',
      'role:assign',
      'user:list',
      'user:read',
      'user:create',
      'user:update',
      'user:delete',
      'banner:read',
      'article:read'
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
