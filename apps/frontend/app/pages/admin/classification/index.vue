<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  approveClassification,
  cancelClassification,
  getClassification,
  listClassifications,
  overrideClassification,
  rejectClassification,
  retryClassification,
  startClassification
} from '~/api/generated/galgame-classification/galgame-classification'
import type {
  DtoClassificationDetailData,
  DtoClassificationListItem,
  DtoOverrideClassificationRequest,
  ListClassificationsStatus
} from '~/api/generated/models'
import { formatDate } from '~/constants/domain'

useSeoMeta({ title: 'AI 分级队列 - Koyomi' })

const { has } = usePermissions()
const canRead = computed(() => has('galgame_classification:read'))
const canRun = computed(() => has('galgame_classification:run'))
const canApply = computed(() => has('galgame_classification:apply'))

const query = reactive({
  status: undefined as ListClassificationsStatus | undefined,
  keyword: '',
  page: 1
})
const limit = 20
const items = ref<DtoClassificationListItem[]>([])
const total = ref(0)
const loading = ref(false)
const actingId = ref<number | null>(null)

const detailOpen = ref(false)
const detailLoading = ref(false)
const detail = ref<DtoClassificationDetailData | null>(null)
const overrideOpen = ref(false)
const overrideValue = ref<'r18' | 'r17' | 'r15' | 'r12' | 'non_r18' | 'unknown'>('r18')
const overrideReason = ref('')
const overrideTargetId = ref<number | undefined>(undefined)

const STATUS_OPTIONS = [
  { value: 'queued', label: '排队中' },
  { value: 'processing', label: 'AI 判断中' },
  { value: 'pending_review', label: '待审核' },
  { value: 'approved', label: '已采用' },
  { value: 'rejected', label: '已拒绝' },
  { value: 'failed', label: '失败' },
  { value: 'cancelled', label: '已取消' }
]

const STATUS_LABELS: Record<string, string> = {
  queued: '排队中',
  processing: 'AI 判断中',
  pending_review: '待审核',
  approved: '已采用',
  rejected: '已拒绝',
  failed: '失败',
  cancelled: '已取消'
}

const STATUS_COLORS: Record<string, string> = {
  queued: 'processing',
  processing: 'blue',
  pending_review: 'gold',
  approved: 'success',
  rejected: 'error',
  failed: 'default',
  cancelled: 'default'
}

const RESULT_LABELS: Record<string, string> = {
  r18: 'R18',
  r17: '17+',
  r15: '15+',
  r12: '12+',
  non_r18: '非 R18',
  unknown: '未知'
}

const RESULT_COLORS: Record<string, string> = {
  r18: 'error',
  r17: 'orange',
  r15: 'gold',
  r12: 'blue',
  non_r18: 'success',
  unknown: 'warning'
}

const SOURCE_TYPE_LABELS: Record<string, string> = {
  official: '官方来源',
  steam: 'Steam',
  vndb: 'VNDB',
  bangumi: 'Bangumi',
  cero: 'CERO',
  esrb: 'ESRB',
  pegi: 'PEGI',
  wikipedia: 'Wikipedia',
  other: '其他'
}

function isActive(item: DtoClassificationListItem): boolean {
  return item.status === 'queued' || item.status === 'processing'
}

function verdictLabel(item: DtoClassificationListItem): string {
  if (isActive(item) || item.status === 'failed' || item.status === 'cancelled') {
    return STATUS_LABELS[item.status ?? ''] ?? item.status ?? ''
  }
  if (item.classification) {
    const confidence = item.confidence
      ? ` · ${Math.round(item.confidence * 100)}%`
      : ''
    return `${RESULT_LABELS[item.classification] ?? item.classification}${confidence}`
  }
  return '未判断'
}

function verdictColor(item: DtoClassificationListItem): string {
  if (isActive(item)) {
    return 'processing'
  }
  if (item.status === 'failed' || item.status === 'cancelled') {
    return 'default'
  }
  return RESULT_COLORS[item.classification ?? ''] ?? 'default'
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = unwrapApiData(
      await listClassifications({
        page: query.page,
        limit,
        status: query.status,
        keyword: query.keyword.trim() || undefined
      }),
      '查询失败'
    )
    items.value = data.items ?? []
    total.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '查询队列失败'))
  } finally {
    loading.value = false
  }
}

