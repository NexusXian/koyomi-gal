<script setup lang="ts">
defineProps<{
  tabs: { key: string; label: string; icon: string }[]
  active: string
}>()

defineEmits<{ change: [key: string] }>()
</script>

<template>
  <nav class="profile-tabs" aria-label="个人空间内容">
    <button
      v-for="tab in tabs"
      :key="tab.key"
      type="button"
      :class="{ active: active === tab.key }"
      @click="$emit('change', tab.key)"
    >
      <KunIcon :name="tab.icon" />{{ tab.label }}
    </button>
  </nav>
</template>

<style scoped>
.profile-tabs { display: flex; overflow-x: auto; gap: 4px; padding: 6px; border: 1px solid var(--app-glass-border); border-radius: var(--radius-kun-lg); background: var(--app-glass-background); scrollbar-width: none; }
.profile-tabs button { display: inline-flex; flex: 0 0 auto; align-items: center; gap: 6px; padding: 8px 14px; border: 0; border-radius: var(--radius-kun-md); background: transparent; color: var(--color-default-500); cursor: pointer; }
.profile-tabs button:hover, .profile-tabs button.active { background: color-mix(in srgb, var(--color-primary) 12%, transparent); color: var(--color-primary); }
</style>
