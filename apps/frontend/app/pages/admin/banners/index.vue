<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { createContentService } from '~/services/content'
import type { BannerAdmin, BannerPayload } from '~/types/content'
import type { ImageAsset } from '~/types/image'

useSeoMeta({ title: '轮播图管理 - Koyomi' })

const contentService = createContentService(useNuxtApp().$api)
const { has } = usePermissions()
const canUploadImages = computed(() => has('image:manage'))
const items = ref<BannerAdmin[]>([])
const total = ref(0)
const page = ref(1)
const limit = 20
const loading = ref(false)
const saving = ref(false)
const updatingId = ref<number | null>(null)
const modalOpen = ref(false)
const editing = ref<BannerAdmin | null>(null)

const emptyForm = () => ({
  title: '',
  subtitle: '',
  image_url: '',
  link_type: 'none',
  link_value: '',
  sort_order: 0,
  is_active: true,
  start_at: '',
  end_at: ''
})
const formState = reactive(emptyForm())

function toDateInput(value?: string | null): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000)
  return local.toISOString().slice(0, 16)
}

function toIsoDate(value: string): string | null {
  if (!value) return null
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? null : date.toISOString()
}

function bannerPayload(source: BannerAdmin | typeof formState): BannerPayload {
  return {
    title: source.title.trim(),
    subtitle: source.subtitle?.trim() || null,
    image_url: source.image_url?.trim() || '',
    link_type: source.link_type?.trim() || 'none',
    link_value: source.link_type === 'none' ? null : source.link_value?.trim() || null,
    sort_order: source.sort_order ?? 0,
    is_active: source.is_active,
    start_at: source === formState ? toIsoDate(formState.start_at) : source.start_at || null,
    end_at: source === formState ? toIsoDate(formState.end_at) : source.end_at || null
  }
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = await contentService.listAdminBanners({ page: page.value, limit })
    items.value = data.items ?? []
    total.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '轮播图列表加载失败'))
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

function openEdit(banner: BannerAdmin): void {
  editing.value = banner
  Object.assign(formState, {
    title: banner.title ?? '',
    subtitle: banner.subtitle ?? '',
    image_url: banner.image_url ?? '',
    link_type: banner.link_type ?? 'none',
    link_value: banner.link_value ?? '',
    sort_order: banner.sort_order ?? 0,
    is_active: banner.is_active,
    start_at: toDateInput(banner.start_at),
    end_at: toDateInput(banner.end_at)
  })
  modalOpen.value = true
}

async function submit(): Promise<void> {
  if (!formState.title.trim() || !formState.image_url.trim()) {
    message.warning('请填写轮播图标题和图片地址')
    return
  }
  if (formState.start_at && formState.end_at && new Date(formState.start_at) > new Date(formState.end_at)) {
    message.warning('结束时间不能早于开始时间')
    return
  }

  saving.value = true
  try {
    const payload = bannerPayload(formState)
    if (editing.value) {
      await contentService.updateBanner(editing.value.id, payload)
    } else {
      await contentService.createBanner(payload)
    }
    message.success(editing.value ? '轮播图已更新' : '轮播图已创建')
    modalOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '轮播图保存失败'))
  } finally {
    saving.value = false
  }
}

async function toggleActive(banner: BannerAdmin, active: boolean): Promise<void> {
  updatingId.value = banner.id
  try {
    await contentService.updateBanner(banner.id, {
      ...bannerPayload(banner),
      is_active: active
    })
    message.success(active ? '轮播图已启用' : '轮播图已停用')
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '状态更新失败'))
  } finally {
    updatingId.value = null
  }
}

async function remove(banner: BannerAdmin): Promise<void> {
  try {
    await contentService.deleteBanner(banner.id)
    message.success('轮播图已删除')
    if (items.value.length === 1 && page.value > 1) page.value -= 1
    else await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '轮播图删除失败'))
  }
}

function onBannerImageUploaded(asset: ImageAsset): void {
  formState.image_url = asset.url
  message.success('Banner 图片已上传')
}

const columns: TableColumnsType = [
  { title: '排序', dataIndex: 'sort_order', width: 74 },
  { title: '预览', key: 'preview', width: 112 },
  { title: '标题', dataIndex: 'title', ellipsis: true },
  { title: '跳转', key: 'link', width: 150, ellipsis: true },
  { title: '排期', key: 'schedule', width: 190 },
  { title: '启用', dataIndex: 'is_active', width: 82 },
  { title: '操作', key: 'actions', width: 150 }
]
</script>

