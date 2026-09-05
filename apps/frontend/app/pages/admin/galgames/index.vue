<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  approveClassification,
  batchApproveClassification,
  batchClassification,
  getClassification,
  overrideClassification,
  rejectClassification,
  retryClassification,
  startClassification
} from '~/api/generated/galgame-classification/galgame-classification'
import {
  batchDeleteGalgames,
  batchUpdateGalgames,
  getAdminGalgame,
  listAdminGalgames,
  reviewGalgame
} from '~/api/generated/admin/admin'
import { updateGalgame } from '~/api/generated/galgames/galgames'
import type {
  DtoBatchData,
  DtoClassificationDetailData,
  DtoGalgameListItem
} from '~/api/generated/models'
import { AGE_RATINGS, GALGAME_SORTS, domainLabel, domainSortSlug } from '~/constants/domain'

useSeoMeta({ title: 'Galgame 审核 - Koyomi' })

const { has } = usePermissions()
const query = reactive({
  status: undefined as number | undefined,
  ageRating: undefined as number | undefined,
  coverSensitive: undefined as boolean | undefined,
  keyword: '',
  sort: 0,
  page: 1,
  aiClassification: undefined as string | undefined,
  aiStatus: undefined as string | undefined,
  aiConflict: undefined as boolean | undefined,
  aiConfidence: undefined as 'high95' | 'low70' | undefined
})

const limit = 20
const items = ref<DtoGalgameListItem[]>([])
const total = ref(0)
const loading = ref(false)
const statusUpdating = ref<number | null>(null)
const quickUpdatingId = ref<number | null>(null)
const aiActingId = ref<number | null>(null)

const selectedKeys = ref<number[]>([])
const batchAgeOpen = ref(false)
const batchAgeValue = ref<number>(0)
const batchCoverOpen = ref(false)
const batchCoverMark = ref<'mark' | 'unmark'>('mark')
const batchDeleteOpen = ref(false)
const batchSubmitting = ref(false)
const rowDeletingId = ref<number | null>(null)
const batchAiRunning = ref(false)

const aiDetailOpen = ref(false)
const aiDetailLoading = ref(false)
const aiDetail = ref<DtoClassificationDetailData | null>(null)
const aiOverrideOpen = ref(false)
const aiOverrideValue = ref<'r18' | 'non_r18' | 'unknown'>('r18')
const aiOverrideReason = ref('')

const fallbackCover = `data:image/svg+xml;utf8,${encodeURIComponent(
  '<svg xmlns="http://www.w3.org/2000/svg" width="300" height="400"><rect width="100%" height="100%" fill="#e9e5f5"/><text x="50%" y="50%" text-anchor="middle" font-size="64" fill="#9b8ec4">G</text></svg>'
)}`

const GALGAME_STATUS_OPTIONS = [
  { value: 0, label: '待审核' },
  { value: 1, label: '已发布' },
  { value: 2, label: '已拒绝' },
  { value: 3, label: '已隐藏' }
]

const COVER_FILTER_OPTIONS = [
  { value: 1, label: '仅敏感封面' },
  { value: 0, label: '仅普通封面' }
]

const AI_CLASSIFICATION_OPTIONS = [
  { value: 'r18', label: 'AI = R18' },
  { value: 'non_r18', label: 'AI = 非 R18' },
  { value: 'unknown', label: 'AI = 未知' }
]

const AI_STATUS_OPTIONS = [
  { value: 'pending_review', label: '仅待审核' },
  { value: 'failed', label: '仅失败' },
  { value: 'approved', label: '仅已采用' },
  { value: 'rejected', label: '仅已拒绝' }
]

const AI_CONFLICT_OPTIONS = [
  { value: true, label: '仅证据冲突' },
  { value: false, label: '仅无冲突' }
]

const AI_CONFIDENCE_OPTIONS = [
  { value: 'high95', label: '置信度 ≥ 95%' },
  { value: 'low70', label: '置信度 < 70%' }
]

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

const AI_STATUS_LABELS: Record<string, string> = {
  queued: '排队中',
  processing: 'AI 判断中',
  pending_review: '待审核',
  approved: '已采用',
  rejected: '已拒绝',
  failed: '失败'
}

const AI_RESULT_LABELS: Record<string, string> = {
  r18: 'R18',
  non_r18: '非 R18',
  unknown: '未知'
}

const AI_RESULT_COLORS: Record<string, string> = {
  r18: 'error',
  non_r18: 'success',
  unknown: 'warning',
  queued: 'processing',
  processing: 'processing',
  failed: 'default'
}

function isAiRunning(item: DtoGalgameListItem): boolean {
  return item.ai_status === 'queued' || item.ai_status === 'processing'
}

