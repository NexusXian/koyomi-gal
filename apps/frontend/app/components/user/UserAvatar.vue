<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    avatarUrl?: string | null
    displayName?: string | null
    username?: string | null
    size?: 'sm' | 'md' | 'lg' | 'xl'
  }>(),
  { avatarUrl: '', displayName: '', username: '', size: 'md' }
)

const label = computed(() => props.displayName || props.username || '用户')
const initial = computed(() => label.value.trim().slice(0, 1).toUpperCase() || '?')
</script>

<template>
  <span class="user-avatar" :class="`user-avatar-${size}`" :title="label">
    <img v-if="avatarUrl" :src="avatarUrl" :alt="`${label}的头像`">
    <span v-else aria-hidden="true">{{ initial }}</span>
  </span>
</template>

<style scoped>
.user-avatar {
  display: inline-grid;
  overflow: hidden;
  flex: 0 0 auto;
  place-items: center;
  border: 1px solid var(--app-glass-border);
  border-radius: 50%;
  background: color-mix(in srgb, var(--color-primary) 14%, var(--color-content2));
  color: var(--color-primary-700);
  font-weight: 700;
  line-height: 1;
}

.user-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.user-avatar-sm { width: 32px; height: 32px; font-size: 12px; }
.user-avatar-md { width: 40px; height: 40px; font-size: 15px; }
.user-avatar-lg { width: 48px; height: 48px; font-size: 18px; }
.user-avatar-xl { width: 104px; height: 104px; font-size: 34px; }
</style>
