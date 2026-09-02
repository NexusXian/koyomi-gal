<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { message } from 'ant-design-vue'
import { ALLOWED_BACKGROUND_TYPES, MAX_BACKGROUND_SIZE } from '~/constants/backgrounds'

const backgroundStore = useBackgroundStore()
const userStore = useUserStore()
const { settings } = storeToRefs(backgroundStore)
const input = ref<HTMLInputElement | null>(null)
const uploading = ref(false)

const accept = ALLOWED_BACKGROUND_TYPES.join(',')
const isAuthenticated = computed(() => userStore.isAuthenticated)
const hasCustomImage = computed(() => settings.value.customImageId !== null)
const customImageActive = computed(() => settings.value.source === 'custom')

function openFilePicker(): void {
  input.value?.click()
}

async function handleFileChange(event: Event): Promise<void> {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  target.value = ''

  if (!file) {
    return
  }

  uploading.value = true
  try {
    await backgroundStore.uploadCustomImage(file)
    message.success('自定义背景已应用')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '背景图片上传失败')
  } finally {
    uploading.value = false
  }
}

function useCustomImage(): void {
  backgroundStore.useCustomImage()
}

async function deleteCustomImage(): Promise<void> {
  backgroundStore.deleteCustomImage()
  message.success('自定义背景已删除')
}
</script>

<template>
  <div class="background-uploader">
    <input
      ref="input"
      class="file-input"
      type="file"
      :accept="accept"
      @change="handleFileChange"
    >

    <template v-if="isAuthenticated">
      <a-button block :loading="uploading" @click="openFilePicker">
        <template #icon>
          <KunIcon name="lucide:upload" />
        </template>
        {{ hasCustomImage ? '更换自定义背景' : '上传自定义背景' }}
      </a-button>

      <p class="upload-hint">
        支持 JPG、PNG、WebP、AVIF，最大 {{ MAX_BACKGROUND_SIZE / 1024 / 1024 }}MB
      </p>

      <div v-if="hasCustomImage" class="custom-actions">
        <span class="saved-label">
          <KunIcon name="lucide:image" />
          已保存自定义背景
        </span>
        <div class="action-buttons">
          <a-button v-if="!customImageActive" size="small" @click="useCustomImage">
            使用
          </a-button>
          <a-popconfirm
            title="确定删除自定义背景吗？"
            ok-text="删除"
            cancel-text="取消"
            @confirm="deleteCustomImage"
          >
            <a-button size="small" danger>删除</a-button>
          </a-popconfirm>
        </div>
      </div>
    </template>

    <p v-else class="upload-hint login-hint">
      <KunIcon name="lucide:lock" />
      登录后可上传自定义背景
    </p>
  </div>
</template>

<style scoped>
.background-uploader,
.custom-actions,
.saved-label,
.action-buttons,
.login-hint {
  display: flex;
}

.background-uploader {
  flex-direction: column;
  gap: 10px;
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

.upload-hint {
  margin: 0;
  color: var(--color-default-500);
  font-size: 12px;
}

.login-hint {
  align-items: center;
  gap: 6px;
}

.custom-actions {
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px;
  border: 1px solid var(--app-glass-border);
  border-radius: var(--radius-kun-md, 8px);
  background: var(--color-content2);
}

.saved-label,
.action-buttons {
  align-items: center;
  gap: 6px;
}

.saved-label {
  min-width: 0;
  color: var(--color-default-600);
  font-size: 13px;
}
</style>
