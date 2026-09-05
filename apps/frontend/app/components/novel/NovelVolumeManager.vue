<script setup lang="ts">
import { message } from 'ant-design-vue'
import {
  createNovelVolume,
  deleteNovelVolume,
  listNovelVolumes,
  reorderNovelVolumes,
  updateNovelVolume
} from '~/api/generated/novels/novels'
import {
  getAdminNovel,
  listAdminNovelVolumes
} from '~/api/generated/admin/admin'
import type {
  DtoVolumeData,
  DtoVolumeSummary
} from '~/api/generated/models'
import { NOVEL_STATUS, domainLabel } from '~/constants/domain'
import type { ImageAsset } from '~/types/image'

const props = defineProps<{
  novelId: number
}>()

const emit = defineEmits<{
  changed: []
}>()

const { has } = usePermissions()
const canReview = computed(() => has('novel:review'))

interface VolumeRow {
  id: number
  volume_number: string
  title: string
  original_title: string
  cover_url: string
  isbn: string
  release_date: string
  summary: string
  status: number
  reject_reason: string
}

const volumes = ref<VolumeRow[]>([])
const loading = ref(false)
const editorOpen = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const reordering = ref(false)

const emptyDraft = (): VolumeRow => ({
  id: 0,
  volume_number: '',
  title: '',
  original_title: '',
  cover_url: '',
  isbn: '',
  release_date: '',
  summary: '',
  status: 0,
  reject_reason: ''
})

const draft = reactive<VolumeRow>(emptyDraft())

async function loadVolumes(): Promise<void> {
  loading.value = true
  try {
    let items: DtoVolumeSummary[] | DtoVolumeData[]
    if (canReview.value) {
      // 未发布卷册只出现在管理端接口里
      const admin = unwrapApiData(
        await listAdminNovelVolumes({ novel_id: props.novelId, limit: 100 })
      )
      items = admin.items ?? []
    } else {
      const publicList = unwrapApiData(
        await listNovelVolumes(props.novelId, { limit: 100 })
      )
      items = publicList.items ?? []
    }
    volumes.value = (items as DtoVolumeData[]).map((item) => ({
      id: item.id ?? 0,
      volume_number:
        item.volume_number !== undefined && item.volume_number !== null
          ? String(item.volume_number)
          : '',
      title: item.title ?? '',
      original_title: item.original_title ?? '',
      cover_url: item.cover_url ?? '',
      isbn: item.isbn ?? '',
      release_date: item.release_date ? item.release_date.slice(0, 10) : '',
      summary: item.summary ?? '',
      status: item.status ?? 1,
      reject_reason: item.reject_reason ?? ''
    }))
  } catch (error) {
    message.error(getApiErrorMessage(error, '加载卷册失败'))
  } finally {
    loading.value = false
  }
}

function openCreate(): void {
  editingId.value = null
  Object.assign(draft, emptyDraft())
  editorOpen.value = true
}

function openEdit(volumeId: number): void {
  const volume = volumes.value.find((item) => item.id === volumeId)
  if (!volume) {
    return
  }
  editingId.value = volume.id
  Object.assign(draft, {
    id: volume.id,
    volume_number: volume.volume_number,
    title: volume.title,
    original_title: volume.original_title,
    cover_url: volume.cover_url,
    isbn: volume.isbn,
    release_date: volume.release_date,
    summary: volume.summary,
    status: volume.status,
    reject_reason: volume.reject_reason
  })
  editorOpen.value = true
}

function onCoverUploaded(asset: ImageAsset): void {
  draft.cover_url = asset.url
}

