<script setup lang="ts">
import { storeToRefs } from 'pinia'
const route = useRoute()
const navigationOpen = ref(false)
const colorMode = useCookie<'light' | 'dark'>('koyomi-color-mode', {
  default: () => 'light',
  maxAge: 60 * 60 * 24 * 365
})

const { isAuthenticated } = storeToRefs(useUserStore())
const { load: loadPermissions, hasAny } = usePermissions()

watchEffect(() => {
  if (isAuthenticated.value) {
    void loadPermissions()
  }
})

const navigationItems = computed(() => {
  const items = [
    { label: '首页', icon: 'lucide:house', to: '/' },
    { label: 'galgame', icon: 'lucide:gamepad-2', to: '/galgames' },
    { label: '帖子', icon: 'lucide:message-square-text', to: '/posts' }
  ]

  if (
    isAuthenticated.value &&
    hasAny([
      'galgame:review',
      'resource_report:list',
      'resource:review',
      'galgame:update',
      'role:list',
      'permission:list',
      'role:assign'
    ])
  ) {
    items.push({ label: '管理', icon: 'lucide:shield', to: '/admin/galgames' })
  }

  return items
})

const isDark = computed(() => colorMode.value === 'dark')

function toggleColorMode() {
  colorMode.value = isDark.value ? 'light' : 'dark'
}

watch(
  () => route.path,
  () => {
    navigationOpen.value = false
  }
)

useHead(() => ({
  htmlAttrs: {
    class: isDark.value ? 'kun-dark-mode' : ''
  }
}))
</script>

<template>
  <div class="app-shell">
    <a class="skip-link" href="#main-content">跳转到主要内容</a>

    <AppHeader
      :is-dark="isDark"
      :navigation-open="navigationOpen"
      @toggle-navigation="navigationOpen = true"
      @toggle-color-mode="toggleColorMode"
    />

    <AppSidebar :items="navigationItems" />
    <AppMobileDrawer v-model="navigationOpen" :items="navigationItems" />

    <div class="app-content">
      <main id="main-content" class="app-main" tabindex="-1">
        <slot />
      </main>
    </div>
  </div>
</template>

<style scoped>
.app-shell {
  min-height: 100vh;
  background:
    radial-gradient(circle at 10% 0%, color-mix(in srgb, var(--color-primary) 9%, transparent), transparent 28rem),
    radial-gradient(circle at 90% 12%, color-mix(in srgb, var(--color-secondary) 7%, transparent), transparent 24rem),
    var(--color-background);
}

.skip-link {
  position: fixed;
  z-index: calc(var(--z-kun-modal) + 1);
  top: 8px;
  left: 12px;
  padding: 8px 12px;
  border-radius: var(--radius-kun-md);
  background: var(--color-primary);
  color: var(--color-primary-foreground);
  font-size: 14px;
  font-weight: 600;
  transform: translateY(-160%);
  transition: transform var(--kun-dur-fast) var(--ease-kun-standard);
}

.skip-link:focus {
  transform: translateY(0);
}

.app-main:focus {
  outline: none;
}

.app-content {
  min-height: 100vh;
  padding-top: 60px;
  transition: margin-left var(--kun-dur-base) var(--ease-kun-standard);
}

.app-main {
  width: 100%;
  max-width: 1360px;
  margin: 0 auto;
  padding: 20px 14px 40px;
}

@media (min-width: 640px) {
  .app-main {
    padding-right: 20px;
    padding-left: 20px;
  }
}

@media (min-width: 768px) {
  .app-main {
    padding: 28px 28px 56px;
  }
}

@media (min-width: 1024px) and (hover: hover) and (pointer: fine) {
  .app-content {
    margin-left: 80px;
    padding-top: 64px;
  }
}
</style>
