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
          class-name="desktop-navigation-toggle"
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

        <NuxtLink class="brand" to="/" aria-label="Koyomi Gal 首页">
          <span class="brand-mark">K</span>
          <span class="brand-name">Koyomi</span>
        </NuxtLink>
      </div>

      <form class="desktop-search" role="search" @submit.prevent="submitSearch">
        <KunInput
          v-model="query"
          type="search"
          size="sm"
          rounded="full"
          placeholder="搜索当前内容"
          aria-label="搜索当前内容"
          :is-clearable="true"
        >
          <template #prefix>
            <KunIcon name="lucide:search" class="search-icon" />
          </template>
        </KunInput>
      </form>

      <div class="header-actions">
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

        <div class="account" :title="isAuthenticated ? user?.username : '未登录'">
          <img
            v-if="isAuthenticated && user?.avatar"
            class="account-avatar"
            :src="user.avatar"
            :alt="user.username"
          >
          <span v-else class="account-avatar account-avatar-fallback">
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
        placeholder="搜索当前内容"
        aria-label="搜索当前内容"
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
  min-width: 0;
  border-bottom: 1px solid var(--color-kun-border);
  background: color-mix(in srgb, var(--color-content1) 88%, transparent);
  box-shadow: var(--shadow-kun-sm);
  backdrop-filter: blur(16px);
}

.header-inner {
  display: flex;
  height: 55px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 14px;
}

.header-leading,
.header-actions,
.brand,
.account {
  display: flex;
  align-items: center;
}

.header-leading,
.header-actions {
  gap: 8px;
}

.brand {
  gap: 9px;
  min-width: 0;
  font-size: 18px;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.brand-mark {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-secondary));
  color: var(--color-primary-foreground);
  box-shadow: var(--shadow-kun-sm);
  font-size: 17px;
}

.brand-name {
  display: none;
}

.desktop-search {
  display: none;
  width: min(44vw, 520px);
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

.account-avatar {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 auto;
  place-items: center;
  border: 1px solid var(--color-kun-border);
  border-radius: 50%;
  object-fit: cover;
}

.account-avatar-fallback {
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
  .brand-name,
  .account-name {
    display: block;
  }
}

@media (min-width: 640px) {
  .header-inner {
    padding-right: 20px;
    padding-left: 20px;
  }

  .desktop-search {
    display: block;
  }

  :deep(.mobile-search-toggle) {
    display: none;
  }

  .mobile-search {
    display: none;
  }
}

@media (min-width: 1024px) and (hover: hover) and (pointer: fine) {
  .header-inner {
    height: 63px;
    padding-right: 24px;
    padding-left: 20px;
  }

  :deep(.desktop-navigation-toggle) {
    display: none;
  }

  .brand {
    min-width: 152px;
  }
}
</style>
