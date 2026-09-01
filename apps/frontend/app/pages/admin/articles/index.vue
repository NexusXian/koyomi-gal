<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { createContentService } from '~/services/content'
import type { Article, ArticlePayload } from '~/types/content'

useSeoMeta({ title: '文章管理 - Koyomi' })

const contentService = createContentService(useNuxtApp().$api)
const { has } = usePermissions()
const items = ref<Article[]>([])
const total = ref(0)
const page = ref(1)
const limit = 20
const loading = ref(false)
const saving = ref(false)
const updatingId = ref<number | null>(null)
const modalOpen = ref(false)
const editing = ref<Article | null>(null)

const emptyForm = () => ({
  title: '',
  summary: '',
  content: '',
  cover_url: '',
  type: 'news',
  is_pinned: false,
  is_published: false,
  published_at: ''
})
const formState = reactive(emptyForm())

function isScheduled(article: Article): boolean {
  return Boolean(article.is_published && article.published_at && new Date(article.published_at).getTime() > Date.now())
}

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

function articlePayload(source: Article | typeof formState): ArticlePayload {
  const publishedAt = source === formState
    ? editing.value && formState.published_at === toDateInput(editing.value.published_at)
      ? editing.value.published_at || null
      : toIsoDate(formState.published_at)
    : source.published_at || null

  return {
    title: source.title.trim(),
    summary: source.summary?.trim() || null,
    content: source.content?.trim() || '',
    cover_url: source.cover_url?.trim() || null,
    type: source.type,
    is_pinned: source.is_pinned,
    is_published: source.is_published ?? false,
    published_at: publishedAt
  }
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = await contentService.listAdminArticles({ page: page.value, limit })
    items.value = data.items ?? []
    total.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '文章列表加载失败'))
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

function openEdit(article: Article): void {
  editing.value = article
  Object.assign(formState, {
    title: article.title ?? '',
    summary: article.summary ?? '',
    content: article.content ?? '',
    cover_url: article.cover_url ?? '',
    type: article.type || 'news',
    is_pinned: article.is_pinned,
    is_published: article.is_published ?? false,
    published_at: toDateInput(article.published_at)
  })
  modalOpen.value = true
}

async function submit(): Promise<void> {
  if (!formState.title.trim() || !formState.content.trim()) {
    message.warning('请填写标题和正文')
    return
  }
  saving.value = true
  try {
    const payload = articlePayload(formState)
    if (editing.value) await contentService.updateArticle(editing.value.id, payload)
    else await contentService.createArticle(payload)
    message.success(editing.value ? '文章已更新' : '文章已创建')
    modalOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '文章保存失败'))
  } finally {
    saving.value = false
  }
}

async function updateState(article: Article, changes: Partial<ArticlePayload>, success: string): Promise<void> {
  updatingId.value = article.id
  try {
    await contentService.updateArticle(article.id, { ...articlePayload(article), ...changes })
    message.success(success)
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '文章状态更新失败'))
  } finally {
    updatingId.value = null
  }
}

async function remove(article: Article): Promise<void> {
  try {
    await contentService.deleteArticle(article.id)
    message.success('文章已删除')
    if (items.value.length === 1 && page.value > 1) page.value -= 1
    else await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '文章删除失败'))
  }
}

const columns: TableColumnsType = [
  { title: 'ID', dataIndex: 'id', width: 70 },
  { title: '标题', dataIndex: 'title', ellipsis: true },
  { title: '类型', dataIndex: 'type', width: 90 },
  { title: '状态', key: 'status', width: 135 },
  { title: '发布时间', dataIndex: 'published_at', width: 180 },
  { title: '操作', key: 'actions', width: 280 }
]
</script>

