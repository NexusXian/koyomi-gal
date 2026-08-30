<script setup lang="ts">
defineOptions({ inheritAttrs: false })

interface NavigationItem {
  label: string
  icon: string
  to: string
}

defineProps<{
  items: NavigationItem[]
}>()

const open = defineModel<boolean>({ required: true })
const route = useRoute()

function isActive(to: string) {
  return to === '/'
    ? route.path === '/'
    : route.path === to || route.path.startsWith(`${to}/`)
}
</script>

<template>
  <div class="mobile-drawer-root">
    <KunDrawer
      v-model="open"
      title="导航"
      placement="left"
      size="sm"
      :responsive="false"
      inner-class-name="navigation-drawer"
    >
      <nav class="mobile-navigation" aria-label="移动端主导航">
        <NuxtLink
          v-for="item in items"
          :key="item.to"
          class="mobile-navigation-item"
          :class="{ 'mobile-navigation-item-active': isActive(item.to) }"
          :to="item.to"
          :aria-current="isActive(item.to) ? 'page' : undefined"
          @click="open = false"
        >
          <span class="mobile-navigation-icon">
            <KunIcon :name="item.icon" />
          </span>
          <span>{{ item.label }}</span>
          <KunIcon name="lucide:chevron-right" class="mobile-navigation-arrow" />
        </NuxtLink>
      </nav>
    </KunDrawer>
  </div>
</template>

<style>
.mobile-drawer-root {
  display: contents;
}

.mobile-navigation {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.mobile-navigation-item {
  display: grid;
  min-height: 52px;
  grid-template-columns: 36px 1fr auto;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--radius-kun-md);
  color: var(--color-default-600);
  transition:
    color var(--kun-dur-fast) var(--ease-kun-standard),
    background-color var(--kun-dur-fast) var(--ease-kun-standard);
}

.mobile-navigation-item:hover {
  background: var(--color-content2);
  color: var(--color-foreground);
}

.mobile-navigation-item-active {
  background: var(--color-primary-50);
  color: var(--color-primary-600);
  font-weight: 600;
}

.mobile-navigation-icon {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: var(--radius-kun-sm);
  background: var(--color-content2);
  font-size: 19px;
}

.mobile-navigation-item-active .mobile-navigation-icon {
  background: var(--color-primary-100);
}

.mobile-navigation-arrow {
  color: var(--color-default-400);
  font-size: 16px;
}
</style>
