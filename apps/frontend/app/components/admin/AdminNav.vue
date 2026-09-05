<script setup lang="ts">
const route = useRoute()
const { hasAny } = usePermissions()

interface AdminNavItem {
  key: string
  label: string
  icon: string
  permissions: string[] | null
}

const items: AdminNavItem[] = [
  {
    key: '/admin/galgames',
    label: 'Galgame 审核',
    icon: 'lucide:gamepad-2',
    permissions: ['galgame:review']
  },
  {
    key: '/admin/novels',
    label: '小说管理',
    icon: 'lucide:book-open',
    permissions: ['novel:review']
  },
  {
    key: '/admin/gallery-reviews',
    label: '插图审核',
    icon: 'lucide:image-plus',
    permissions: ['galgame_gallery:review']
  },
  {
    key: '/admin/import',
    label: '外部数据导入',
    icon: 'lucide:cloud-download',
    permissions: ['galgame:import', 'galgame:import:batch']
  },
  {
    key: '/admin/import/matches',
    label: '匹配审核',
    icon: 'lucide:link-2',
    permissions: ['galgame:import']
  },
  {
    key: '/admin/reports',
    label: '资源举报',
    icon: 'lucide:flag',
    permissions: ['resource_report:list']
  },
  {
    key: '/admin/feedback',
    label: '意见反馈',
    icon: 'lucide:inbox',
    permissions: ['feedback:read']
  },
  {
    key: '/admin/images',
    label: '图片管理',
    icon: 'lucide:image',
    permissions: ['image:read', 'image:manage']
  },
  {
    key: '/admin/resources',
    label: '资源审核',
    icon: 'lucide:folder-down',
    permissions: ['resource:review']
  },
  {
    key: '/admin/community',
    label: '社区管理',
    icon: 'lucide:messages-square',
    permissions: ['post:moderate', 'comment:moderate']
  },
  {
    key: '/admin/tags',
    label: 'Tag',
    icon: 'lucide:tags',
    permissions: ['galgame:update']
  },
  {
    key: '/admin/developers',
    label: '开发商',
    icon: 'lucide:building-2',
    permissions: ['galgame:update']
  },
  {
    key: '/admin/banners',
    label: '轮播图',
    icon: 'lucide:gallery-horizontal-end',
    permissions: ['banner:read']
  },
  {
    key: '/admin/backgrounds',
    label: '背景预设',
    icon: 'lucide:wallpaper',
    permissions: [
      'background_preset:read',
      'background_preset:create',
      'background_preset:update',
      'background_preset:delete'
    ]
  },
  {
    key: '/admin/articles',
    label: '文章',
    icon: 'lucide:newspaper',
    permissions: ['article:read']
  },
  {
    key: '/admin/roles',
    label: '角色',
    icon: 'lucide:shield',
    permissions: [
      'role:list',
      'role:create',
      'role:update',
      'role:delete',
      'permission:assign'
    ]
  },
  {
    key: '/admin/permissions',
    label: '权限',
    icon: 'lucide:key-round',
    permissions: [
      'permission:list',
      'permission:create',
      'permission:update',
      'permission:delete'
    ]
  },
  {
    key: '/admin/users',
    label: '用户管理',
    icon: 'lucide:users',
    permissions: [
      'user:list',
      'user:read',
      'user:create',
      'user:update',
      'user:delete',
      'role:assign'
    ]
  }
]

const visibleItems = computed(() =>
  items.filter((item) => item.permissions === null || hasAny(item.permissions))
)

const canSeeAdmin = computed(() => visibleItems.value.length > 0)

const selectedKeys = computed(() => {
  const match = visibleItems.value
    .filter((item) => route.path.startsWith(item.key))
    .sort((a, b) => b.key.length - a.key.length)[0]
  return match ? [match.key] : []
})
</script>

<template>
  <KunCard v-if="canSeeAdmin" padding="none" class-name="admin-nav-card">
    <a-menu
      mode="horizontal"
      :selected-keys="selectedKeys"
      :disabled-overflow="true"
      class="admin-menu"
    >
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
  flex-wrap: wrap;
  row-gap: 2px;
  padding-block: 4px;
  height: auto;
  line-height: 2.5;
}

.admin-menu-item {
  display: inline-flex;
  align-items: center;
  gap: 7px;
}
</style>