function aiCellLabel(item: DtoGalgameListItem): string {
  if (isAiRunning(item)) {
    return AI_STATUS_LABELS[item.ai_status ?? ''] ?? 'AI 判断中'
  }
  if (item.ai_status === 'failed') {
    return '失败'
  }
  if (item.ai_classification) {
    const label = AI_RESULT_LABELS[item.ai_classification] ?? item.ai_classification
    const confidence = item.ai_confidence
      ? ` ${Math.round(item.ai_confidence * 100)}%`
      : ''
    return `${label}${confidence}`
  }
  return '未判断'
}

function aiCellColor(item: DtoGalgameListItem): string {
  if (isAiRunning(item) || item.ai_status === 'failed') {
    return AI_RESULT_COLORS[item.ai_status ?? ''] ?? 'default'
  }
  return AI_RESULT_COLORS[item.ai_classification ?? ''] ?? 'default'
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = unwrapApiData(
      await listAdminGalgames({
        status: query.status,
        age_rating: query.ageRating,
        cover_sensitive: query.coverSensitive,
        keyword: query.keyword.trim() || undefined,
        sort: domainSortSlug(GALGAME_SORTS, query.sort),
        page: query.page,
        limit,
        ai_classification: query.aiClassification,
        ai_status: query.aiStatus,
        ai_conflict: query.aiConflict,
        ai_min_confidence: query.aiConfidence === 'high95' ? 0.95 : undefined,
        ai_max_confidence: query.aiConfidence === 'low70' ? 0.7 : undefined
      })
    )
    items.value = data.items ?? []
    total.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '查询失败'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void load()
})

watch(
  () => [
    query.status,
    query.ageRating,
    query.coverSensitive,
    query.sort,
    query.page,
    query.aiClassification,
    query.aiStatus,
    query.aiConflict,
    query.aiConfidence
  ],
  () => {
    void load()
  }
)

// Poll while any row on the current page is still queued or processing.
let aiPollTimer: ReturnType<typeof setInterval> | null = null
watchEffect(() => {
  const running = items.value.some(isAiRunning)
  if (running && !aiPollTimer) {
    aiPollTimer = setInterval(() => void load(), 5000)
  }
  if (!running && aiPollTimer) {
    clearInterval(aiPollTimer)
    aiPollTimer = null
  }
})

onBeforeUnmount(() => {
  if (aiPollTimer) {
    clearInterval(aiPollTimer)
  }
})

watch(
  () => query.keyword,
  () => {
    query.page = 1
  }
)

function search(): void {
  query.page = 1
  void load()
}

function openEdit(item: DtoGalgameListItem): void {
  if (item.id) {
    void navigateTo(`/galgames/${item.id}/edit`)
  }
}

const QUICK_ACTIONS: Record<number, { status: number; label: string }[]> = {
  0: [
    { status: 1, label: '通过发布' },
    { status: 2, label: '拒绝' }
  ],
  1: [
    { status: 3, label: '隐藏' },
    { status: 2, label: '拒绝' }
  ],
  2: [
    { status: 1, label: '发布' },
    { status: 0, label: '重新待审' }
  ],
  3: [
    { status: 1, label: '恢复发布' },
    { status: 0, label: '重新待审' }
  ]
}

async function changeStatus(
  item: DtoGalgameListItem,
  status: number
): Promise<void> {
  const isReview = item.status === 0 && (status === 1 || status === 2)
  if (
    !item.id ||
    (isReview ? !has('galgame:review') : !has('galgame:update'))
  ) {
    return
  }

  statusUpdating.value = item.id
  try {
    if (isReview) {
      await reviewGalgame(item.id, { status: status as 1 | 2 })
      message.success(status === 1 ? 'Galgame 已发布' : 'Galgame 已拒绝')
      await load()
      return
    }
    const detail = unwrapApiData(await getAdminGalgame(item.id))
    await updateGalgame(item.id, {
      title: detail.title ?? '',
      slug: detail.slug ?? '',
      romaji_title: detail.romaji_title || undefined,
      original_title: detail.original_title || undefined,
      developer_id: detail.developer?.id,
      release_date: detail.release_date?.slice(0, 10),
      age_rating: (detail.age_rating ?? 0) as 0 | 1 | 2 | 3 | 4 | 5,
      cover_sensitive: detail.cover_sensitive ?? false,
      status: status as 0 | 1 | 2 | 3,
      tag_ids: (detail.tags ?? [])
        .map((tag) => tag.id)
        .filter((id): id is number => Boolean(id)),
      aliases: detail.aliases ?? [],
      cover_url: detail.cover_url || undefined,
      banner_url: detail.banner_url || undefined,
      description: detail.description || undefined
    })
    message.success(
      `「${item.title ?? item.id}」状态已更新为「${GALGAME_STATUS_OPTIONS.find((option) => option.value === status)?.label}」`
    )
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '更新状态失败'))
  } finally {
    statusUpdating.value = null
  }
}

