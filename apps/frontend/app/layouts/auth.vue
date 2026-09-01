<script setup lang="ts">
const colorMode = useCookie<'light' | 'dark'>('koyomi-color-mode', {
  default: () => 'light',
  maxAge: 60 * 60 * 24 * 365
})

const isDark = computed(() => colorMode.value === 'dark')

function toggleColorMode() {
  colorMode.value = isDark.value ? 'light' : 'dark'
}

useHead(() => ({
  htmlAttrs: {
    class: isDark.value ? 'kun-dark-mode' : ''
  }
}))
</script>

<template>
  <div class="auth-shell">
    <div class="theme-toggle">
      <KunButton
        color="primary"
        variant="light"
        size="sm"
        :is-icon-only="true"
        :aria-label="isDark ? '切换到浅色模式' : '切换到深色模式'"
        @click="toggleColorMode"
      >
        <KunIcon :name="isDark ? 'lucide:sun' : 'lucide:moon'" />
      </KunButton>
    </div>

    <main class="auth-content">
      <slot />
    </main>
  </div>
</template>

<style scoped>
.auth-shell {
  position: relative;
  display: flex;
  min-height: 100vh;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: transparent;
}

.theme-toggle {
  position: absolute;
  z-index: 1;
  top: 16px;
  right: 16px;
}

.theme-toggle :deep(button) {
  font-size: 20px;
}

.auth-content {
  display: flex;
  width: 100%;
  justify-content: center;
}
</style>