function search(): void {
  query.page = 1
  void load()
}

watch(
  () => query.keyword,
  () => {
    query.page = 1
  }
)

watch(
  () => [query.status, query.page],
  () => void load()
)

onMounted(() => void load())

// Poll while any row on the current page is still queued or processing.
let pollTimer: ReturnType<typeof setInterval> | null = null
watchEffect(() => {
  const running = items.value.some(isActive)
  if (running && !pollTimer) {
    pollTimer = setInterval(() => void load(), 5000)
  }
  if (!running && pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
})

onBeforeUnmount(() => {
  if (pollTimer) {
    clearInterval(pollTimer)
  }
})

async function cancel(item: DtoClassificationListItem): Promise<void> {
  if (!item.game_id || actingId.value) {
    return
  }
  actingId.value = item.game_id
  try {
    unwrapApiData(await cancelClassification(item.game_id), '取消失败')
    message.success(`「${item.game_title ?? item.game_id}」的 AI 判断已取消`)
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '取消失败'))
  } finally {
    actingId.value = null
  }
}

async function retry(item: DtoClassificationListItem): Promise<void> {
  if (!item.game_id || actingId.value) {
    return
  }
  actingId.value = item.game_id
  try {
    unwrapApiData(await retryClassification(item.game_id), '重试失败')
    message.success(`「${item.game_title ?? item.game_id}」已重新排队`)
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '重试失败'))
  } finally {
    actingId.value = null
  }
}

async function start(item: DtoClassificationListItem): Promise<void> {
  if (!item.game_id || actingId.value) {
    return
  }
  actingId.value = item.game_id
  try {
    unwrapApiData(await startClassification(item.game_id), '启动失败')
    message.success(`已启动「${item.game_title ?? item.game_id}」的 AI 判断`)
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '启动失败'))
  } finally {
    actingId.value = null
  }
}

async function openDetail(item: DtoClassificationListItem): Promise<void> {
  if (!item.game_id) {
    return
  }
  detail.value = null
  detailOpen.value = true
  detailLoading.value = true
  try {
    detail.value = unwrapApiData(
      await getClassification(item.game_id),
      '查询失败'
    )
  } catch (error) {
    message.error(getApiErrorMessage(error, '查询 AI 结果失败'))
    detailOpen.value = false
  } finally {
    detailLoading.value = false
  }
}

function detailGameId(): number | undefined {
  return detail.value?.game?.id
}

async function detailCancel(): Promise<void> {
  const id = detailGameId()
  if (!id || actingId.value) {
    return
  }
  actingId.value = id
  try {
    unwrapApiData(await cancelClassification(id), '取消失败')
    message.success('已取消该 AI 判断任务')
    detailOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '取消失败'))
  } finally {
    actingId.value = null
  }
}

async function detailRetry(): Promise<void> {
  const id = detailGameId()
  if (!id || actingId.value) {
    return
  }
  actingId.value = id
  try {
    unwrapApiData(await retryClassification(id), '重试失败')
    message.success('已重新排队 AI 判断')
    detailOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '重试失败'))
  } finally {
    actingId.value = null
  }
}

async function detailStart(): Promise<void> {
  const id = detailGameId()
  if (!id || actingId.value) {
    return
  }
  actingId.value = id
  try {
    unwrapApiData(await startClassification(id), '启动失败')
    message.success('已启动 AI 判断')
    detailOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '启动失败'))
  } finally {
    actingId.value = null
  }
}

async function detailApprove(): Promise<void> {
  const id = detailGameId()
  if (!id || actingId.value) {
    return
  }
  actingId.value = id
  try {
    unwrapApiData(await approveClassification(id), '操作失败')
    message.success('已采用 AI 结果并更新游戏年龄等级')
    detailOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '采用失败'))
  } finally {
    actingId.value = null
  }
}

async function detailReject(): Promise<void> {
  const id = detailGameId()
  if (!id || actingId.value) {
    return
  }
  actingId.value = id
  try {
    unwrapApiData(await rejectClassification(id), '操作失败')
    message.success('已拒绝该 AI 结果')
    detailOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '拒绝失败'))
  } finally {
    actingId.value = null
  }
}