<template>
  <div>
    <div class="table-toolbar">
      <KunHeader name="文章内容" description="发布公告、资讯与社区专题内容。" scale="h3" />
      <a-button v-if="has('article:create')" type="primary" @click="openCreate">新建文章</a-button>
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
      :scroll="{ x: 1000 }"
      @change="(pagination: { current?: number }) => { page = pagination.current ?? 1 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'type'">
          <a-tag>{{ record.type === 'announcement' ? '公告' : record.type === 'news' ? '资讯' : record.type }}</a-tag>
        </template>
        <template v-else-if="column.key === 'status'">
          <div class="status-tags">
            <a-tag :color="isScheduled(record) ? 'processing' : record.is_published ? 'success' : 'default'">
              {{ isScheduled(record) ? '待发布' : record.is_published ? '已发布' : '草稿' }}
            </a-tag>
            <a-tag v-if="record.is_pinned" color="processing">置顶</a-tag>
          </div>
        </template>
        <template v-else-if="column.dataIndex === 'published_at'">
          {{ record.published_at ? new Date(record.published_at).toLocaleString('zh-CN', { hour12: false }) : '-' }}
        </template>
        <template v-else-if="column.key === 'actions'">
          <div class="table-actions">
            <a-button v-if="has('article:update')" size="small" @click="openEdit(record)">编辑</a-button>
            <a-button
              v-if="has('article:publish')"
              size="small"
              :type="record.is_published ? 'default' : 'primary'"
              :loading="updatingId === record.id"
              @click="updateState(record, { is_published: !record.is_published, published_at: record.published_at || null }, record.is_published ? '文章已取消发布' : '文章已发布')"
            >
              {{ record.is_published ? '取消发布' : '发布' }}
            </a-button>
            <a-button
              v-if="has('article:update')"
              size="small"
              :loading="updatingId === record.id"
              @click="updateState(record, { is_pinned: !record.is_pinned }, record.is_pinned ? '文章已取消置顶' : '文章已置顶')"
            >
              {{ record.is_pinned ? '取消置顶' : '置顶' }}
            </a-button>
            <a-popconfirm
              v-if="has('article:delete')"
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
      :title="editing ? '编辑文章' : '新建文章'"
      :confirm-loading="saving"
      width="820px"
      ok-text="保存"
      cancel-text="取消"
      @ok="submit"
    >
      <a-form layout="vertical">
        <a-form-item label="标题" required><a-input v-model:value="formState.title" :maxlength="255" /></a-form-item>
        <a-form-item label="摘要"><a-textarea v-model:value="formState.summary" :rows="2" :maxlength="500" /></a-form-item>
        <a-form-item label="正文" required>
          <a-textarea v-model:value="formState.content" :rows="12" placeholder="支持按原始文本换行显示" />
        </a-form-item>
        <a-form-item label="封面地址"><a-input v-model:value="formState.cover_url" placeholder="https://" /></a-form-item>
        <div class="form-grid">
          <a-form-item label="类型" required>
            <a-select v-model:value="formState.type" :options="[{ value: 'news', label: '资讯' }, { value: 'announcement', label: '公告' }, { value: 'event', label: '活动' }, { value: 'update', label: '更新' }]" />
          </a-form-item>
          <a-form-item label="发布时间"><a-input v-model:value="formState.published_at" type="datetime-local" :disabled="!has('article:publish')" /></a-form-item>
        </div>
        <div class="switch-row">
          <span><a-switch v-model:checked="formState.is_published" :disabled="!has('article:publish')" /> 发布</span>
          <span><a-switch v-model:checked="formState.is_pinned" /> 置顶</span>
        </div>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.table-toolbar { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; margin-bottom: 16px; }
.status-tags, .table-actions, .switch-row { display: flex; align-items: center; flex-wrap: wrap; gap: 6px; }
.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.switch-row { gap: 24px; }
.switch-row span { display: inline-flex; align-items: center; gap: 8px; }
@media (max-width: 639px) { .table-toolbar { align-items: stretch; flex-direction: column; } .form-grid { grid-template-columns: 1fr; gap: 0; } }
</style>
