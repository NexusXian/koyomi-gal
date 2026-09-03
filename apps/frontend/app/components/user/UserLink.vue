<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    username?: string | null
    displayName?: string | null
    userId?: number | null
    popover?: boolean
  }>(),
  { username: '', displayName: '', userId: null, popover: true }
)

const desktopHover = ref(false)
const popoverOpen = ref(false)
const label = computed(() =>
  props.displayName || props.username || (props.userId ? `用户 #${props.userId}` : '未知用户')
)

onMounted(() => {
  desktopHover.value = window.matchMedia('(hover: hover) and (pointer: fine)').matches
})
</script>

<template>
  <a-popover
    v-if="username && popover && desktopHover"
    v-model:open="popoverOpen"
    trigger="hover"
    placement="topLeft"
  >
    <template #content>
      <UserPopover :username="username" :active="popoverOpen" />
    </template>
    <NuxtLink class="user-link" :to="`/user/${encodeURIComponent(username)}`" @click.stop>
      <slot>{{ label }}</slot>
    </NuxtLink>
  </a-popover>
  <NuxtLink
    v-else-if="username"
    class="user-link"
    :to="`/user/${encodeURIComponent(username)}`"
    @click.stop
  >
    <slot>{{ label }}</slot>
  </NuxtLink>
  <span v-else class="user-label"><slot>{{ label }}</slot></span>
</template>

<style scoped>
.user-link, .user-label { display: inline-flex; min-width: 0; align-items: center; gap: 6px; }
.user-link { color: inherit; }
.user-link:hover { color: var(--color-primary); }
</style>
