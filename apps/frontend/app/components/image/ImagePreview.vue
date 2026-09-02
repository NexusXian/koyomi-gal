<script setup lang="ts">
import type { ImageAsset } from '~/types/image'

withDefaults(
  defineProps<{
    image: ImageAsset | null
    removable?: boolean
    width?: string
    height?: string
  }>(),
  { removable: false, width: '96px', height: '96px' }
)

const emit = defineEmits<{
  remove: []
}>()
</script>

<template>
  <div v-if="image" class="image-preview" :style="{ width, height }">
    <img
      class="preview-img"
      :src="image.url"
      :alt="image.original_name || '已上传图片'"
      loading="lazy"
    >
    <div v-if="removable" class="preview-remove" role="button" @click="emit('remove')">
      <KunIcon name="lucide:x" />
    </div>
  </div>
  <div v-else class="image-preview image-preview-empty" :style="{ width, height }">
    <KunIcon name="lucide:image-off" />
  </div>
</template>

<style scoped>
.image-preview {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid var(--app-glass-border);
  border-radius: var(--radius-kun-md, 8px);
  background: var(--color-content2);
}

.preview-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-preview-empty {
  color: var(--color-default-400);
}

.preview-remove {
  position: absolute;
  top: 4px;
  right: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 50%;
  color: #fff;
  cursor: pointer;
  background: rgb(0 0 0 / 55%);
}

.preview-remove:hover {
  background: rgb(0 0 0 / 75%);
}
</style>
