<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { createBackgroundPresetService } from '~/services/backgroundPreset'
import type {
  BackgroundPresetData,
  BackgroundPresetPayload
} from '~/types/background'
import type { ImageAsset } from '~/types/image'

useSeoMeta({ title: '背景预设管理 - Koyomi' })

const presetService = createBackgroundPresetService(useNuxtApp().$api)
const { has } = usePermissions()
const canUploadImages = computed(() => has('image:manage'))
const items = ref<BackgroundPresetData[]>([])
const total = ref(0)
const page = ref(1)
const limit = 20
const loading = ref(false)
const saving = ref(false)
const updatingId = ref<number | null>(null)
const modalOpen = ref(false)
const editing = ref<BackgroundPresetData | null>(null)

const emptyForm = () => ({
  name: '',
  image_url: '',
  sort_order: 0,
  is_active: true
})
const formState = reactive(emptyForm())

function presetPayload(
  source: BackgroundPresetData | typeof formState
): BackgroundPresetPayload {
  return {
    name: source.name.trim(),
    image_url: source.image_url.trim(),
    sort_order: source.sort_order ?? 0,
    is_active: source.is_active
  }
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = await presetService.listAdminPresets({ page: page.value, limit })
    items.value = data.items ?? []
    total.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '背景预设列表加载失败'))
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())
watch(page, () => void load())

function openCreate(): void {
  editing.value = null
  Object.assign(formState, emptyForm())
  modalOpen.value = true
}

function openEdit(preset: BackgroundPresetData): void {
  editing.value = preset
  Object.assign(formState, {
    name: preset.name ?? '',
    image_url: preset.image_url ?? '',
    sort_order: preset.sort_order ?? 0,
    is_active: preset.is_active
  })
  modalOpen.value = true
}

async function submit(): Promise<void> {
  if (!formState.name.trim() || !formState.image_url.trim()) {
    message.warning('请填写预设名称和图片地址')
    return
  }

  saving.value = true
  try {
    const payload = presetPayload(formState)
    if (editing.value) {
      await presetService.updatePreset(editing.value.id, payload)
    } else {
      await presetService.createPreset(payload)
    }
    message.success(editing.value ? '背景预设已更新' : '背景预设已创建')
    modalOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '背景预设保存失败'))
  } finally {
    saving.value = false
  }
}

async function toggleActive(
  preset: BackgroundPresetData,
  active: boolean
): Promise<void> {
  updatingId.value = preset.id
  try {
    await presetService.updatePreset(preset.id, {
      ...presetPayload(preset),
      is_active: active
    })
    message.success(active ? '背景预设已启用' : '背景预设已停用')
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '状态更新失败'))
  } finally {
    updatingId.value = null
  }
}

async function remove(preset: BackgroundPresetData): Promise<void> {
  try {
    await presetService.deletePreset(preset.id)
    message.success('背景预设已删除')
    if (items.value.length === 1 && page.value > 1) page.value -= 1
    else await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '背景预设删除失败'))
  }
}

function onPresetImageUploaded(asset: ImageAsset): void {
  formState.image_url = asset.url
  message.success('背景预设图片已上传')
}

const columns: TableColumnsType = [
  { title: '排序', dataIndex: 'sort_order', width: 74 },
  { title: '预览', key: 'preview', width: 140 },
  { title: '名称', dataIndex: 'name', ellipsis: true },
  { title: '标识', dataIndex: 'key', width: 150, ellipsis: true },
  { title: '启用', dataIndex: 'is_active', width: 82 },
  { title: '操作', key: 'actions', width: 150 }
]
</script>

<template>
  <div>
    <div class="table-toolbar">
      <KunHeader
        name="背景预设"
        description="管理用户个性化背景的预设图片，停用后不再出现在选择列表中。"
        scale="h3"
      />
      <a-button
        v-if="has('background_preset:create')"
        type="primary"
        @click="openCreate"
      >
        新建预设
      </a-button>
    </div>

    <a-table
      :columns="columns"
      :data-source="items"
      :loading="loading"
      :pagination="{
        current: page,
        pageSize: limit,
        total,
        showSizeChanger: false,
        showTotal: (count: number) => `共 ${count} 条`
      }"
      row-key="id"
      :scroll="{ x: 760 }"
      @change="(pagination: { current?: number }) => { page = pagination.current ?? 1 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'preview'">
          <img
            v-if="record.image_url"
            class="preset-preview"
            :src="record.image_url"
            :alt="record.name"
          />
          <span v-else>-</span>
        </template>
        <template v-else-if="column.dataIndex === 'is_active'">
          <a-switch
            :checked="record.is_active"
            :loading="updatingId === record.id"
            :disabled="!has('background_preset:update')"
            @change="(value: boolean) => toggleActive(record, value)"
          />
        </template>
        <template v-else-if="column.key === 'actions'">
          <div class="table-actions">
            <a-button
              v-if="has('background_preset:update')"
              size="small"
              @click="openEdit(record)"
            >
              编辑
            </a-button>
            <a-popconfirm
              v-if="has('background_preset:delete')"
              :title="`确定删除「${record.name}」吗？已选用该预设的用户将失去背景。`"
              ok-text="删除"
              cancel-text="取消"
              @confirm="remove(record)"
            >
              <a-button size="small" danger>删除</a-button>
            </a-popconfirm>
          </div>
        </template>
      </template>
    </a-table>

    <a-modal
      v-model:open="modalOpen"
      :title="editing ? '编辑背景预设' : '新建背景预设'"
      :confirm-loading="saving"
      width="560px"
      ok-text="保存"
      cancel-text="取消"
      @ok="submit"
    >
      <a-form layout="vertical">
        <a-form-item label="名称" required>
          <a-input v-model:value="formState.name" :maxlength="255" />
        </a-form-item>
        <a-form-item label="图片" required>
          <div class="preset-image-field">
            <ImageUploader
              v-if="canUploadImages"
              category="admin"
              :preview-url="formState.image_url || null"
              width="160px"
              height="90px"
              @success="onPresetImageUploaded"
              @remove="formState.image_url = ''"
            />
            <a-input
              v-model:value="formState.image_url"
              placeholder="https:// 或上传后自动填充"
            />
          </div>
        </a-form-item>
        <div class="form-grid">
          <a-form-item label="排序">
            <a-input-number
              v-model:value="formState.sort_order"
              :min="-9999"
              :max="9999"
              class="full-width"
            />
          </a-form-item>
          <a-form-item label="启用">
            <a-switch v-model:checked="formState.is_active" />
          </a-form-item>
        </div>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.table-toolbar { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; margin-bottom: 16px; }
.preset-preview { display: block; width: 120px; height: 68px; border-radius: 6px; object-fit: cover; }
.table-actions { display: flex; gap: 6px; }
.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.preset-image-field { display: flex; flex-direction: column; gap: 8px; }
.full-width { width: 100%; }
@media (max-width: 639px) { .table-toolbar { align-items: stretch; flex-direction: column; } .form-grid { grid-template-columns: 1fr; gap: 0; } }
</style>
