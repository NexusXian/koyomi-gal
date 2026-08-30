<script setup lang="ts">
import { storeToRefs } from 'pinia'

defineProps<{
  isDark: boolean
  navigationOpen: boolean
}>()

defineEmits<{
  toggleNavigation: []
  toggleColorMode: []
}>()

const route = useRoute()
const userStore = useUserStore()
const { user, isAuthenticated } = storeToRefs(userStore)
const query = ref(typeof route.query.q === 'string' ? route.query.q : '')
const mobileSearchOpen = ref(false)
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

function submitSearch() {
  const normalizedQuery = query.value.trim()

  navigateTo({
    path: route.path,
    query: {
      ...route.query,
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

        <div class="account" :title="isAuthenticated ? user?.username : '未登录'">
          <KunAvatar
            v-if="isAuthenticated"
            :user="avatarUser"
            :is-navigation="false"
            size="sm"
          />
          <span v-else class="account-fallback">
            <KunIcon name="lucide:user-round" />
          </span>
          <span class="account-name">
            {{ isAuthenticated ? user?.username : '未登录' }}
          </span>
        </div>
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
  border-bottom: 1px solid var(--color-kun-border);
  background: color-mix(in srgb, var(--color-content1) 90%, transparent);
  backdrop-filter: blur(18px);
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
}

.account-fallback {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 auto;
  place-items: center;
  border: 1px solid var(--color-kun-border);
  border-radius: 50%;
  background: var(--color-content2);
  color: var(--color-default-500);
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
  border: 1px solid var(--color-kun-border);
  border-radius: var(--radius-kun-lg);
  background: var(--color-content1);
  box-shadow: var(--shadow-kun-lg);
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
