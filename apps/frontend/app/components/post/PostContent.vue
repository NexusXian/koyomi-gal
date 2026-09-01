<script setup lang="ts">
import type { EditorMode } from '~/types/post'
import { renderMarkdown } from '~/utils/markdown'

const props = withDefaults(
  defineProps<{
    content: string
    mode?: EditorMode
  }>(),
  {
    mode: 'plain'
  }
)

// Computed keeps parsing tied to content changes; renderMarkdown owns both
// markdown parsing and HTML sanitization and never returns raw user HTML.
const safeHtml = computed(() =>
  props.mode === 'markdown' ? renderMarkdown(props.content) : ''
)
</script>

<template>
  <div class="post-content-body">
    <div v-if="mode === 'markdown'" class="markdown-body kun-prose" v-html="safeHtml" />
    <div v-else class="post-content-plain">{{ content }}</div>
  </div>
</template>

<style>
.post-content-plain {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

/* v-html content is not scoped; these globals are namespaced on purpose. */
.markdown-body {
  max-width: none;
  margin-inline: 0;
  font-size: 15px;
  line-height: 1.9;
  overflow-wrap: anywhere;
}
</style>