async function quickUpdate(
  item: DtoGalgameListItem,
  updates: { age_rating?: 0 | 1 | 2 | 3 | 4 | 5; cover_sensitive?: boolean },
  successText: string
): Promise<void> {
  if (!item.id || quickUpdatingId.value) {
    return
  }
  quickUpdatingId.value = item.id
  try {
    unwrapApiData(
      await batchUpdateGalgames({ ids: [item.id], ...updates }),
      '更新失败'
    )
    if (updates.age_rating !== undefined) {
      item.age_rating = updates.age_rating
    }
    if (updates.cover_sensitive !== undefined) {
      item.cover_sensitive = updates.cover_sensitive
    }
    message.success(`「${item.title ?? item.id}」${successText}`)
  } catch (error) {
    message.error(getApiErrorMessage(error, '更新失败'))
  } finally {
    quickUpdatingId.value = null
  }
}

function onSelectionChange(keys: (number | string)[]): void {
  selectedKeys.value = keys.filter(
    (key): key is number => typeof key === 'number'
  )
}

function clearSelection(): void {
  selectedKeys.value = []
}

async function runAi(item: DtoGalgameListItem, retry = false): Promise<void> {
  if (!item.id || aiActingId.value) {
    return
  }
  aiActingId.value = item.id
  try {
    unwrapApiData(
      retry
        ? await retryClassification(item.id)
        : await startClassification(item.id),
      '启动失败'
    )
    message.success(retry ? '已重新排队 AI 判断' : '已启动 AI 判断')
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, retry ? '重试失败' : '启动失败'))
  } finally {
    aiActingId.value = null
  }
}

async function approveAi(item: DtoGalgameListItem): Promise<void> {
  if (!item.id || aiActingId.value) {
    return
  }
  aiActingId.value = item.id
  try {
    unwrapApiData(await approveClassification(item.id), '操作失败')
    message.success('已采用 AI 结果并更新游戏年龄等级')
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '采用失败'))
  } finally {
    aiActingId.value = null
  }
}

async function rejectAi(item: DtoGalgameListItem): Promise<void> {
  if (!item.id || aiActingId.value) {
    return
  }
  aiActingId.value = item.id
  try {
    unwrapApiData(await rejectClassification(item.id), '操作失败')
    message.success('已拒绝该 AI 结果')
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '拒绝失败'))
  } finally {
    aiActingId.value = null
  }
}

async function openAiDetail(item: DtoGalgameListItem): Promise<void> {
  if (!item.id) {
    return
  }
  aiDetail.value = null
  aiDetailOpen.value = true
  aiDetailLoading.value = true
  try {
    aiDetail.value = unwrapApiData(
      await getClassification(item.id),
      '查询失败'
    )
  } catch (error) {
    message.error(getApiErrorMessage(error, '查询 AI 结果失败'))
    aiDetailOpen.value = false
  } finally {
    aiDetailLoading.value = false
  }
}

function openOverride(item: DtoGalgameListItem): void {
  aiOverrideValue.value = 'r18'
  aiOverrideReason.value = ''
  aiOverrideTargetId.value = item.id
  aiOverrideOpen.value = true
}

async function runAiFromDetail(): Promise<void> {
  const id = aiDetail.value?.game?.id
  if (!id) {
    return
  }
  aiDetailOpen.value = false
  await runAi({ id } as DtoGalgameListItem)
}

const aiOverrideTargetId = ref<number | undefined>(undefined)

async function submitOverride(): Promise<void> {
  const targetId = aiOverrideTargetId.value
  if (!targetId || aiActingId.value) {
    return
  }
  aiActingId.value = targetId
  try {
    unwrapApiData(
      await overrideClassification(targetId, {
        classification: aiOverrideValue.value,
        reason: aiOverrideReason.value.trim() || undefined
      }),
      '操作失败'
    )
    message.success('已保存人工结论，等待审核采用')
    aiOverrideOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '覆盖失败'))
  } finally {
    aiActingId.value = null
  }
}

const selectedItems = computed<DtoGalgameListItem[]>(() =>
  items.value.filter((item) => item.id && selectedKeys.value.includes(item.id))
)

const selectedAiActionable = computed(() => selectedItems.value.length > 0)

const selectedPendingReview = computed(() =>
  selectedItems.value.filter((item) => item.ai_status === 'pending_review')
)

async function submitBatchAi(): Promise<void> {
  if (batchSubmitting.value || selectedKeys.value.length === 0) {
    return
  }
  batchSubmitting.value = true
  try {
    const data = unwrapApiData(
      await batchClassification({ game_ids: selectedKeys.value }),
      '批量启动失败'
    )
    const parts = [`已入队 ${data.enqueued ?? 0} 个游戏`]
    if ((data.already_running ?? []).length > 0) {
      parts.push(`${data.already_running?.length} 个游戏已在判断中`)
    }
    message.success(parts.join('，'))
    clearSelection()
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '批量启动失败'))
  } finally {
    batchSubmitting.value = false
  }
}

