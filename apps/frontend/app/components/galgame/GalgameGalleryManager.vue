<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { Rule } from 'ant-design-vue/es/form'
import {
  createGalgameGalleryImage,
  deleteGalgameGalleryImage,
  listAdminGalgameGallery,
  reorderGalgameGallery,
  updateGalgameGalleryImage
} from '~/api/generated/admin/admin'
import type {
  DtoGalleryImageData,
  DtoUpdateGalleryImageRequest
} from '~/api/generated/models'
import { GALLERY_IMAGE_TYPES, domainLabel } from '~/constants/domain'

const MAX_GALLERY_IMAGES = 30
const UPLOAD_CONCURRENCY = 3

type GalleryImageType = 0 | 1 | 2 | 3 | 4

interface UploadTask {
  key: number
  name: string
  progress: number
  status: 'uploading' | 'done' | 'error'
  error?: string
}

const props = defineProps<{
  galgameId: number
}>()

const { uploadImage } = useImageUpload()
const fileInput = ref<HTMLInputElement | null>(null)

const items = ref<DtoGalleryImageData[]>([])
const loading = ref(false)
const uploading = ref(false)
const uploadTasks = ref<UploadTask[]>([])
let uploadTaskSeq = 0

const editOpen = ref(false)
const editSaving = ref(false)
const editing = ref<DtoGalleryImageData | null>(null)
const editForm = reactive({ title: '', description: '', image_type: 0 as GalleryImageType, is_spoiler: false })
const editRules: Record<string, Rule[]> = {
  title: [{ max: 255, message: '标题最多 255 个字符', trigger: 'blur' }]
}

const deletingId = ref<number | null>(null)
const orderSaving = ref(false)

const dragIndex = ref<number | null>(null)
const orderDirty = ref(false)

const canAcceptMore = computed(() => items.value.length < MAX_GALLERY_IMAGES)

async function loadGallery(): Promise<void> {
  loading.value = true
  try {
    const data = unwrapApiData(await listAdminGalgameGallery(props.galgameId))
    items.value = data.items ?? []
    orderDirty.value = false
  } catch (error) {
    message.error(getApiErrorMessage(error, '加载游戏画面失败'))
  } finally {
    loading.value = false
  }
}

function pickFiles(): void {
  fileInput.value?.click()
}

function onFilesChange(event: Event): void {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  if (files.length === 0) {
    return
  }

  const remaining = MAX_GALLERY_IMAGES - items.value.length - activeTaskCount()
  if (remaining <= 0) {
    message.warning(`游戏画面最多 ${MAX_GALLERY_IMAGES} 张`)
    return
  }
  const accepted = files.slice(0, remaining)
  if (accepted.length < files.length) {
    message.warning(`游戏画面最多 ${MAX_GALLERY_IMAGES} 张，已忽略多余图片`)
  }
  void runUploadQueue(accepted)
}

function activeTaskCount(): number {
  return uploadTasks.value.filter((task) => task.status === 'uploading').length
}

// Bounded-concurrency upload: presign → PUT → asset → gallery item, 3 at a time.
async function runUploadQueue(files: File[]): Promise<void> {
  if (uploading.value) {
    return
  }
  uploading.value = true
  const queue = [...files]

  const worker = async (): Promise<void> => {
    while (queue.length > 0) {
      const file = queue.shift()
      if (!file) {
        return
      }
      const task = reactive<UploadTask>({
        key: ++uploadTaskSeq,
        name: file.name,
        progress: 0,
        status: 'uploading'
      })
      uploadTasks.value.push(task)
      try {
        const asset = await uploadImage(file, 'galgames', {
          onProgress: (pct) => {
            task.progress = pct
          }
        })
        await createGalgameGalleryImage(props.galgameId, {
          asset_id: asset.id,
          image_type: 0,
          is_spoiler: false
        })
        task.progress = 100
        task.status = 'done'
      } catch (error) {
        task.status = 'error'
        task.error = getApiErrorMessage(error, '上传失败')
      }
    }
  }

  try {
    await Promise.all(
      Array.from({ length: Math.min(UPLOAD_CONCURRENCY, files.length) }, worker)
    )
  } finally {
    uploading.value = false
    setTimeout(() => {
      uploadTasks.value = uploadTasks.value.filter(
        (task) => task.status === 'uploading'
      )
    }, 3000)
    await loadGallery()
  }
}

function openEdit(item?: DtoGalleryImageData): void {
  if (!item?.id) {
    return
  }
  editForm.title = item.title ?? ''
  editForm.description = item.description ?? ''
  editForm.image_type = (item.image_type ?? 0) as GalleryImageType
  editForm.is_spoiler = Boolean(item.is_spoiler)
  editOpen.value = true
}

