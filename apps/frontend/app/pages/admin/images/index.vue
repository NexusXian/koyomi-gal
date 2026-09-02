<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { createImageService } from '~/services/image'
import type { ImageAsset, ImageCategory } from '~/types/image'

useSeoMeta({ title: '图片管理 - Koyomi' })

const imageService = createImageService(useNuxtApp().$api)
const { has, load: loadPermissions } = usePermissions()
const canRead = computed(() => has('image:read'))
const canUpload = computed(() => has('image:manage'))
const items = ref<ImageAsset[]>([])
const total = ref(0)
const page = ref(1)
const limit = 20
const loading = ref(false)
const deletingId = ref<number | null>(null)
const uploadCategory = ref<ImageCategory>('galgames')
const uploadedImage = ref<ImageAsset | null>(null)

const filters = reactive({
  category: undefined as ImageCategory | undefined,
  status: undefined as number | undefined,
  userId: '' as string
})

const categoryOptions = [
  { value: 'avatars', label: '头像' },
  { value: 'posts', label: '帖子' },
  { value: 'comments', label: '评论' },
  { value: 'galgames', label: 'Galgame' },
  { value: 'backgrounds', label: '背景' },
  { value: 'banners', label: 'Banner' },
  { value: 'admin', label: '运营' }
]

const managementCategoryOptions: { value: ImageCategory; label: string }[] = [
  { value: 'galgames', label: 'Galgame' },
  { value: 'banners', label: 'Banner' },
  { value: 'admin', label: '运营' }
]

const statusOptions = [
  { value: 0, label: '待上传' },
  { value: 1, label: '已启用' },
  { value: 2, label: '已删除' },
  { value: 3, label: '失败' }
]

const statusLabels: Record<number, string> = {
  0: '待上传',
  1: '已启用',
  2: '已删除',
  3: '失败'
}

