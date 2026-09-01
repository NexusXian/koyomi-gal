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
      <KunTooltip
        v-for="item in items"
        :key="item.to"
        :text="item.label"
        position="right"
        :show-arrow="true"
      >
        <KunButton
          :href="item.to"
          :color="isActive(item.to) ? 'primary' : 'default'"
          :variant="isActive(item.to) ? 'flat' : 'light'"
          size="lg"
          rounded="lg"
          :is-icon-only="true"
          :aria-label="item.label"
          :aria-current="isActive(item.to) ? 'page' : undefined"
        >
          <KunIcon :name="item.icon" class="navigation-icon" />
        </KunButton>
      </KunTooltip>
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
  width: 80px;
  border-right: 1px solid var(--app-glass-border);
  background: var(--app-glass-background);
  -webkit-backdrop-filter: var(--app-glass-filter);
  backdrop-filter: var(--app-glass-filter);
}

.sidebar-navigation {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 16px 12px;
}

.navigation-icon {
  font-size: 21px;
}

@media (min-width: 1024px) and (hover: hover) and (pointer: fine) {
  .app-sidebar {
    display: block;
  }
}
</style>
