<script setup lang="ts">
import type Cherry from 'cherry-markdown'

const props = defineProps<{
  modelValue: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const container = ref<HTMLElement | null>(null)

let cherry: Cherry | null = null
// Guards against feedback loops between Cherry's afterChange and the external
// modelValue watcher.
let applyingExternal = false
let disposed = false

onMounted(async () => {
  if (!container.value) {
    return
  }

  // Cherry is a browser-only editor; load it (and its stylesheet) lazily so it
  // stays out of the initial bundle and is never evaluated during SSR.
  const [{ default: CherryConstructor }] = await Promise.all([
    import('cherry-markdown'),
    import('cherry-markdown/dist/cherry-markdown.css')
  ])

  if (disposed || !container.value) {
    return
  }

  cherry = new CherryConstructor({
    el: container.value,
    value: props.modelValue,
    locale: 'zh_CN',
    isPreviewOnly: false,
    editor: {
      defaultModel: 'edit&preview',
      height: '460px'
    },
    toolbars: {
      toolbar: [
        'bold',
        'italic',
        'strikethrough',
        'header',
        '|',
        'ul',
        'ol',
        'quote',
        '|',
        'link',
        'image',
        'hr',
        '|',
        'code',
        'codeBlock',
        'table',
        '|',
        'togglePreview'
      ]
    },
    callback: {
      afterChange: (markdown: string) => {
        if (applyingExternal) {
          return
        }
        emit('update:modelValue', markdown)
      }
    }
  })
  cherry.editor.setReadOnly(Boolean(props.disabled))
})

watch(
  () => props.modelValue,
  (value) => {
    if (!cherry) {
      return
    }
    if (cherry.getMarkdown() === value) {
      return
    }
    applyingExternal = true
    try {
      cherry.setMarkdown(value ?? '', true)
    } finally {
      applyingExternal = false
    }
  }
)

watch(
  () => props.disabled,
  (disabled) => cherry?.editor.setReadOnly(Boolean(disabled))
)

onBeforeUnmount(() => {
  disposed = true
  cherry?.destroy()
  cherry = null
})
</script>

<template>
  <div class="markdown-editor" :class="{ 'markdown-editor-disabled': disabled }">
    <div ref="container" class="markdown-editor-container" />
  </div>
</template>

<style scoped>
.markdown-editor {
  width: 100%;
}

.markdown-editor-container {
  width: 100%;
  border: 1px solid var(--color-default-200);
  border-radius: var(--radius-kun-md, 8px);
  overflow: hidden;
}

.markdown-editor-disabled {
  opacity: 0.6;
  pointer-events: none;
}
</style>
