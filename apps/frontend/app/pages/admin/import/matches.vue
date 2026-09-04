<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  approveMatchCandidate,
  batchApproveMatchCandidates,
  batchRejectMatchCandidates,
  listMatchCandidates,
  rejectMatchCandidate
} from '~/api/generated/admin-import/admin-import'
import type { DtoMatchCandidateItem } from '~/api/generated/models'

useSeoMeta({ title: '匹配审核 - Koyomi' })

const { has } = usePermissions()
const canImport = computed(() => has('galgame:import'))

const REASON_LABELS: Record<string, string> = {
  original_title_match: '原始标题一致',
  alias_match: '别名一致',
  normalized_title_match: '标题归一化一致',
  release_year_match: '发行年份一致',
  developer_match: '开发商一致'
}

const STATUS_OPTIONS = [
  { value: 0, label: '待审核' },
  { value: 1, label: '已通过' },
  { value: 2, label: '已拒绝' }
]

const status = ref<number>(0)
const page = ref(1)
const limit = 20
const items = ref<DtoMatchCandidateItem[]>([])
const total = ref(0)
const loading = ref(false)
const actingId = ref<number | null>(null)
const selectedIds = ref<number[]>([])
const batchLoading = ref<'approve' | 'reject' | null>(null)

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = unwrapApiData(
      await listMatchCandidates({
        status: status.value,
        page: page.value,
        limit
      })
    )
    items.value = data.items ?? []
    total.value = data.total ?? 0
    selectedIds.value = []
  } catch (error) {
    message.error(getApiErrorMessage(error, '查询匹配候选失败'))
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())
watch(page, () => void load())
watch(status, () => {
  if (page.value !== 1) {
    page.value = 1
  } else {
    void load()
  }
})

async function approve(item: DtoMatchCandidateItem): Promise<void> {
  if (!item.id) {
    return
  }
  actingId.value = item.id
  try {
    const result = unwrapApiData(await approveMatchCandidate(item.id))
    const updated = (result.updated_fields ?? []).length
    message.success(
      updated > 0
        ? `已关联并补全 ${updated} 个字段`
        : '已关联外部条目'
    )
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '通过候选失败'))
  } finally {
    actingId.value = null
  }
}

async function reject(item: DtoMatchCandidateItem): Promise<void> {
  if (!item.id) {
    return
  }
  actingId.value = item.id
  try {
    await rejectMatchCandidate(item.id)
    message.success('已拒绝该候选')
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '拒绝候选失败'))
  } finally {
    actingId.value = null
  }
}

const rowSelection = computed(() =>
  status.value === 0
    ? {
        selectedRowKeys: selectedIds.value,
        onChange: (keys: (number | string)[]) => {
          selectedIds.value = keys.map(Number)
        }
      }
    : undefined
)

async function runBatch(action: 'approve' | 'reject'): Promise<void> {
  if (selectedIds.value.length === 0) {
    return
  }
  batchLoading.value = action
  try {
    const request = { ids: selectedIds.value }
    const result = unwrapApiData(
      action === 'approve'
        ? await batchApproveMatchCandidates(request)
        : await batchRejectMatchCandidates(request)
    )
    const failed = (result.items ?? []).filter((item) => item.status === 'failed')
    if (failed.length > 0) {
      const detail = failed
        .slice(0, 3)
        .map((item) => `#${item.id} ${item.message ?? '处理失败'}`)
        .join('；')
      message.warning(
        `成功 ${result.succeeded_count ?? 0} 条，失败 ${result.failed_count ?? failed.length} 条：${detail}${failed.length > 3 ? ' 等' : ''}`
      )
    } else {
      message.success(`已批量${action === 'approve' ? '通过' : '拒绝'} ${result.succeeded_count ?? 0} 条`)
    }
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '批量审核失败'))
  } finally {
    batchLoading.value = null
  }
}

function formatConfidence(value?: number): string {
  return value === null || value === undefined
    ? '-'
    : `${Math.round(value * 100)}%`
}

function reasonText(reasons?: string[]): string {
  return (reasons ?? [])
    .map((reason) => REASON_LABELS[reason] ?? reason)
    .join('、')
}

function formatDateTime(value?: string): string {
  return value ? value.replace('T', ' ').slice(0, 19) : '-'
}

const columns: TableColumnsType = [
  { title: '封面', key: 'cover', width: 64 },
  { title: '站内作品', key: 'galgame', ellipsis: true },
  { title: '外部候选', key: 'external', ellipsis: true },
  { title: '匹配度', key: 'confidence', width: 90 },
  { title: '匹配依据', key: 'reasons', ellipsis: true },
  { title: '状态', key: 'status', width: 90 },
  { title: '提交时间', key: 'created', width: 165 },
  { title: '操作', key: 'actions', width: 140 }
]
</script>

