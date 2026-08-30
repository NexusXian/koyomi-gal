<script setup lang="ts">
interface NavigationItem {
  label: string
  icon: string
  to: string
}

defineProps<{
  items: NavigationItem[]
}>()

const route = useRoute()

function isActive(to: string) {
  return to === '/'
    ? route.path === '/'
    : route.path === to || route.path.startsWith(`${to}/`)
}
</script>

<template>
  <aside class="app-sidebar">
    <nav class="sidebar-navigation" aria-label="主导航">
      <NuxtLink
        v-for="item in items"
        :key="item.to"
        class="navigation-item"
        :class="{ 'navigation-item-active': isActive(item.to) }"
        :to="item.to"
        :aria-current="isActive(item.to) ? 'page' : undefined"
      >
        <KunIcon :name="item.icon" class="navigation-icon" />
        <span>{{ item.label }}</span>
      </NuxtLink>
    </nav>
  </aside>
</template>

<style scoped>
.app-sidebar {
  position: fixed;
  z-index: calc(var(--z-kun-sticky) - 1);
  top: 64px;
  bottom: 0;
  left: 0;
  display: none;
  width: 72px;
  border-right: 1px solid var(--color-kun-border);
  background: color-mix(in srgb, var(--color-content1) 92%, transparent);
  backdrop-filter: blur(16px);
}

.sidebar-navigation {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 8px;
}

.navigation-item {
  position: relative;
  display: flex;
  min-height: 54px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border-radius: var(--radius-kun-md);
  color: var(--color-default-500);
  font-size: 11px;
  transition:
    color var(--kun-dur-fast) var(--ease-kun-standard),
    background-color var(--kun-dur-fast) var(--ease-kun-standard);
}

.navigation-item:hover {
  background: var(--color-content2);
  color: var(--color-foreground);
}

.navigation-item-active {
  background: var(--color-primary-50);
  color: var(--color-primary-600);
  font-weight: 600;
}

.navigation-item-active::before {
  position: absolute;
  top: 12px;
  bottom: 12px;
  left: -8px;
  width: 3px;
  border-radius: 0 4px 4px 0;
  background: var(--color-primary);
  content: '';
}

.navigation-icon {
  font-size: 20px;
}

@media (min-width: 1024px) and (hover: hover) and (pointer: fine) {
  .app-sidebar {
    display: block;
  }
}
</style>
