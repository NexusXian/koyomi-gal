<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { createFeedbackService } from '~/services/feedback'
import type { FeedbackData } from '~/types/feedback'

useSeoMeta({ title: '反馈管理 - Koyomi' })

const feedbackService = createFeedbackService(useNuxtApp().$api)
const { has } = usePermissions()
const items = ref<FeedbackData[]>([])
const total = ref(0)
const page = ref(1)
const limit = 20
const loading = ref(false)
const updatingId = ref<number | null>(null)
const activeType = ref<'all' | 'feedback' | 'copyright'>('all')
const activeHandled = ref<'all' | 'pending' | 'handled'>('pending')

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = await feedbackService.listAdminFeedback({
      page: page.value,
      limit,
      type: activeType.value === 'all' ? undefined : activeType.value,
      handled: activeHandled.value === 'all' ? undefined : activeHandled.value === 'handled'
    })
    items.value = data.items ?? []
    total.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '反馈列表加载失败'))
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())
watch(page, () => void load())
watch([activeType, activeHandled], () => {
  if (page.value !== 1) page.value = 1
  else void load()
})

async function setHandled(feedback: FeedbackData, handled: boolean): Promise<void> {
  updatingId.value = feedback.id
  try {
    await feedbackService.handleFeedback(feedback.id, handled)
    message.success(handled ? '已标记为已处理' : '已标记为待处理')
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '状态更新失败'))
  } finally {
    updatingId.value = null
  }
}

function formatTime(value: string | null): string {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

const columns: TableColumnsType = [
  { title: '类型', dataIndex: 'type', width: 96 },
  { title: '内容', dataIndex: 'content', ellipsis: true },
  { title: '联系方式', dataIndex: 'contact', width: 160, ellipsis: true },
  { title: 'IP', dataIndex: 'ip', width: 130, ellipsis: true },
  { title: '提交时间', key: 'created', width: 165 },
  { title: '状态', key: 'status', width: 96 },
  { title: '操作', key: 'actions', width: 110 }
]
</script>

<template>
  <div>
    <div class="table-toolbar">
      <KunHeader
        name="反馈处理"
        description="查看用户提交的意见反馈与版权投诉，处理完成后标记状态。"
        scale="h3"
      />
      <div class="toolbar-filters">
        <a-select v-model:value="activeType" style="width: 120px">
          <a-select-option value="all">全部类型</a-select-option>
          <a-select-option value="feedback">意见反馈</a-select-option>
          <a-select-option value="copyright">版权投诉</a-select-option>
        </a-select>
        <a-select v-model:value="activeHandled" style="width: 120px">
          <a-select-option value="pending">待处理</a-select-option>
          <a-select-option value="handled">已处理</a-select-option>
          <a-select-option value="all">全部状态</a-select-option>
        </a-select>
      </div>
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
      :scroll="{ x: 960 }"
      @change="(pagination: { current?: number }) => { page = pagination.current ?? 1 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'type'">
          <a-tag :color="record.type === 'copyright' ? 'volcano' : 'blue'">
            {{ record.type === 'copyright' ? '版权投诉' : '意见反馈' }}
          </a-tag>
        </template>
        <template v-else-if="column.dataIndex === 'contact'">
          {{ record.contact || '-' }}
        </template>
        <template v-else-if="column.key === 'created'">
          {{ formatTime(record.created_at) }}
        </template>
        <template v-else-if="column.key === 'status'">
          <a-tag :color="record.handled_at ? 'success' : 'warning'">
            {{ record.handled_at ? '已处理' : '待处理' }}
          </a-tag>
        </template>
        <template v-else-if="column.key === 'actions'">
          <KunButton
            v-if="has('feedback:handle')"
            size="sm"
            :color="record.handled_at ? 'default' : 'primary'"
            variant="light"
            :loading="updatingId === record.id"
            @click="setHandled(record, !record.handled_at)"
          >
            {{ record.handled_at ? '标记待处理' : '标记已处理' }}
          </KunButton>
        </template>
      </template>
    </a-table>
  </div>
</template>

<style scoped>
.table-toolbar { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; margin-bottom: 16px; }
.toolbar-filters { display: flex; gap: 8px; }
@media (max-width: 639px) { .table-toolbar { align-items: stretch; flex-direction: column; } }
</style>