<template>
  <div>
    <div class="table-toolbar">
      <KunHeader name="首页轮播图" description="管理展示顺序、跳转目标和生效时间。" scale="h3" />
      <a-button v-if="has('banner:create')" type="primary" @click="openCreate">新建轮播图</a-button>
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
      :scroll="{ x: 980 }"
      @change="(pagination: { current?: number }) => { page = pagination.current ?? 1 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'preview'">
          <img v-if="record.image_url" class="banner-preview" :src="record.image_url" :alt="record.title" />
          <span v-else>-</span>
        </template>
        <template v-else-if="column.key === 'link'">
          {{ record.link_type ? `${record.link_type}: ${record.link_value || '-'}` : '-' }}
        </template>
        <template v-else-if="column.key === 'schedule'">
          <div class="schedule-text">
            <span>{{ record.start_at ? new Date(record.start_at).toLocaleString('zh-CN', { hour12: false }) : '立即生效' }}</span>
            <span>至 {{ record.end_at ? new Date(record.end_at).toLocaleString('zh-CN', { hour12: false }) : '长期' }}</span>
          </div>
        </template>
        <template v-else-if="column.dataIndex === 'is_active'">
          <a-switch
            :checked="record.is_active"
            :loading="updatingId === record.id"
            :disabled="!has('banner:update')"
            @change="(value: boolean) => toggleActive(record, value)"
          />
        </template>
        <template v-else-if="column.key === 'actions'">
          <div class="table-actions">
            <a-button v-if="has('banner:update')" size="small" @click="openEdit(record)">编辑</a-button>
            <a-popconfirm
              v-if="has('banner:delete')"
              :title="`确定删除「${record.title}」吗？`"
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
      :title="editing ? '编辑轮播图' : '新建轮播图'"
      :confirm-loading="saving"
      width="720px"
      ok-text="保存"
      cancel-text="取消"
      @ok="submit"
    >
      <a-form layout="vertical">
        <a-form-item label="标题" required><a-input v-model:value="formState.title" :maxlength="255" /></a-form-item>
        <a-form-item label="副标题"><a-textarea v-model:value="formState.subtitle" :rows="2" :maxlength="500" /></a-form-item>
        <a-form-item label="图片" required>
          <div class="banner-image-field">
            <ImageUploader
              v-if="canUploadImages"
              category="banners"
              :preview-url="formState.image_url || null"
              width="160px"
              height="72px"
              @success="onBannerImageUploaded"
              @remove="formState.image_url = ''"
            />
            <a-input v-model:value="formState.image_url" placeholder="https:// 或上传后自动填充" />
          </div>
        </a-form-item>
        <div class="form-grid">
          <a-form-item label="跳转类型">
            <a-select
              v-model:value="formState.link_type"
              :options="[
                { value: 'none', label: '不跳转' },
                { value: 'galgame', label: 'Galgame' },
                { value: 'post', label: '帖子' },
                { value: 'news', label: '文章' },
                { value: 'url', label: '外部链接' }
              ]"
            />
          </a-form-item>
          <a-form-item label="跳转值"><a-input v-model:value="formState.link_value" :disabled="formState.link_type === 'none'" :placeholder="formState.link_type === 'url' ? 'https://' : '资源 ID'" /></a-form-item>
        </div>
        <div class="form-grid">
          <a-form-item label="排序"><a-input-number v-model:value="formState.sort_order" :min="-9999" :max="9999" class="full-width" /></a-form-item>
          <a-form-item label="启用"><a-switch v-model:checked="formState.is_active" /></a-form-item>
        </div>
        <div class="form-grid">
          <a-form-item label="开始时间"><a-input v-model:value="formState.start_at" type="datetime-local" /></a-form-item>
          <a-form-item label="结束时间"><a-input v-model:value="formState.end_at" type="datetime-local" /></a-form-item>
        </div>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.table-toolbar { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; margin-bottom: 16px; }
.banner-preview { display: block; width: 88px; height: 40px; border-radius: 6px; object-fit: cover; }
.schedule-text { display: flex; flex-direction: column; gap: 2px; color: var(--color-default-500); font-size: 12px; }
.table-actions { display: flex; gap: 6px; }
.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.banner-image-field { display: flex; flex-direction: column; gap: 8px; }
.full-width { width: 100%; }
@media (max-width: 639px) { .table-toolbar { align-items: stretch; flex-direction: column; } .form-grid { grid-template-columns: 1fr; gap: 0; } }
</style>