async function saveVolume(): Promise<void> {
  if (!draft.title.trim() && !draft.volume_number.trim()) {
    message.warning('请填写卷号或标题')
    return
  }
  const volumeNumber = draft.volume_number.trim()
  const payload = {
    volume_number: volumeNumber ? Number(volumeNumber) : undefined,
    title: draft.title.trim() || undefined,
    original_title: draft.original_title.trim() || undefined,
    cover_url: draft.cover_url.trim() || undefined,
    isbn: draft.isbn.trim() || undefined,
    release_date: draft.release_date || undefined,
    summary: draft.summary.trim() || undefined,
    status: canReview.value ? (draft.status as 0 | 1) : 0
  }
  saving.value = true
  try {
    if (editingId.value) {
      await updateNovelVolume(props.novelId, editingId.value, {
        ...payload,
        status: (canReview.value ? draft.status : 1) as 0 | 1 | 2 | 3
      })
      message.success('卷册已更新')
    } else {
      await createNovelVolume(props.novelId, payload)
      message.success('卷册已添加')
    }
    editorOpen.value = false
    await loadVolumes()
    emit('changed')
  } catch (error) {
    message.error(getApiErrorMessage(error, '保存卷册失败'))
  } finally {
    saving.value = false
  }
}

async function removeVolume(volumeId: number): Promise<void> {
  try {
    await deleteNovelVolume(props.novelId, volumeId)
    message.success('卷册已删除')
    await loadVolumes()
    emit('changed')
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除卷册失败'))
  }
}

async function moveVolume(index: number, offset: -1 | 1): Promise<void> {
  const target = index + offset
  if (target < 0 || target >= volumes.value.length) {
    return
  }
  const next = [...volumes.value]
  const [moved] = next.splice(index, 1)
  if (!moved) {
    return
  }
  next.splice(target, 0, moved)
  volumes.value = next
  reordering.value = true
  try {
    await reorderNovelVolumes(props.novelId, {
      ids: next.map((item) => item.id)
    })
    emit('changed')
  } catch (error) {
    message.error(getApiErrorMessage(error, '排序卷册失败'))
    await loadVolumes()
  } finally {
    reordering.value = false
  }
}

onMounted(() => {
  void loadVolumes()
})
</script>

