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
  <KunDrawer
    v-model="open"
    title="Koyomi"
    placement="left"
    size="sm"
    :responsive="false"
  >
    <nav class="mobile-navigation" aria-label="移动端主导航">
      <KunButton
        v-for="item in items"
        :key="item.to"
        :href="item.to"
        :color="isActive(item.to) ? 'primary' : 'default'"
        :variant="isActive(item.to) ? 'flat' : 'light'"
        size="lg"
        rounded="lg"
        :full-width="true"
        class-name="mobile-navigation-item"
        :aria-current="isActive(item.to) ? 'page' : undefined"
        @click="open = false"
      >
        <KunIcon :name="item.icon" />
        <span>{{ item.label }}</span>
      </KunButton>
    </nav>
  </KunDrawer>
</template>

<style scoped>
.mobile-navigation {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

:deep(.mobile-navigation-item) {
  justify-content: flex-start;
}
</style>
