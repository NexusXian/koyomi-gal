<script setup lang="ts">
import type { EditorMode } from '~/types/post'

defineProps<{
  modelValue: string
  mode: EditorMode
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

// Cherry Markdown is heavy; only pull its chunk when markdown mode is on.
const MarkdownEditor = defineAsyncComponent(
  () => import('./MarkdownEditor.vue')
)
</script>

<template>
  <PlainEditor
    v-if="mode === 'plain'"
    :model-value="modelValue"
    :disabled="disabled"
    @update:model-value="emit('update:modelValue', $event)"
  />

  <MarkdownEditor
    v-else
    :model-value="modelValue"
    :disabled="disabled"
    @update:model-value="emit('update:modelValue', $event)"
  />
</template>