async function submitEdit(): Promise<void> {
  if (!editing.value?.id) {
    return
  }
  editSaving.value = true
  try {
    const payload: DtoUpdateGalleryImageRequest = {
      title: editForm.title,
      description: editForm.description,
      image_type: editForm.image_type,
      is_spoiler: editForm.is_spoiler
    }
    await updateGalgameGalleryImage(props.galgameId, editing.value.id, payload)
    message.success('游戏画面已更新')
    editOpen.value = false
    await loadGallery()
  } catch (error) {
    message.error(getApiErrorMessage(error, '更新游戏画面失败'))
  } finally {
    editSaving.value = false
  }
}

async function toggleSpoiler(
  item: DtoGalleryImageData | undefined,
  checked: boolean
): Promise<void> {
  if (!item?.id) {
    return
  }
  try {
    await updateGalgameGalleryImage(props.galgameId, item.id, { is_spoiler: checked })
    item.is_spoiler = checked
  } catch (error) {
    message.error(getApiErrorMessage(error, '更新剧透状态失败'))
  }
}

async function removeItem(item: DtoGalleryImageData | undefined): Promise<void> {
  if (!item?.id) {
    return
  }
  deletingId.value = item.id
  try {
    await deleteGalgameGalleryImage(props.galgameId, item.id)
    message.success('游戏画面已删除')
    await loadGallery()
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除游戏画面失败'))
  } finally {
    deletingId.value = null
  }
}

// HTML5 drag reorder: live-preview the move, then submit the full order once.
function onDragStart(index: number): void {
  dragIndex.value = index
}

function onDragOver(index: number): void {
  if (dragIndex.value === null || dragIndex.value === index) {
    return
  }
  const from = dragIndex.value
  const moved = items.value.splice(from, 1)[0]
  if (!moved) {
    dragIndex.value = null
    return
  }
  items.value.splice(index, 0, moved)
  dragIndex.value = index
  orderDirty.value = true
}

async function onDragEnd(): Promise<void> {
  dragIndex.value = null
  if (!orderDirty.value) {
    return
  }
  orderDirty.value = false
  await saveOrder()
}

function moveItem(index: number, offset: -1 | 1): void {
  const target = index + offset
  if (target < 0 || target >= items.value.length) {
    return
  }
  const moved = items.value.splice(index, 1)[0]
  if (!moved) {
    return
  }
  items.value.splice(target, 0, moved)
  void saveOrder()
}

async function saveOrder(): Promise<void> {
  const ids = items.value.map((item) => item.id).filter((id): id is number => Boolean(id))
  if (ids.length !== items.value.length) {
    await loadGallery()
    return
  }
  orderSaving.value = true
  try {
    await reorderGalgameGallery(props.galgameId, { ids })
  } catch (error) {
    message.error(getApiErrorMessage(error, '保存排序失败'))
    await loadGallery()
  } finally {
    orderSaving.value = false
  }
}

onMounted(() => {
  void loadGallery()
})
</script>

