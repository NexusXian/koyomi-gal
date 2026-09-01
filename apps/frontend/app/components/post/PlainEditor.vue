<script setup lang="ts">
defineProps<{
  modelValue: string
  disabled?: boolean
}>()

defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>

<template>
  <a-textarea
    :value="modelValue"
    :rows="12"
    :disabled="disabled"
    placeholder="写下你想分享的内容"
    @update:value="$emit('update:modelValue', $event)"
  />
</template>
