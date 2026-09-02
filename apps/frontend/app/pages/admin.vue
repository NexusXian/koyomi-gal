<script setup lang="ts">
useSeoMeta({
  title: '管理 - Koyomi',
  description: 'Koyomi 管理端'
})

const router = useRouter()
const { load, hasAny } = usePermissions()
const { initialize } = useAuth()
const { isAuthenticated } = storeToRefs(useUserStore())

onMounted(async () => {
  // Wait for the in-flight session refresh (auth-init plugin) so a full page
  // load is not misread as "not logged in".
  await initialize()

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
      'background_preset:read',
      'article:read',
      'feedback:read'
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