async function submitBatchAiApprove(): Promise<void> {
  if (batchSubmitting.value || selectedPendingReview.value.length === 0) {
    return
  }
  batchSubmitting.value = true
  try {
    const data = unwrapApiData(
      await batchApproveClassification({
        game_ids: selectedPendingReview.value
          .map((item) => item.id)
          .filter((id): id is number => Boolean(id))
      }),
      '批量采用失败'
    )
    reportBatchResult(data, '批量采用完成')
    clearSelection()
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '批量采用失败'))
  } finally {
    batchSubmitting.value = false
  }
}

function reportBatchResult(data: DtoBatchData, fallback: string): void {
  const approved = (data.approved ?? []).length
  const skipped = (data.skipped ?? []).length
  if (approved === 0 && skipped === 0) {
    message.warning('没有满足条件的 AI 结果（需待审核、置信度 ≥ 95% 且无冲突）')
    return
  }
  const parts = [`已采用 ${approved} 个`]
  if (skipped > 0) {
    parts.push(`跳过 ${skipped} 个（置信度不足 / 冲突 / unknown）`)
  }
  message.success(`${fallback}：${parts.join('，')}`)
}

function openBatchAge(): void {
  batchAgeValue.value = 0
  batchAgeOpen.value = true
}

function openBatchCover(): void {
  batchCoverMark.value = 'mark'
  batchCoverOpen.value = true
}

async function submitBatchAge(): Promise<void> {
  if (batchSubmitting.value || selectedKeys.value.length === 0) {
    return
  }
  batchSubmitting.value = true
  try {
    const data = unwrapApiData(
      await batchUpdateGalgames({
        ids: selectedKeys.value,
        age_rating: batchAgeValue.value as 0 | 1 | 2 | 3 | 4 | 5
      }),
      '批量设置失败'
    )
    message.success(`已批量设置 ${data.updated ?? 0} 个游戏的年龄等级`)
    batchAgeOpen.value = false
    clearSelection()
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '批量设置失败'))
  } finally {
    batchSubmitting.value = false
  }
}

async function submitBatchCover(): Promise<void> {
  if (batchSubmitting.value || selectedKeys.value.length === 0) {
    return
  }
  batchSubmitting.value = true
  try {
    const data = unwrapApiData(
      await batchUpdateGalgames({
        ids: selectedKeys.value,
        cover_sensitive: batchCoverMark.value === 'mark'
      }),
      '批量设置失败'
    )
    message.success(
      batchCoverMark.value === 'mark'
        ? `已将 ${data.updated ?? 0} 个游戏标记为敏感封面`
        : `已取消 ${data.updated ?? 0} 个游戏的敏感封面标记`
    )
    batchCoverOpen.value = false
    clearSelection()
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '批量设置失败'))
  } finally {
    batchSubmitting.value = false
  }
}

async function removeGalgame(item: DtoGalgameListItem): Promise<void> {
  if (!item.id || rowDeletingId.value) {
    return
  }
  rowDeletingId.value = item.id
  try {
    unwrapApiData(await batchDeleteGalgames({ ids: [item.id] }), '删除失败')
    message.success(`「${item.title ?? item.id}」已删除`)
    if (selectedKeys.value.length > 0) {
      selectedKeys.value = selectedKeys.value.filter((key) => key !== item.id)
    }
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除失败'))
  } finally {
    rowDeletingId.value = null
  }
}

async function submitBatchDelete(): Promise<void> {
  if (batchSubmitting.value || selectedKeys.value.length === 0) {
    return
  }
  batchSubmitting.value = true
  try {
    const data = unwrapApiData(
      await batchDeleteGalgames({ ids: selectedKeys.value }),
      '批量删除失败'
    )
    message.success(`已删除 ${data.deleted ?? 0} 个 Galgame`)
    batchDeleteOpen.value = false
    clearSelection()
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '批量删除失败'))
  } finally {
    batchSubmitting.value = false
  }
}

const canUpdate = computed(() => has('galgame:update'))
const canDelete = computed(() => has('galgame:delete'))
const canSelect = computed(() => canUpdate.value || canDelete.value)
const canRunAi = computed(() => has('galgame_classification:run'))
const canReadAi = computed(() => has('galgame_classification:read'))
const canApplyAi = computed(() => has('galgame_classification:apply'))

