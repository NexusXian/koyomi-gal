<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { message } from 'ant-design-vue'

defineProps<{
  isDark: boolean
  navigationOpen: boolean
}>()

defineEmits<{
  toggleNavigation: []
  toggleColorMode: []
}>()

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { user, isAuthenticated } = storeToRefs(userStore)
const { logout } = useAuth()
const query = ref(typeof route.query.q === 'string' ? route.query.q : '')
const mobileSearchOpen = ref(false)
const loggingOut = ref(false)
const isHydrated = ref(false)
const brandIcon = `data:image/svg+xml;utf8,${encodeURIComponent(
  '<svg xmlns="http://www.w3.org/2000/svg" width="40" height="40"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop stop-color="#7c3aed"/><stop offset="1" stop-color="#ec4899"/></linearGradient></defs><rect width="40" height="40" rx="11" fill="url(#g)"/><text x="20" y="27" font-size="20" font-weight="700" fill="#fff" text-anchor="middle" font-family="sans-serif">K</text></svg>'
)}`

const avatarUser = computed(() =>
  user.value
    ? {
        id: user.value.id,
        name: user.value.username,
        avatar: user.value.avatar
      }
    : null
)

async function handleAccountAction({ key }: { key: string | number }): Promise<void> {
  if (key === 'logout') {
    loggingOut.value = true
    try {
      await logout()
      message.success('已退出登录')
      void router.push('/')
    } catch {
      message.error('退出失败，请重试')
    } finally {
      loggingOut.value = false
    }
  }
}

function submitSearch() {
  const normalizedQuery = query.value.trim()

  // Search results live on /galgames; keep its filters only when already there.
  const carriedQuery = route.path === '/galgames' ? route.query : {}

  navigateTo({
    path: '/galgames',
    query: {
      ...carriedQuery,
      q: normalizedQuery || undefined
    }
  })

  mobileSearchOpen.value = false
}

watch(
  () => [route.path, route.query.q] as const,
  ([, nextQuery]) => {
    query.value = typeof nextQuery === 'string' ? nextQuery : ''
    mobileSearchOpen.value = false
  }
)

onMounted(() => {
  isHydrated.value = true
})
</script>

