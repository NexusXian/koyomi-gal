<script setup lang="ts">
const route = useRoute()
const { has, hasAny } = usePermissions()

interface AdminNavItem {
  key: string
  label: string
  icon: string
  permission: string | null
}

const items: AdminNavItem[] = [
  {
    key: '/admin/galgames',
    label: 'Galgame 审核',
    icon: 'lucide:gamepad-2',
    permission: 'galgame:review'
  },
  {
    key: '/admin/reports',
    label: '资源举报',
    icon: 'lucide:flag',
    permission: 'resource_report:list'
  },
  {
    key: '/admin/resources',
    label: '资源审核',
    icon: 'lucide:folder-down',
    permission: 'resource:review'
  },
  {
    key: '/admin/tags',
    label: 'Tag',
    icon: 'lucide:tags',
    permission: 'galgame:update'
  },
  {
    key: '/admin/developers',
    label: '开发商',
    icon: 'lucide:building-2',
    permission: 'galgame:update'
  },
  {
    key: '/admin/banners',
    label: '轮播图',
    icon: 'lucide:gallery-horizontal-end',
    permission: 'banner:read'
  },
  {
    key: '/admin/articles',
    label: '文章',
    icon: 'lucide:newspaper',
    permission: 'article:read'
  },
  {
    key: '/admin/roles',
    label: '角色',
    icon: 'lucide:shield',
    permission: 'role:list'
  },
  {
    key: '/admin/permissions',
    label: '权限',
    icon: 'lucide:key-round',
    permission: 'permission:list'
  },
  {
    key: '/admin/users',
    label: '用户角色',
    icon: 'lucide:users',
    permission: 'role:assign'
  }
]

const visibleItems = computed(() =>
  items.filter((item) => item.permission === null || has(item.permission))
)

const canSeeAdmin = computed(() =>
  hasAny([
    'galgame:review',
    'resource_report:list',
    'resource:review',
    'galgame:update',
    'role:list',
    'permission:list',
    'role:assign',
    'banner:read',
    'article:read'
  ])
)

const selectedKeys = computed(() => {
  const match = visibleItems.value
    .filter((item) => route.path.startsWith(item.key))
    .sort((a, b) => b.key.length - a.key.length)[0]
  return match ? [match.key] : []
})
</script>

<template>
  <KunCard v-if="canSeeAdmin" padding="none" class-name="admin-nav-card">
    <a-menu mode="horizontal" :selected-keys="selectedKeys" class="admin-menu">
      <a-menu-item
        v-for="item in visibleItems"
        :key="item.key"
        @click="navigateTo(item.key)"
      >
        <span class="admin-menu-item">
          <KunIcon :name="item.icon" />
          {{ item.label }}
        </span>
      </a-menu-item>
    </a-menu>
  </KunCard>
</template>

<style scoped>
.admin-nav-card {
  margin-bottom: 18px;
  overflow: hidden;
}

:deep(.admin-menu) {
  border-bottom: 0;
  overflow-x: auto;
}

.admin-menu-item {
  display: inline-flex;
  align-items: center;
  gap: 7px;
}
</style>