const columns = computed<TableColumnsType>(() => [
  { title: 'ID', dataIndex: 'id', width: 70 },
  {
    title: '封面',
    key: 'cover',
    width: 76
  },
  {
    title: '标题',
    dataIndex: 'title',
    ellipsis: true
  },
  {
    title: '开发商',
    dataIndex: ['developer', 'name'],
    width: 130,
    ellipsis: true
  },
  {
    title: '年龄等级',
    key: 'age_rating',
    width: 110
  },
  {
    title: '敏感封面',
    key: 'cover_sensitive',
    width: 100
  },
  {
    title: '状态',
    dataIndex: 'status',
    width: 90
  },
  {
    title: 'AI 分级',
    key: 'ai_classification',
    width: 200
  },
  {
    title: '评分',
    key: 'rating',
    width: 90
  },
  {
    title: '收藏',
    key: 'favorite',
    width: 70
  },
  ...(
    canUpdate.value || canDelete.value
      ? [{ title: '操作', key: 'actions', width: 260, fixed: 'right' as const }]
      : []
  )
])

const rowSelection = computed(() =>
  canSelect.value
    ? {
        selectedRowKeys: selectedKeys.value,
        onChange: onSelectionChange
      }
    : undefined
)
</script>

<template>
  <div>
    <KunCard padding="md" class-name="admin-filter-card">
      <div class="admin-filter-row">
        <a-select
          v-model:value="query.status"
          class="filter-control"
          :options="GALGAME_STATUS_OPTIONS"
          placeholder="全部状态"
          allow-clear
        />
        <a-select
          v-model:value="query.ageRating"
          class="filter-control"
          :options="
            AGE_RATINGS.map((item) => ({
              value: item.value,
              label: item.label
            }))
          "
          placeholder="全部年龄等级"
          allow-clear
        />
        <a-select
          v-model:value="query.coverSensitive"
          class="filter-control"
          :options="COVER_FILTER_OPTIONS"
          placeholder="全部封面"
          allow-clear
        />
        <template v-if="canReadAi">
          <a-select
            v-model:value="query.aiClassification"
            class="filter-control"
            :options="AI_CLASSIFICATION_OPTIONS"
            placeholder="全部 AI 结果"
            allow-clear
          />
          <a-select
            v-model:value="query.aiStatus"
            class="filter-control"
            :options="AI_STATUS_OPTIONS"
            placeholder="全部 AI 状态"
            allow-clear
          />
          <a-select
            v-model:value="query.aiConflict"
            class="filter-control"
            :options="AI_CONFLICT_OPTIONS"
            placeholder="全部冲突状态"
            allow-clear
          />
          <a-select
            v-model:value="query.aiConfidence"
            class="filter-control"
            :options="AI_CONFIDENCE_OPTIONS"
            placeholder="全部置信度"
            allow-clear
          />
        </template>
        <a-select
          v-model:value="query.sort"
          class="filter-control"
          :options="
            GALGAME_SORTS.map((item, index) => ({
              value: index,
              label: item.label
            }))
          "
        />
        <a-input-search
          v-model:value="query.keyword"
          class="filter-keyword"
          placeholder="搜索标题或别名"
          allow-clear
          @search="search"
        />
      </div>
    </KunCard>

    <div v-if="canSelect && selectedKeys.length > 0" class="batch-bar">
      <span class="batch-count">已选择 {{ selectedKeys.length }} 项</span>
      <div class="batch-actions">
        <a-button
          v-if="canUpdate"
          size="small"
          @click="openBatchAge"
        >
          批量设置年龄等级
        </a-button>
        <a-button
          v-if="canUpdate"
          size="small"
          @click="openBatchCover"
        >
          批量设置敏感封面
        </a-button>
        <a-button
          v-if="canDelete"
          size="small"
          danger
          @click="batchDeleteOpen = true"
        >
          批量删除
        </a-button>
        <a-button
          v-if="canRunAi"
          size="small"
          type="primary"
          :loading="batchSubmitting"
          :disabled="!selectedAiActionable"
          @click="submitBatchAi"
        >
          AI 判断选中游戏
        </a-button>
        <a-button
          v-if="canApplyAi && selectedPendingReview.length > 0"
          size="small"
          @click="submitBatchAiApprove"
        >
          批量采用高置信度 AI 结果（{{ selectedPendingReview.length }}）
        </a-button>
        <a-button size="small" type="text" @click="clearSelection">取消选择</a-button>
      </div>
    </div>

    <a-table
      :columns="columns"
      :data-source="items"
      :loading="loading"
      :row-selection="rowSelection"
      :row-key="(record: DtoGalgameListItem) => record.id ?? 0"
      :pagination="{
        current: query.page,
        pageSize: limit,
        total,
        showSizeChanger: false,
        showTotal: (count: number) => `共 ${count} 条`
      }"
      :scroll="{ x: 1520 }"
      @change="
        (pagination: { current?: number }) => {
          query.page = pagination.current ?? 1
        }
      "
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'cover'">          <a
            v-if="record.cover_url"
            :href="record.cover_url"
            target="_blank"
            rel="noopener noreferrer"
            :title="record.cover_sensitive ? '敏感封面，点击查看原图' : '点击查看原图'"
            class="cover-cell"
          >
            <SensitiveImage
              :src="record.cover_url"
              :alt="record.title || 'Galgame 封面'"
              :sensitive="record.cover_sensitive"
            />
          </a>
          <div v-else class="cover-cell cover-empty" title="未设置封面">
            <img :src="fallbackCover" alt="未设置封面" loading="lazy" />
          </div>
        </template>

        <template v-else-if="column.key === 'ai_classification'">
          <div class="ai-cell">
            <div class="ai-cell-row">
              <a-tag :color="aiCellColor(record)" class="ai-main-tag">
                {{ aiCellLabel(record) }}
              </a-tag>
              <a-tag v-if="record.ai_conflict" color="orange">证据冲突</a-tag>
              <a-tag
                v-if="record.ai_status === 'pending_review'"
                color="blue"
              >
                待审核
              </a-tag>
              <a-tag
                v-else-if="record.ai_status === 'approved'"
                color="green"
              >
                已采用
              </a-tag>
              <a-tag
                v-else-if="record.ai_status === 'rejected'"
                color="default"
              >
                已拒绝
              </a-tag>
            </div>
            <div v-if="(canRunAi || canReadAi || canApplyAi)" class="ai-actions">
              <a-button
                v-if="canRunAi && !isAiRunning(record)"
                size="small"
                type="link"
                :loading="aiActingId === record.id"
                @click="runAi(record, record.ai_status === 'failed')"
              >
                {{ record.ai_status === 'failed' ? '重试' : 'AI 判断' }}
              </a-button>
              <template
                v-if="canApplyAi && record.ai_status === 'pending_review'"
              >
                <a-button
                  v-if="record.ai_classification !== 'unknown'"
                  size="small"
                  type="link"
                  :loading="aiActingId === record.id"
                  @click="approveAi(record)"
                >
                  采用
                </a-button>
                <a-button
                  size="small"
                  type="link"
                  :loading="aiActingId === record.id"
                  @click="rejectAi(record)"
                >
                  拒绝
                </a-button>
                <a-button
                  size="small"
                  type="link"
                  :disabled="aiActingId === record.id"
                  @click="openOverride(record)"
                >
                  覆盖
                </a-button>
              </template>
              <a-button
                v-if="canReadAi && record.ai_status"
                size="small"
                type="link"
                :disabled="aiActingId === record.id"
                @click="openAiDetail(record)"
              >
                查看证据
              </a-button>
            </div>
          </div>
        </template>

        <template v-else-if="column.key === 'rating'">
          <template v-if="record.rating?.count">
            {{ record.rating.average?.toFixed(1) }}（{{ record.rating.count }}）
          </template>
          <template v-else>-</template>
        </template>

        <template v-else-if="column.key === 'favorite'">
          {{ record.statistics?.favorite_count ?? 0 }}
        </template>

        <template v-else-if="column.key === 'age_rating'">
          <a-select
            v-if="canUpdate && record.id"
            :value="record.age_rating ?? 0"
            size="small"
            class="age-select"
            :options="
              AGE_RATINGS.map((item) => ({
                value: item.value,
                label: item.label
              }))
            "
            :disabled="quickUpdatingId === record.id"
            @change="
              (value: number) =>
                quickUpdate(
                  record,
                  { age_rating: value as 0 | 1 | 2 | 3 | 4 | 5 },
                  `年龄等级已设为「${domainLabel(AGE_RATINGS, value)}」`
                )
            "
          />
          <span v-else>
            {{ domainLabel(AGE_RATINGS, record.age_rating) }}
          </span>
        </template>

        <template v-else-if="column.key === 'cover_sensitive'">
          <a-switch
            v-if="canUpdate && record.id"
            :checked="record.cover_sensitive ?? false"
            size="small"
            :disabled="quickUpdatingId === record.id"
            @change="
              (checked: unknown) =>
                quickUpdate(
                  record,
                  { cover_sensitive: Boolean(checked) },
                  checked ? '已标记为敏感封面' : '已取消敏感封面标记'
                )
            "
          />
          <a-tag v-else :color="record.cover_sensitive ? 'error' : 'default'">
            {{ record.cover_sensitive ? '敏感' : '正常' }}
          </a-tag>
        </template>

        <template v-else-if="column.dataIndex === 'status'">
          <a-tag
            :color="
              GALGAME_STATUS_OPTIONS.find(
                (option) => option.value === record.status
              )?.value === 1
                ? 'success'
                : GALGAME_STATUS_OPTIONS.find(
                      (option) => option.value === record.status
                    )?.value === 0
                  ? 'processing'
                  : 'default'
            "
          >
            {{
              GALGAME_STATUS_OPTIONS.find(
                (option) => option.value === record.status
              )?.label ?? record.status
            }}
          </a-tag>
        </template>

        <template v-else-if="column.key === 'actions'">
          <div v-if="canUpdate || canDelete" class="table-actions">
            <a-button
              v-for="action in QUICK_ACTIONS[record.status as number] ?? []"
              :key="action.status"
              size="small"
              :type="action.status === 1 ? 'primary' : 'default'"
              :loading="statusUpdating === record.id"
              @click="changeStatus(record, action.status)"
            >
              {{ action.label }}
            </a-button>
            <a-button v-if="canUpdate" size="small" @click="openEdit(record)">
              编辑
            </a-button>
            <a-popconfirm
              v-if="canDelete"
              :title="`确定删除「${record.title ?? record.id}」吗？此操作无法撤销。`"
              ok-text="删除"
              cancel-text="取消"
              @confirm="removeGalgame(record)"
            >
              <a-button
                size="small"
                danger
                :loading="rowDeletingId === record.id"
              >
                删除
              </a-button>
            </a-popconfirm>
          </div>
        </template>
      </template>
    </a-table>

    <a-modal
      v-model:open="batchAgeOpen"
      title="批量设置年龄等级"
      :confirm-loading="batchSubmitting"
      ok-text="确认修改"
      cancel-text="取消"
      @ok="submitBatchAge"
    >
      <p class="batch-modal-text">已选择 {{ selectedKeys.length }} 个游戏</p>
      <div class="batch-modal-field">
        <span class="batch-modal-label">年龄等级</span>
        <a-select
          v-model:value="batchAgeValue"
          class="batch-modal-select"
          :options="
            AGE_RATINGS.map((item) => ({
              value: item.value,
              label: item.label
            }))
          "
        />
      </div>
      <p class="batch-modal-warning">
        此操作会修改所有选中游戏的年龄等级。
      </p>
    </a-modal>

    <a-modal
      v-model:open="batchCoverOpen"
      title="批量设置敏感封面"
      :confirm-loading="batchSubmitting"
      ok-text="确认修改"
      cancel-text="取消"
      @ok="submitBatchCover"
    >
      <p class="batch-modal-text">已选择 {{ selectedKeys.length }} 个游戏</p>
      <a-radio-group v-model:value="batchCoverMark" class="batch-modal-radios">
        <a-radio value="mark">标记为敏感封面</a-radio>
        <a-radio value="unmark">取消敏感封面标记</a-radio>
      </a-radio-group>
      <p class="batch-modal-warning">
        敏感封面与游戏年龄等级无关，启用后前台默认对封面进行模糊处理。
      </p>
    </a-modal>

    <a-modal
      v-model:open="batchDeleteOpen"
      title="批量删除 Galgame"
      :confirm-loading="batchSubmitting"
      ok-text="确认删除"
      ok-type="danger"
      cancel-text="取消"
      @ok="submitBatchDelete"
    >
      <p class="batch-modal-text">已选择 {{ selectedKeys.length }} 个游戏</p>
      <p class="batch-modal-warning batch-delete-warning">
        删除为物理删除且不可恢复，游戏的收藏、评分、资源、帖子、图库等关联数据会一并删除；如只需暂时下架，请改用「隐藏」状态。
      </p>
    </a-modal>

    <a-modal
      v-model:open="aiOverrideOpen"
      title="人工覆盖 AI 分级"
      :confirm-loading="aiActingId !== null"
      ok-text="保存建议"
      cancel-text="取消"
      @ok="submitOverride"
    >
      <p class="batch-modal-text">
        覆盖后该游戏回到「待审核」状态，仍需点击「采用」才会写入正式年龄等级。
      </p>
      <div class="batch-modal-field">
        <span class="batch-modal-label">分级结论</span>
        <a-select
          v-model:value="aiOverrideValue"
          class="batch-modal-select"
          :options="AI_CLASSIFICATION_OPTIONS"
        />
      </div>
      <div class="batch-modal-field batch-modal-field-top">
        <span class="batch-modal-label">备注</span>
        <a-textarea
          v-model:value="aiOverrideReason"
          :rows="3"
          maxlength="1000"
          placeholder="选填：人工结论的依据"
        />
      </div>
    </a-modal>

    <a-modal
      v-model:open="aiDetailOpen"
      title="AI 分级证据"
      :confirm-loading="aiDetailLoading"
      :footer="null"
      width="640px"
    >
      <a-spin :spinning="aiDetailLoading">
        <template v-if="aiDetail">
          <div v-if="aiDetail.classification" class="ai-detail">
            <div class="ai-detail-header">
              <div class="ai-detail-verdict">
                <a-tag
                  :color="
                    AI_RESULT_COLORS[aiDetail.classification.classification ?? ''] ??
                    'default'
                  "
                  class="ai-detail-verdict-tag"
                >
                  {{
                    aiDetail.classification.classification
                      ? AI_RESULT_LABELS[aiDetail.classification.classification] ??
                        aiDetail.classification.classification
                      : '未知'
                  }}
                </a-tag>
                <a-tag v-if="aiDetail.classification.conflict" color="orange">
                  证据冲突
                </a-tag>
                <a-tag
                  :color="
                    aiDetail.classification.status === 'approved'
                      ? 'green'
                      : aiDetail.classification.status === 'pending_review'
                        ? 'blue'
                        : aiDetail.classification.status === 'failed'
                          ? 'default'
                          : 'orange'
                  "
                >
                  {{
                    AI_STATUS_LABELS[aiDetail.classification.status ?? ''] ??
                    aiDetail.classification.status
                  }}
                </a-tag>
              </div>
              <div class="ai-detail-model">
                <span v-if="aiDetail.classification.confidence">
                  置信度
                  {{
                    Math.round(aiDetail.classification.confidence * 100)
                  }}%
                </span>
                <span v-if="aiDetail.classification.model">
                  模型：{{ aiDetail.classification.model }}
                </span>
              </div>
            </div>

            <p class="ai-detail-reason">
              {{ aiDetail.classification.reason }}
            </p>

            <div v-if="aiDetail.classification.error_message" class="ai-detail-error">
              {{ aiDetail.classification.error_message }}
            </div>

            <div
              v-if="(aiDetail.classification.evidences ?? []).length > 0"
              class="ai-detail-evidence"
            >
              <div
                v-for="evidence in aiDetail.classification.evidences"
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
            <p v-else class="ai-detail-empty">
              该结果没有引用任何网页证据。
            </p>
          </div>
          <a-empty v-else description="该游戏还没有 AI 判断记录">
            <a-button v-if="canRunAi" type="primary" @click="runAiFromDetail">
              立即 AI 判断
            </a-button>
          </a-empty>
        </template>
        <a-empty v-else description="暂无记录" />
      </a-spin>
    </a-modal>
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
  width: 140px;
}