<template>
  <header class="app-header">
    <div class="header-inner">
      <div class="header-leading">
        <KunButton
          class-name="navigation-toggle"
          color="default"
          variant="light"
          size="sm"
          rounded="full"
          :is-icon-only="true"
          aria-label="打开导航"
          :aria-expanded="navigationOpen"
          @click="$emit('toggleNavigation')"
        >
          <KunIcon name="lucide:menu" />
        </KunButton>

        <div class="brand">
          <KunBrand
            name="Koyomi"
            :icon-src="brandIcon"
            icon-alt="Koyomi"
            icon-class="size-9 rounded-kun-lg"
            name-class="text-xl font-bold tracking-tight"
            to="/"
          />
        </div>
      </div>

      <form class="desktop-search" role="search" @submit.prevent="submitSearch">
        <KunInput
          v-model="query"
          type="search"
          size="sm"
          rounded="full"
          placeholder="搜索 Galgame、角色或话题"
          aria-label="搜索"
          :is-clearable="true"
        >
          <template #prefix>
            <KunIcon name="lucide:search" class="search-icon" />
          </template>
        </KunInput>
      </form>

      <div class="header-actions">
        <KunTooltip :text="mobileSearchOpen ? '关闭搜索' : '打开搜索'" position="bottom">
          <KunButton
            class-name="mobile-search-toggle"
            color="default"
            variant="light"
            size="sm"
            rounded="full"
            :is-icon-only="true"
            :aria-label="mobileSearchOpen ? '关闭搜索' : '打开搜索'"
            :aria-expanded="mobileSearchOpen"
            @click="mobileSearchOpen = !mobileSearchOpen"
          >
            <KunIcon :name="mobileSearchOpen ? 'lucide:x' : 'lucide:search'" />
          </KunButton>
        </KunTooltip>

        <KunTooltip :text="isDark ? '切换到浅色模式' : '切换到深色模式'" position="bottom">
          <KunButton
            color="default"
            variant="light"
            size="sm"
            rounded="full"
            :is-icon-only="true"
            :aria-label="isDark ? '切换到浅色模式' : '切换到深色模式'"
            @click="$emit('toggleColorMode')"
          >
            <KunIcon :name="isDark ? 'lucide:sun' : 'lucide:moon'" />
          </KunButton>
        </KunTooltip>

        <template v-if="isHydrated && isAuthenticated">
          <a-dropdown>
            <div class="account" :title="user?.username">
              <KunAvatar :user="avatarUser" :is-navigation="false" size="sm" />
              <span class="account-name">{{ user?.username }}</span>
            </div>
            <template #overlay>
              <a-menu @click="handleAccountAction">
                <a-menu-item key="galgames" disabled>
                  <span class="menu-item-label">
                    <KunIcon name="lucide:gamepad-2" />
                    我的 Galgame
                  </span>
                </a-menu-item>
                <a-menu-divider />
                <a-menu-item key="logout" :disabled="loggingOut">
                  <span class="menu-item-label menu-item-danger">
                    <KunIcon name="lucide:log-out" />
                    {{ loggingOut ? '退出中...' : '退出登录' }}
                  </span>
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </template>

        <template v-else>
          <KunButton
            color="primary"
            variant="bordered"
            size="sm"
            rounded="full"
            href="/login"
          >
            登录
          </KunButton>
          <KunButton
            color="primary"
            variant="solid"
            size="sm"
            rounded="full"
            href="/register"
          >
            注册
          </KunButton>
        </template>
      </div>
    </div>

    <form
      v-if="mobileSearchOpen"
      class="mobile-search"
      role="search"
      @submit.prevent="submitSearch"
    >
      <KunInput
        v-model="query"
        type="search"
        size="sm"
        rounded="full"
        placeholder="搜索 Galgame、角色或话题"
        aria-label="搜索"
        :is-clearable="true"
        autofocus
      >
        <template #prefix>
          <KunIcon name="lucide:search" class="search-icon" />
        </template>
      </KunInput>
    </form>
  </header>
</template>

<style scoped>
.app-header {
  position: fixed;
  z-index: var(--z-kun-sticky);
  top: 0;
  right: 0;
  left: 0;
  border-bottom: 1px solid var(--app-glass-border);
  background: var(--app-glass-background);
  -webkit-backdrop-filter: var(--app-glass-filter);
  backdrop-filter: var(--app-glass-filter);
}

.header-inner {
  display: flex;
  height: 60px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 16px;
}

.header-leading,
.header-actions,
.account {
  display: flex;
  align-items: center;
}

.header-leading,
.header-actions {
  gap: 8px;
}

.brand {
  min-width: 0;
}

.desktop-search {
  display: none;
  width: min(42vw, 520px);
}

.search-icon {
  color: var(--color-default-400);
  font-size: 16px;
}

.account {
  min-width: 0;
  gap: 8px;
  padding-left: 2px;
  border-radius: var(--radius-kun-md);
  cursor: pointer;
}

.menu-item-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.menu-item-danger {
  color: var(--color-danger);
}

.account-name {
  display: none;
  max-width: 112px;
  overflow: hidden;
  color: var(--color-default-600);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mobile-search {
  position: absolute;
  top: calc(100% + 8px);
  right: 12px;
  left: 12px;
  padding: 10px;
  border: 1px solid var(--app-glass-border);
  border-radius: var(--radius-kun-lg);
  background: var(--app-glass-background);
  box-shadow: var(--shadow-kun-lg);
  -webkit-backdrop-filter: var(--app-glass-filter);
  backdrop-filter: var(--app-glass-filter);
}

@media (min-width: 480px) {
  .account-name {
    display: block;
  }
}

@media (min-width: 640px) {
  .header-inner {
    padding-right: 24px;
    padding-left: 24px;
  }

  .desktop-search {
    display: block;
  }

  :deep(.mobile-search-toggle),
  .mobile-search {
    display: none;
  }
}

@media (min-width: 1024px) and (hover: hover) and (pointer: fine) {
  .header-inner {
    height: 64px;
  }

  :deep(.navigation-toggle) {
    display: none;
  }

  .brand {
    min-width: 164px;
  }
}
</style>