function formatSize(size: number): string {
  if (size >= 1024 * 1024) return `${(size / 1024 / 1024).toFixed(2)} MB`
  if (size >= 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${size} B`
}

async function load(): Promise<void> {
  if (!canRead.value) {
    return
  }

  loading.value = true
  try {
    const userId = Number(filters.userId)
    const data = await imageService.listAdminImages({
      page: page.value,
      limit,
      category: filters.category,
      status: filters.status,
      user_id: filters.userId && Number.isInteger(userId) && userId > 0 ? userId : undefined
    })
    items.value = data.items ?? []
    total.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '图片列表加载失败'))
  } finally {
    loading.value = false
  }
}

function applyFilters(): void {
  if (!canRead.value) {
    return
  }
  if (page.value !== 1) {
    page.value = 1
  } else {
    void load()
  }
}

onMounted(async () => {
  await loadPermissions()
  if (canRead.value) {
    await load()
  }
})
watch(page, () => {
  if (canRead.value) {
    void load()
  }
})

watch(uploadCategory, () => {
  uploadedImage.value = null
})

async function handleUploaded(): Promise<void> {
  if (!canRead.value) {
    return
  }
  if (page.value !== 1) {
    page.value = 1
  } else {
    await load()
  }
}

async function remove(image: ImageAsset): Promise<void> {
  deletingId.value = image.id
  try {
    await imageService.deleteAdminImage(image.id)
    message.success('图片已删除')
    if (items.value.length === 1 && page.value > 1) page.value -= 1
    else await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '图片删除失败'))
  } finally {
    deletingId.value = null
  }
}

const columns: TableColumnsType = [
  { title: '预览', key: 'preview', width: 96 },
  { title: '分类', dataIndex: 'category', width: 110 },
  { title: '状态', key: 'status', width: 88 },
  { title: '上传用户', dataIndex: 'user_id', width: 100 },
  { title: '文件', key: 'file', ellipsis: true },
  { title: '尺寸', key: 'dimensions', width: 120 },
  { title: '大小', key: 'size', width: 100 },
  { title: '上传时间', key: 'created_at', width: 170 },
  { title: '操作', key: 'actions', width: 90 }
]
</script>

<template>
  <div>
    <KunCard v-if="canUpload" padding="md" class-name="upload-card">
      <div class="upload-layout">
        <div>
          <KunHeader
            name="上传管理图片"
            description="上传 Galgame、Banner 或运营内容使用的图片。"
            scale="h3"
          />
          <a-select
            v-model:value="uploadCategory"
            :options="managementCategoryOptions"
            class="upload-category"
          />
        </div>
        <ImageUploader
          v-model="uploadedImage"
          :category="uploadCategory"
          width="128px"
          height="96px"
          @success="handleUploaded"
        />
      </div>
    </KunCard>

    <div v-if="canRead" class="table-toolbar">
      <KunHeader name="图片管理" description="查看和删除已上传的图片资源。" scale="h3" />
      <div class="filters">
        <a-select
          v-model:value="filters.category"
          allow-clear
          placeholder="全部分类"
          :options="categoryOptions"
          class="filter-item"
          @change="applyFilters"
        />
        <a-select
          v-model:value="filters.status"
          allow-clear
          placeholder="全部状态"
          :options="statusOptions"
          class="filter-item"
          @change="applyFilters"
        />
        <a-input-search
          v-model:value="filters.userId"
          allow-clear
          placeholder="用户 ID"
          class="filter-item"
          @search="applyFilters"
        />
      </div>
    </div>

    <a-table
      v-if="canRead"
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
      :scroll="{ x: 1100 }"
      @change="(pagination: { current?: number }) => { page = pagination.current ?? 1 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'preview'">
          <img
            v-if="record.status === 1 && record.url"
            class="image-thumb"
            :src="record.url"
            :alt="record.original_name || record.object_key"
            loading="lazy"
          >
          <span v-else>-</span>
        </template>
        <template v-else-if="column.key === 'status'">
          <a-tag :color="record.status === 1 ? 'green' : record.status === 2 ? 'red' : 'default'">
            {{ statusLabels[record.status] ?? record.status }}
          </a-tag>
        </template>
        <template v-else-if="column.key === 'file'">
          <div class="file-cell">
            <span class="file-name">{{ record.original_name || '-' }}</span>
            <span class="file-key">{{ record.object_key }}</span>
          </div>
        </template>
        <template v-else-if="column.key === 'dimensions'">
          {{ record.width && record.height ? `${record.width} × ${record.height}` : '-' }}
        </template>
        <template v-else-if="column.key === 'size'">
          {{ formatSize(record.size) }}
        </template>
        <template v-else-if="column.key === 'created_at'">
          {{ record.created_at ? new Date(record.created_at).toLocaleString('zh-CN', { hour12: false }) : '-' }}
        </template>
        <template v-else-if="column.key === 'actions'">
          <a-popconfirm
            v-if="has('image:delete') && record.status !== 2"
            title="确定删除该图片吗？引用该图片的内容将无法显示。"
            ok-text="删除"
            cancel-text="取消"
            @confirm="remove(record)"
          >
            <a-button size="small" danger :loading="deletingId === record.id">删除</a-button>
          </a-popconfirm>
          <span v-else>-</span>
        </template>
      </template>
    </a-table>

    <KunNull
      v-if="!canRead && canUpload"
      message="当前账号可上传管理图片，但无权查看图片列表。"
    />
  </div>
</template>

<style scoped>
.upload-card {
  margin-bottom: 20px;
}

.upload-layout {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}

.upload-category {
  width: 180px;
  margin-top: 12px;
}

.table-toolbar {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.filters {
  display: flex;
  gap: 10px;
}

.filter-item {
  width: 140px;
}

.image-thumb {
  display: block;
  width: 56px;
  height: 56px;
  border-radius: 6px;
  object-fit: cover;
}

.file-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.file-name {
  font-size: 13px;
}

.file-key {
  overflow: hidden;
  color: var(--color-default-500);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 767px) {
  .upload-layout {
    flex-direction: column;
  }

  .table-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .filters {
    flex-wrap: wrap;
  }
}
</style>
