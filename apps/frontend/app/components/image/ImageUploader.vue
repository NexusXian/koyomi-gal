<script setup lang="ts">
import { message } from 'ant-design-vue'
import {
  ALLOWED_IMAGE_MIME_TYPES,
  IMAGE_CATEGORY_MAX_SIZES,
  useImageUpload
} from '~/composables/useImageUpload'
import type { ImageAsset, ImageCategory } from '~/types/image'

const props = withDefaults(
  defineProps<{
    category: ImageCategory
    modelValue?: ImageAsset | null
    // Fallback preview when only a URL is known (e.g. an existing cover).
    previewUrl?: string | null
    disabled?: boolean
    width?: string
    height?: string
  }>(),
  { modelValue: null, previewUrl: null, disabled: false, width: '96px', height: '96px' }
)

const emit = defineEmits<{
  'update:modelValue': [value: ImageAsset | null]
  success: [value: ImageAsset]
  remove: []
}>()

const { uploadImage } = useImageUpload()
const input = ref<HTMLInputElement | null>(null)
const uploading = ref(false)
const progress = ref(0)
const dragOver = ref(false)

const accept = ALLOWED_IMAGE_MIME_TYPES.join(',')
const maxSize = computed(() => IMAGE_CATEGORY_MAX_SIZES[props.category])

const previewImage = computed<ImageAsset | null>(() => {
  if (props.modelValue) {
    return props.modelValue
  }
  if (props.previewUrl) {
    return { url: props.previewUrl } as ImageAsset
  }
  return null
})

function openFilePicker(): void {
  if (!props.disabled && !uploading.value) {
    input.value?.click()
  }
}

async function handleFile(file: File | undefined): Promise<void> {
  if (!file || props.disabled || uploading.value) {
    return
  }

  uploading.value = true
  progress.value = 0
  try {
    const image = await uploadImage(file, props.category, {
      onProgress: (percentage) => {
        progress.value = percentage
      }
    })
    emit('update:modelValue', image)
    emit('success', image)
    message.success('图片上传成功')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '图片上传失败')
  } finally {
    uploading.value = false
    progress.value = 0
  }
}

function onDrop(event: DragEvent): void {
  dragOver.value = false
  const file = event.dataTransfer?.files?.[0]
  if (file) {
    void handleFile(file)
  }
}

function handleRemove(): void {
  emit('update:modelValue', null)
  emit('remove')
}
</script>

<template>
  <div class="image-uploader">
    <input
      ref="input"
      class="file-input"
      type="file"
      :accept="accept"
      :disabled="disabled"
      @change="(event: Event) => { void handleFile((event.target as HTMLInputElement).files?.[0]); (event.target as HTMLInputElement).value = '' }"
    >

    <ImagePreview
      v-if="previewImage && !uploading"
      :image="previewImage"
      :width="width"
      :height="height"
      removable
      @remove="handleRemove"
    />

    <div
      v-else
      class="upload-trigger"
      :class="{ 'upload-trigger-drag': dragOver }"
      :style="{ width, height }"
      role="button"
      @click="openFilePicker"
      @dragover.prevent="dragOver = true"
      @dragleave.prevent="dragOver = false"
      @drop.prevent="onDrop"
    >
      <template v-if="uploading">
        <a-progress type="circle" :percent="progress" :size="48" />
      </template>
      <template v-else>
        <KunIcon name="lucide:plus" />
        <span class="upload-text">点击或拖拽上传</span>
      </template>
    </div>

    <p class="upload-hint">支持 JPG、PNG、WebP、AVIF、GIF，最大 {{ Math.floor(maxSize / 1024 / 1024) }}MB</p>
  </div>
</template>

<style scoped>
.image-uploader {
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: flex-start;
}

.file-input {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  clip-path: inset(50%);
  white-space: nowrap;
}

.upload-trigger {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
  justify-content: center;
  border: 1px dashed var(--app-glass-border);
  border-radius: var(--radius-kun-md, 8px);
  color: var(--color-default-500);
  cursor: pointer;
  transition: border-color 0.2s;
}

.upload-trigger:hover,
.upload-trigger-drag {
  border-color: var(--color-primary-400, #7aa2f7);
}

.upload-text {
  font-size: 12px;
}

.upload-hint {
  margin: 0;
  color: var(--color-default-500);
  font-size: 12px;
}
</style>