.filter-keyword {
  flex: 1;
  min-width: 200px;
}

.batch-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
  padding: 10px 14px;
  border: 1px solid color-mix(in srgb, var(--color-primary) 35%, transparent);
  border-radius: var(--radius-kun-md);
  background: color-mix(in srgb, var(--color-primary) 8%, transparent);
}

.batch-count {
  color: var(--color-foreground);
  font-size: 14px;
  font-weight: 600;
}

.batch-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.age-select {
  width: 96px;
}

.cover-cell {
  display: block;
  width: 48px;
  height: 64px;
  overflow: hidden;
  border-radius: var(--radius-kun-sm);
}

.cover-cell img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.table-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.batch-modal-text {
  margin: 0 0 12px;
  color: var(--color-default-600);
  font-size: 14px;
}

.batch-modal-field {
  display: flex;
  align-items: center;
  gap: 12px;
}

.batch-modal-label {
  color: var(--color-foreground);
  font-size: 14px;
}

.batch-modal-select {
  width: 160px;
}

.batch-modal-radios {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.batch-modal-warning {
  margin: 14px 0 0;
  color: var(--color-default-500);
  font-size: 13px;
}

.batch-delete-warning {
  color: var(--color-danger);
}

.ai-cell {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  min-width: 150px;
}

.ai-cell-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
}

