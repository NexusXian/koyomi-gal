<script setup lang="ts">
import type { BackgroundPreset } from '~/types/background'

defineProps<{
  preset: BackgroundPreset
  selected: boolean
}>()

defineEmits<{
  select: []
}>()
</script>

<template>
  <button
    type="button"
    class="preset-card"
    :class="{ 'preset-card-selected': selected }"
    :aria-pressed="selected"
    @click="$emit('select')"
  >
    <img
      class="preset-preview"
      :src="preset.thumbnail ?? preset.src"
      :alt="`${preset.name}背景预览`"
      loading="lazy"
    >
    <span class="preset-name">{{ preset.name }}</span>
    <KunIcon v-if="selected" class="selected-icon" name="lucide:circle-check" />
  </button>
</template>

<style scoped>
.preset-card {
  position: relative;
  overflow: hidden;
  padding: 0;
  border: 2px solid transparent;
  border-radius: var(--radius-kun-lg);
  background: var(--color-content2);
  box-shadow: var(--shadow-kun-sm);
  color: var(--color-foreground);
  cursor: pointer;
  text-align: left;
  transition:
    border-color var(--kun-dur-fast) var(--ease-kun-standard),
    transform var(--kun-dur-fast) var(--ease-kun-standard);
}

.preset-card:hover {
  border-color: color-mix(in srgb, var(--color-primary) 55%, transparent);
  transform: translateY(-1px);
}

.preset-card:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

.preset-card-selected {
  border-color: var(--color-primary);
}

.preset-preview {
  display: block;
  width: 100%;
  aspect-ratio: 16 / 9;
  object-fit: cover;
}

.preset-name {
  display: block;
  overflow: hidden;
  padding: 8px 10px;
  font-size: 13px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.selected-icon {
  position: absolute;
  top: 7px;
  right: 7px;
  padding: 2px;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--color-primary-foreground);
  font-size: 18px;
}
</style>