function openOverride(): void {
  overrideValue.value = 'r18'
  overrideReason.value = ''
  overrideTargetId.value = detailGameId()
  overrideOpen.value = true
}

async function submitOverride(): Promise<void> {
  const targetId = overrideTargetId.value
  if (!targetId || actingId.value) {
    return
  }
  actingId.value = targetId
  try {
    unwrapApiData(
      await overrideClassification(targetId, {
        classification: overrideValue.value as DtoOverrideClassificationRequest['classification'],
        reason: overrideReason.value.trim() || undefined
      }),
      '操作失败'
    )
    message.success('已保存人工结论，等待审核采用')
    overrideOpen.value = false
    detailOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '覆盖失败'))
  } finally {
    actingId.value = null
  }
}

function rowActions(item: DtoClassificationListItem): { key: string; text: string }[] {
  const actions: { key: string; text: string }[] = []
  if (!canRun.value) {
    return actions
  }
  if (item.status === 'queued' || item.status === 'processing') {
    actions.push({ key: 'cancel', text: '取消' })
  } else if (item.status === 'failed') {
    actions.push({ key: 'retry', text: '重试' })
  }
  return actions
}

function detailStatus(): string | undefined {
  return detail.value?.classification?.status
}

const columns: TableColumnsType = [
  { title: 'ID', dataIndex: 'game_id', width: 80 },
  {
    title: '游戏',
    key: 'game',
    ellipsis: true
  },
  {
    title: '状态',
    key: 'status',
    width: 100
  },
  {
    title: 'AI 结论',
    key: 'verdict',
    width: 150
  },
  {
    title: '提交时间',
    key: 'created_at',
    width: 170
  },
  {
    title: '操作',
    key: 'actions',
    width: 140
  }
]
</script>