<template>
  <div class="match-review">
    <KunNull v-if="!canImport" text="没有外部数据导入权限" />

    <template v-else>
      <div class="match-review-toolbar">
        <KunHeader
          name="匹配审核"
          description="自动匹配产生的中等置信度候选，确认后才会关联外部条目并补全元数据。"
          scale="h3"
        />
        <a-space>
          <template v-if="status === 0">
            <span
              v-if="selectedIds.length > 0"
              class="match-review-selected"
            >
              已选 {{ selectedIds.length }} 条
            </span>
            <a-popconfirm
              :title="`确认批量关联并补全所选 ${selectedIds.length} 条候选？`"
              ok-text="确认"
              cancel-text="取消"
              :disabled="selectedIds.length === 0"
              @confirm="runBatch('approve')"
            >
              <a-button
                size="small"
                type="primary"
                :disabled="selectedIds.length === 0"
                :loading="batchLoading === 'approve'"
              >
                批量通过
              </a-button>
            </a-popconfirm>
            <a-popconfirm
              :title="`确认批量拒绝所选 ${selectedIds.length} 条候选？`"
              ok-text="确认"
              cancel-text="取消"
              :disabled="selectedIds.length === 0"
              @confirm="runBatch('reject')"
            >
              <a-button
                size="small"
                danger
                :disabled="selectedIds.length === 0"
                :loading="batchLoading === 'reject'"
              >
                批量拒绝
              </a-button>
            </a-popconfirm>
          </template>
          <a-select
            v-model:value="status"
            class="match-review-status"
            :options="STATUS_OPTIONS"
          />
        </a-space>
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
        :row-selection="rowSelection"
        row-key="id"
        size="middle"
        :scroll="{ x: 1080 }"
        @change="(pagination: { current?: number }) => { page = pagination.current ?? 1 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'cover'">
            <img
              v-if="record.preview?.cover_url"
              :src="record.preview.cover_url"
              class="match-review-cover"
              loading="lazy"
              referrerpolicy="no-referrer"
              alt=""
            />
            <span v-else>-</span>
          </template>
          <template v-else-if="column.key === 'galgame'">
            <div class="match-review-title-cell">
              <span class="match-review-title-main">
                {{ record.galgame_title || `#${record.galgame_id}` }}
              </span>
              <a-button
                type="link"
                size="small"
                class="match-review-link"
                @click="navigateTo(`/galgames/${record.galgame_id}`)"
              >
                站内 #{{ record.galgame_id }}
              </a-button>
            </div>
          </template>
          <template v-else-if="column.key === 'external'">
            <div class="match-review-title-cell">
              <span class="match-review-title-main">
                {{ record.preview?.title || `#${record.external_id}` }}
              </span>
              <span class="match-review-title-sub">
                {{ record.provider?.toUpperCase() }} {{ record.external_id }}
                {{ record.preview?.original_title ? ` · ${record.preview.original_title}` : '' }}
                <template v-if="record.preview?.rating != null">
                  · {{ record.preview.rating.toFixed(1) }}（{{ record.preview.rating_count ?? 0 }} 票）
                </template>
              </span>
            </div>
          </template>
          <template v-else-if="column.key === 'confidence'">
            <a-tag
              :color="(record.confidence ?? 0) >= 0.85 ? 'green' : 'orange'"
            >
              {{ formatConfidence(record.confidence) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'reasons'">
            {{ reasonText(record.reasons) || '-' }}
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag
              :color="
                record.status === 1 ? 'success' : record.status === 2 ? 'default' : 'warning'
              "
            >
              {{ record.status_label }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'created'">
            {{ formatDateTime(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <template v-if="record.status === 0">
              <a-popconfirm
                title="确认关联该外部条目并执行补全？"
                ok-text="确认"
                cancel-text="取消"
                @confirm="approve(record)"
              >
                <a-button
                  size="small"
                  type="primary"
                  :loading="actingId === record.id"
                >
                  通过
                </a-button>
              </a-popconfirm>
              <a-popconfirm
                title="确认拒绝该候选？"
                ok-text="确认"
                cancel-text="取消"
                @confirm="reject(record)"
              >
                <a-button
                  size="small"
                  class="match-review-action"
                  :loading="actingId === record.id"
                >
                  拒绝
                </a-button>
              </a-popconfirm>
            </template>
            <span v-else>-</span>
          </template>
        </template>
        <template #emptyText>
          <KunNull text="没有待审核的匹配" />
        </template>
      </a-table>
    </template>
  </div>
</template>

<style scoped>
.match-review-toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.match-review-status {
  width: 120px;
  flex: none;
}

.match-review-selected {
  font-size: 12px;
  opacity: 0.75;
}

.match-review-cover {
  width: 40px;
  height: 56px;
  object-fit: cover;
  border-radius: 4px;
  display: block;
}

.match-review-title-cell {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.match-review-title-main {
  font-weight: 600;
}

.match-review-title-sub {
  font-size: 12px;
  opacity: 0.65;
}

.match-review-link {
  padding: 0;
  height: auto;
  font-size: 12px;
}

.match-review-action {
  margin-left: 8px;
}
</style>