<template>
  <KunCard padding="lg" class-name="gallery-manager">
    <div class="section-head-row">
      <KunHeader
        name="游戏画面"
        :description="`共 ${items.length} / ${MAX_GALLERY_IMAGES} 张，拖动调整顺序`"
        scale="h3"
        class="section-heading"
      />
      <KunButton
        color="primary"
        size="sm"
        :disabled="!canAcceptMore || uploading"
        @click="pickFiles"
      >
        <KunIcon name="lucide:upload" />
        {{ uploading ? '上传中…' : '上传图片' }}
      </KunButton>
      <input
        ref="fileInput"
        type="file"
        class="hidden-file-input"
        multiple
        accept="image/jpeg,image/png,image/webp,image/avif,image/gif"
        @change="onFilesChange"
      />
    </div>

    <ul v-if="uploadTasks.length" class="upload-tasks">
      <li v-for="task in uploadTasks" :key="task.key" :class="`task-${task.status}`">
        <span class="task-name" :title="task.name">{{ task.name }}</span>
        <a-progress
          v-if="task.status === 'uploading'"
          :percent="task.progress"
          size="small"
          class="task-progress"
        />
        <span v-else class="task-status">
          {{ task.status === 'done' ? '已完成' : task.error }}
        </span>
      </li>
    </ul>

    <a-spin :spinning="loading">
      <KunNull
        v-if="items.length === 0"
        message="暂无游戏画面，点击上传添加"
      />

      <ul v-else class="manager-list">
        <li
          v-for="(item, index) in items"
          :key="item.id"
          class="manager-item"
          :class="{ 'drag-origin': dragIndex === index }"
          draggable="true"
          @dragstart="onDragStart(index)"
          @dragover.prevent="onDragOver(index)"
          @dragend="onDragEnd"
        >
          <span class="drag-handle" title="拖动排序">
            <KunIcon name="lucide:grip-vertical" />
          </span>

          <button
            type="button"
            class="move-button"
            :disabled="index === 0 || orderSaving"
            aria-label="上移"
            @click="moveItem(index, -1)"
          >
            <KunIcon name="lucide:chevron-up" />
          </button>
          <button
            type="button"
            class="move-button"
            :disabled="index === items.length - 1 || orderSaving"
            aria-label="下移"
            @click="moveItem(index, 1)"
          >
            <KunIcon name="lucide:chevron-down" />
          </button>

          <img
            class="item-thumb"
            :src="item.url"
            :alt="item.title || `游戏画面 ${index + 1}`"
            loading="lazy"
            decoding="async"
          />

          <div class="item-info">
            <p class="item-title">{{ item.title || `画面 ${index + 1}` }}</p>
            <div class="item-meta">
              <a-tag color="default">
                {{ domainLabel(GALLERY_IMAGE_TYPES, item.image_type) }}
              </a-tag>
              <span class="item-dimension">
                {{ item.width && item.height ? `${item.width}×${item.height}` : '' }}
              </span>
            </div>
          </div>

          <div class="item-actions">
            <span class="spoiler-toggle">
              剧透
              <a-switch
                :checked="item.is_spoiler"
                size="small"
                @change="(checked: any) => toggleSpoiler(item, Boolean(checked))"
              />
            </span>
            <a-button size="small" @click="openEdit(item)">
              <template #icon><KunIcon name="lucide:pencil" /></template>
              编辑
            </a-button>
            <a-popconfirm
              :title="`确定从画廊移除「${item.title || `画面 ${index + 1}`}」吗？图片资源本身不会被删除。`"
              ok-text="移除"
              cancel-text="取消"
              @confirm="removeItem(item)"
            >
              <a-button size="small" danger :loading="deletingId === item.id">
                <template #icon><KunIcon name="lucide:trash-2" /></template>
                移除
              </a-button>
            </a-popconfirm>
          </div>
        </li>
      </ul>
    </a-spin>

    <a-modal
      v-model:open="editOpen"
      title="编辑游戏画面"
      :confirm-loading="editSaving"
      ok-text="保存"
      cancel-text="取消"
      @ok="submitEdit"
    >
      <a-form layout="vertical" class="edit-form" :rules="editRules">
        <a-form-item label="标题" name="title">
          <a-input
            v-model:value="editForm.title"
            :maxlength="255"
            placeholder="留空则显示「画面 N」"
          />
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea
            v-model:value="editForm.description"
            :rows="3"
            placeholder="图片描述（可选）"
          />
        </a-form-item>
        <a-form-item label="类型" name="image_type">
          <a-select
            v-model:value="editForm.image_type"
            :options="
              GALLERY_IMAGE_TYPES.map((type) => ({
                value: type.value,
                label: type.label
              }))
            "
          />
        </a-form-item>
        <a-form-item label="剧透" name="is_spoiler">
          <a-switch v-model:checked="editForm.is_spoiler" />
          <span class="spoiler-hint">勾选后普通用户需点击才会显示图片</span>
        </a-form-item>
      </a-form>
    </a-modal>
  </KunCard>
</template>

<style scoped>
.gallery-manager {
  margin-top: 18px;
}

.section-heading {
  margin-bottom: 4px;
}

.section-head-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.hidden-file-input {
  display: none;
}

.upload-tasks {
  margin: 12px 0 0;
  padding: 0;
  list-style: none;
}

.upload-tasks li {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 0;
  font-size: 13px;
}

.task-name {
  flex: 0 0 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-progress {
  flex: 1;
  margin: 0;
}

.task-status {
  color: var(--color-default-500);
}

.task-error .task-status {
  color: var(--color-danger);
}

.manager-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 14px 0 0;
  padding: 0;
  list-style: none;
}

.manager-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  border: 1px solid var(--color-default-200);
  border-radius: var(--radius-kun-lg);
  background: var(--color-background);
}

.manager-item.drag-origin {
  border-color: var(--color-primary);
  opacity: 0.85;
}

.drag-handle {
  display: inline-flex;
  color: var(--color-default-400);
  cursor: grab;
}

.move-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 16px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--color-default-500);
  cursor: pointer;
}

.move-button:disabled {
  color: var(--color-default-200);
  cursor: default;
}

.item-thumb {
  flex: 0 0 auto;
  width: 96px;
  aspect-ratio: 16 / 9;
  border-radius: var(--radius-kun-md);
  object-fit: cover;
}

.item-info {
  flex: 1;
  min-width: 0;
}

.item-title {
  margin: 0;
  overflow: hidden;
  color: var(--color-foreground);
  font-size: 14px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}

.item-dimension {
  color: var(--color-default-400);
  font-size: 12px;
}

.item-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.spoiler-toggle {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--color-default-500);
  font-size: 13px;
}

.edit-form {
  margin-top: 12px;
}

.spoiler-hint {
  margin-left: 10px;
  color: var(--color-default-400);
  font-size: 12px;
}

@media (max-width: 767px) {
  .manager-item {
    flex-wrap: wrap;
  }

  .item-actions {
    width: 100%;
    justify-content: flex-start;
  }
}
</style>