<template>
  <div class="classification-queue">
    <KunNull v-if="!canRead" text="没有 AI 分级队列权限" />
    <template v-else>
      <KunCard padding="md" class-name="admin-filter-card">
        <div class="admin-filter-row">
          <a-select
            v-model:value="query.status"
            class="filter-control"
            placeholder="全部状态"
            allow-clear
            :options="STATUS_OPTIONS"
          />
          <a-input-search
            v-model:value="query.keyword"
            class="filter-keyword"
            placeholder="搜索游戏标题 / 原名"
            allow-clear
            @search="search"
          />
          <a-button :loading="loading" @click="load">刷新</a-button>
        </div>
      </KunCard>

      <a-table
        :columns="columns"
        :data-source="items"
        :loading="loading"
        :pagination="{
          current: query.page,
          pageSize: limit,
          total,
          showSizeChanger: false,
          showTotal: (count: number) => `共 ${count} 条`
        }"
        row-key="id"
        size="middle"
        :scroll="{ x: 900 }"
        @change="
          (pagination: { current?: number }) => {
            query.page = pagination.current ?? 1
          }
        "
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'game'">
            <div class="game-cell">
              <span class="game-title">{{ record.game_title || '未命名游戏' }}</span>
              <span v-if="record.original_title" class="game-original">
                {{ record.original_title }}
              </span>
            </div>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="STATUS_COLORS[record.status ?? ''] ?? 'default'">
              {{ STATUS_LABELS[record.status ?? ''] ?? record.status }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'verdict'">
            <a-tooltip
              v-if="record.error_message"
              :title="`失败原因：${record.error_message}`"
            >
              <a-tag :color="verdictColor(record)">
                {{ verdictLabel(record) }}
              </a-tag>
            </a-tooltip>
            <template v-else>
              <a-tag :color="verdictColor(record)">
                {{ verdictLabel(record) }}
              </a-tag>
              <a-tag v-if="record.conflict" color="orange">证据冲突</a-tag>
            </template>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatDate(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-popconfirm
                v-if="rowActions(record).some((action) => action.key === 'cancel')"
                :title="`确认取消「${record.game_title ?? record.game_id}」的 AI 判断？`"
                ok-text="确认取消"
                ok-type="danger"
                cancel-text="再想想"
                @confirm="cancel(record)"
              >
                <a-button
                  size="small"
                  danger
                  :loading="actingId === record.game_id"
                >
                  取消
                </a-button>
              </a-popconfirm>
              <a-button
                v-else-if="rowActions(record).some((action) => action.key === 'retry')"
                size="small"
                type="primary"
                :loading="actingId === record.game_id"
                @click="retry(record)"
              >
                重试
              </a-button>
              <a-button size="small" @click="openDetail(record)">详情</a-button>
            </a-space>
          </template>
        </template>
        <template #emptyText>
          <KunNull text="队列是空的，先到 Galgame 审核页启动 AI 判断" />
        </template>
      </a-table>

      <a-modal
        v-model:open="detailOpen"
        title="AI 分级任务详情"
        width="640px"
        :footer="null"
      >
        <a-spin :spinning="detailLoading">
          <template v-if="detail && detail.classification">
            <div class="ai-detail-header">
              <div class="ai-detail-game">
                <span class="ai-detail-game-title">
                  {{ detail.game?.title || `游戏 #${detail.game?.id}` }}
                </span>
                <span v-if="detail.game?.original_title" class="ai-detail-game-original">
                  {{ detail.game?.original_title }}
                </span>
              </div>
              <div>
                <a-tag
                  :color="STATUS_COLORS[detail.classification.status ?? ''] ?? 'default'"
                >
                  {{ STATUS_LABELS[detail.classification.status ?? ''] ?? detail.classification.status }}
                </a-tag>
                <a-tag
                  :color="RESULT_COLORS[detail.classification.classification ?? ''] ?? 'default'"
                >
                  {{
                    detail.classification.classification
                      ? RESULT_LABELS[detail.classification.classification] ??
                        detail.classification.classification
                      : '未判断'
                  }}
                </a-tag>
                <a-tag v-if="detail.classification.conflict" color="orange">
                  证据冲突
                </a-tag>
              </div>
            </div>

            <div class="ai-detail-meta">
              <span v-if="detail.classification.confidence">
                置信度 {{ Math.round(detail.classification.confidence * 100) }}%
              </span>
              <span v-if="detail.classification.model">
                模型：{{ detail.classification.model }}
              </span>
              <span>提交：{{ formatDate(detail.classification.created_at) }}</span>
            </div>

            <p v-if="detail.classification.reason" class="ai-detail-reason">
              {{ detail.classification.reason }}
            </p>
            <div
              v-if="detail.classification.error_message"
              class="ai-detail-error"
            >
              {{ detail.classification.error_message }}
            </div>

            <div
              v-if="(detail.classification.evidences ?? []).length > 0"
              class="ai-detail-evidence"
            >
              <div
                v-for="evidence in detail.classification.evidences"
                :key="evidence.id"
                class="ai-detail-evidence-item"
              >
                <div class="ai-detail-evidence-head">
                  <a-tag color="geekblue">
                    {{
                      SOURCE_TYPE_LABELS[evidence.source_type ?? ''] ??
                      evidence.source_type
                    }}
                  </a-tag>
                  <a
                    v-if="evidence.url"
                    :href="evidence.url"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {{ evidence.title || evidence.url }}
                  </a>
                  <span v-else>{{ evidence.title }}</span>
                </div>
                <blockquote v-if="evidence.evidence" class="ai-detail-quote">
                  {{ evidence.evidence }}
                </blockquote>
              </div>
            </div>
            <p v-else class="ai-detail-empty">该结果没有引用任何网页证据。</p>

            <div class="ai-detail-actions">
              <a-popconfirm
                v-if="
                  (detailStatus() === 'queued' || detailStatus() === 'processing') &&
                  canRun
                "
                :title="`确认取消「${detail.game?.title ?? detail.game?.id}」的 AI 判断？`"
                ok-text="确认取消"
                ok-type="danger"
                cancel-text="再想想"
                @confirm="detailCancel"
              >
                <a-button danger :loading="actingId === detail.game?.id">取消任务</a-button>
              </a-popconfirm>
              <a-button
                v-if="detailStatus() === 'failed' && canRun"
                type="primary"
                :loading="actingId === detail.game?.id"
                @click="detailRetry"
              >
                重试
              </a-button>
              <template v-if="detailStatus() === 'pending_review' && canApply">
                <a-button
                  type="primary"
                  :loading="actingId === detail.game?.id"
                  @click="detailApprove"
                >
                  采用（写入正式年龄分级）
                </a-button>
                <a-popconfirm
                  title="确认拒绝该 AI 结果？不会改动游戏数据。"
                  ok-text="确认拒绝"
                  ok-type="danger"
                  cancel-text="再想想"
                  @confirm="detailReject"
                >
                  <a-button danger :loading="actingId === detail.game?.id">
                    拒绝
                  </a-button>
                </a-popconfirm>
              </template>
              <a-button
                v-if="
                  (detailStatus() === 'pending_review' ||
                    detailStatus() === 'failed' ||
                    detailStatus() === 'rejected') &&
                  canApply
                "
                :loading="actingId === detail.game?.id"
                @click="openOverride"
              >
                人工覆盖
              </a-button>
              <a-button
                v-if="
                  (detailStatus() === 'approved' ||
                    detailStatus() === 'rejected' ||
                    detailStatus() === 'cancelled') &&
                  canRun
                "
                :loading="actingId === detail.game?.id"
                @click="detailStart"
              >
                重新判断
              </a-button>
            </div>
          </template>
          <a-empty v-else description="该游戏还没有 AI 判断记录">
            <a-button
              v-if="detail?.game?.id && canRun"
              type="primary"
              @click="detailStart"
            >
              立即 AI 判断
            </a-button>
          </a-empty>
        </a-spin>
      </a-modal>

      <a-modal
        v-model:open="overrideOpen"
        title="人工覆盖 AI 分级"
        :confirm-loading="actingId === overrideTargetId"
        ok-text="保存"
        cancel-text="取消"
        @ok="submitOverride"
      >
        <div class="override-form">
          <a-select
            v-model:value="overrideValue"
            class="override-select"
            :options="Object.entries(RESULT_LABELS).map(([value, label]) => ({ value, label }))"
          />
          <a-textarea
            v-model:value="overrideReason"
            :rows="3"
            :maxlength="1000"
            placeholder="选填：人工结论的依据"
          />
        </div>
      </a-modal>
    </template>
  </div>
</template>

<style scoped>
.admin-filter-card {
  margin-bottom: 16px;
}

.admin-filter-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.filter-control {
  width: 150px;
}

.filter-keyword {
  width: 280px;
}

.game-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  line-height: 1.4;
}

.game-title {
  font-weight: 600;
}

.game-original {
  color: var(--color-muted-foreground);
  font-size: 12px;
}

.ai-detail-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.ai-detail-game {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.ai-detail-game-title {
  font-size: 15px;
  font-weight: 600;
}

.ai-detail-game-original {
  color: var(--color-muted-foreground);
  font-size: 12px;
}

.ai-detail-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  margin-bottom: 10px;
  color: var(--color-muted-foreground);
  font-size: 13px;
}

.ai-detail-reason {
  margin: 0 0 10px;
  color: var(--color-foreground);
  white-space: pre-wrap;
}

.ai-detail-error {
  margin-bottom: 10px;
  padding: 8px 10px;
  border-radius: var(--radius-kun-sm);
  background: color-mix(in srgb, var(--color-danger) 10%, transparent);
  color: var(--color-danger);
  font-size: 13px;
  white-space: pre-wrap;
}

.ai-detail-evidence {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.ai-detail-evidence-item {
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--color-border) 70%, transparent);
  border-radius: var(--radius-kun-sm);
}

.ai-detail-evidence-head {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.ai-detail-quote {
  margin: 8px 0 0;
  padding-left: 12px;
  border-left: 3px solid var(--color-primary);
  color: var(--color-muted-foreground);
  font-size: 13px;
  white-space: pre-wrap;
}

.ai-detail-empty {
  margin: 0;
  color: var(--color-muted-foreground);
  font-size: 13px;
}

.ai-detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid color-mix(in srgb, var(--color-border) 70%, transparent);
}

.override-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.override-select {
  width: 100%;
}
</style>