.ai-main-tag {
  margin-inline-end: 0;
}

.ai-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 4px;
}

.ai-detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.ai-detail-header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.ai-detail-verdict {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.ai-detail-verdict-tag {
  padding: 2px 10px;
  font-size: 14px;
  font-weight: 600;
}

.ai-detail-model {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  color: var(--color-default-500);
  font-size: 12px;
}

.ai-detail-reason {
  margin: 0;
  color: var(--color-foreground);
  font-size: 14px;
  line-height: 1.7;
  white-space: pre-wrap;
}

.ai-detail-error {
  padding: 8px 12px;
  border: 1px solid color-mix(in srgb, var(--color-danger) 40%, transparent);
  border-radius: var(--radius-kun-sm);
  background: color-mix(in srgb, var(--color-danger) 8%, transparent);
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
  border: 1px solid var(--color-border, rgba(128, 128, 128, 0.2));
  border-radius: var(--radius-kun-sm);
}

.ai-detail-evidence-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.ai-detail-evidence-head a {
  color: var(--color-primary);
  word-break: break-all;
}

.ai-detail-quote {
  margin: 8px 0 0;
  padding-left: 10px;
  border-left: 3px solid color-mix(in srgb, var(--color-primary) 45%, transparent);
  color: var(--color-default-600);
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
}

.ai-detail-empty {
  margin: 0;
  color: var(--color-default-500);
  font-size: 13px;
}

.batch-modal-field-top {
  align-items: flex-start;
  margin-top: 14px;
}

.batch-modal-field-top .batch-modal-label {
  padding-top: 6px;
}

</style>
