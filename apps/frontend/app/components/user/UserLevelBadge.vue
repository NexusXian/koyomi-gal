<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    level: number
    name: string
    color?: string | null
    iconUrl?: string | null
    size?: 'sm' | 'md'
  }>(),
  { color: '', iconUrl: '', size: 'sm' }
)

const tooltip = computed(() => `LV${props.level} · ${props.name}`)

const badgeStyle = computed(() => {
  if (!props.color) {
    return undefined
  }
  return {
    color: props.color,
    borderColor: `color-mix(in srgb, ${props.color} 45%, transparent)`,
    backgroundColor: `color-mix(in srgb, ${props.color} 12%, transparent)`
  }
})
</script>

<template>
  <a-tooltip :title="tooltip" placement="top">
    <span class="user-level-badge" :class="`user-level-badge-${size}`" :style="badgeStyle">
      <img v-if="iconUrl" :src="iconUrl" :alt="tooltip" class="user-level-badge-icon">
      <template v-else>
        <span class="user-level-badge-level">LV{{ level }}</span>
        <span v-if="size === 'md'" class="user-level-badge-name">{{ name }}</span>
      </template>
    </span>
  </a-tooltip>
</template>

<style scoped>
.user-level-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex: 0 0 auto;
  padding: 0 7px;
  border: 1px solid var(--app-glass-border);
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-primary) 10%, transparent);
  color: var(--color-primary-700);
  font-weight: 700;
  line-height: 1.6;
  white-space: nowrap;
}

.user-level-badge-sm {
  height: 20px;
  font-size: 11px;
}

.user-level-badge-md {
  height: 24px;
  font-size: 13px;
}

.user-level-badge-icon {
  width: 100%;
  height: 100%;
  border-radius: 999px;
  object-fit: cover;
}

.user-level-badge-name {
  font-weight: 600;
}
</style>