<template>
  <KunCard padding="md">
    <template #header>
      <div class="manager-header">
        <KunHeader name="卷册管理" scale="h3" />
        <a-button type="primary" @click="openCreate">
          <template #icon><KunIcon name="lucide:plus" /></template>
          添加卷册
        </a-button>
      </div>
    </template>

    <a-spin :spinning="loading || reordering">
      <KunNull v-if="volumes.length === 0" message="暂无卷册" />
      <div v-else class="volume-list">
        <div v-for="(volume, index) in volumes" :key="volume.id" class="volume-row">
          <div class="volume-cover">
            <SensitiveImage
              :src="volume.cover_url || undefined"
              :alt="volume.title || `Vol.${volume.volume_number}`"
              :sensitive="false"
            />
          </div>
          <div class="volume-info">
            <div class="volume-title">
              <span v-if="volume.volume_number" class="volume-number">
                Vol.{{ volume.volume_number }}
              </span>
              <span>{{ volume.title || '未命名卷' }}</span>
              <KunChip v-if="volume.status !== 1" size="sm" variant="flat">
                {{ domainLabel(NOVEL_STATUS, volume.status) }}
              </KunChip>
            </div>
            <p v-if="volume.original_title" class="volume-subtitle">
              {{ volume.original_title }}
            </p>
            <p class="volume-meta">
              <span v-if="volume.release_date">{{ volume.release_date }}</span>
              <span v-if="volume.isbn">ISBN {{ volume.isbn }}</span>
            </p>
            <p v-if="volume.status === 2 && volume.reject_reason" class="volume-reject">
              拒绝原因：{{ volume.reject_reason }}
            </p>
          </div>
          <div class="volume-actions">
            <a-button size="small" :disabled="index === 0" @click="moveVolume(index, -1)">
              <template #icon><KunIcon name="lucide:arrow-up" /></template>
            </a-button>
            <a-button
              size="small"
              :disabled="index === volumes.length - 1"
              @click="moveVolume(index, 1)"
            >
              <template #icon><KunIcon name="lucide:arrow-down" /></template>
            </a-button>
            <a-button size="small" @click="openEdit(volume.id)">
              <template #icon><KunIcon name="lucide:pencil" /></template>
            </a-button>
            <a-popconfirm title="确认删除该卷册？" @confirm="removeVolume(volume.id)">
              <a-button size="small" danger>
                <template #icon><KunIcon name="lucide:trash-2" /></template>
              </a-button>
            </a-popconfirm>
          </div>
        </div>
      </div>
    </a-spin>

    <a-modal
      v-model:open="editorOpen"
      :title="editingId ? '编辑卷册' : '添加卷册'"
      :confirm-loading="saving"
      ok-text="保存"
      cancel-text="取消"
      @ok="saveVolume"
    >
      <a-form layout="vertical">
        <div class="editor-grid">
          <a-form-item label="卷号（短篇集 / 外传可留空）">
            <a-input-number
              :value="draft.volume_number ? Number(draft.volume_number) : undefined"
              :min="0"
              :max="9999"
              placeholder="例如：1"
              class="full-width"
              @change="
                (value: number | string | null) => {
                  draft.volume_number = value === null ? '' : String(value)
                }
              "
            />
          </a-form-item>
          <a-form-item label="发售日期">
            <a-input
              v-model:value="draft.release_date"
              placeholder="YYYY-MM-DD"
            />
          </a-form-item>
          <a-form-item label="ISBN">
            <a-input v-model:value="draft.isbn" placeholder="ISBN-10 / ISBN-13" :maxlength="20" />
          </a-form-item>
          <a-form-item v-if="canReview" label="状态（需要审核权限）">
            <a-select
              v-model:value="draft.status"
              :options="
                NOVEL_STATUS.filter(
                  (item) => item.value === 0 || item.value === 1
                ).map((item) => ({ value: item.value, label: item.label }))
              "
            />
          </a-form-item>
        </div>
        <a-form-item label="标题">
          <a-input v-model:value="draft.title" :maxlength="255" />
        </a-form-item>
        <a-form-item label="原文标题">
          <a-input v-model:value="draft.original_title" :maxlength="255" />
        </a-form-item>
        <a-form-item label="封面">
          <div class="image-field">
            <ImageUploader
              v-if="has('image:manage')"
              category="novels"
              :preview-url="draft.cover_url || null"
              width="84px"
              height="112px"
              @success="onCoverUploaded"
              @remove="draft.cover_url = ''"
            />
            <a-input
              v-model:value="draft.cover_url"
              placeholder="https:// 或上传后自动填充"
            />
          </div>
        </a-form-item>
        <a-form-item label="简介">
          <a-textarea v-model:value="draft.summary" :rows="4" />
        </a-form-item>
      </a-form>
    </a-modal>
  </KunCard>
</template>

<style scoped>
.manager-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.volume-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.volume-row {
  display: flex;
  gap: 12px;
  padding: 10px;
  border: 1px solid var(--color-content3);
  border-radius: var(--radius-kun-md);
}

.volume-cover {
  width: 63px;
  flex-shrink: 0;
}

.volume-cover :deep(img) {
  width: 63px;
  height: 84px;
  border-radius: var(--radius-kun-sm);
  object-fit: cover;
}

.volume-info {
  flex: 1;
  min-width: 0;
}

.volume-title {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  font-weight: 600;
}

.volume-number {
  color: var(--color-primary);
}

.volume-subtitle {
  margin: 2px 0 0;
  color: var(--color-default-500);
  font-size: 12px;
}

.volume-meta {
  display: flex;
  gap: 12px;
  margin: 4px 0 0;
  color: var(--color-default-500);
  font-size: 13px;
}

.volume-reject {
  margin: 4px 0 0;
  color: var(--color-danger);
  font-size: 12px;
}

.volume-actions {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.editor-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 16px;
}

.image-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.full-width {
  width: 100%;
}
</style>
