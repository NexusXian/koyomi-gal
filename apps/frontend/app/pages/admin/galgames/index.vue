<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  batchUpdateGalgames,
  getAdminGalgame,
  listAdminGalgames,
  reviewGalgame
} from '~/api/generated/admin/admin'
import { updateGalgame } from '~/api/generated/galgames/galgames'
import type { DtoGalgameListItem } from '~/api/generated/models'
import { AGE_RATINGS, GALGAME_SORTS, domainLabel, domainSortSlug } from '~/constants/domain'

useSeoMeta({ title: 'Galgame 审核 - Koyomi' })

const { has } = usePermissions()
const query = reactive({
  status: undefined as number | undefined,
  ageRating: undefined as number | undefined,
  coverSensitive: undefined as boolean | undefined,
  keyword: '',
  sort: 0,
  page: 1
})

const limit = 20
const items = ref<DtoGalgameListItem[]>([])
const total = ref(0)
const loading = ref(false)
const statusUpdating = ref<number | null>(null)
const quickUpdatingId = ref<number | null>(null)

const selectedKeys = ref<number[]>([])
const batchAgeOpen = ref(false)
const batchAgeValue = ref<number>(0)
const batchCoverOpen = ref(false)
const batchCoverMark = ref<'mark' | 'unmark'>('mark')
const batchSubmitting = ref(false)

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
        limit
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
  () => [query.status, query.ageRating, query.coverSensitive, query.sort, query.page],
  () => {
    void load()
  }
)

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

const canUpdate = computed(() => has('galgame:update'))

const columns = computed<TableColumnsType>(() => [
  { title: 'ID', dataIndex: 'id', width: 70 },
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
    title: '评分',
    key: 'rating',
    width: 90
  },
  {
    title: '收藏',
    key: 'favorite',
    width: 70
  },
  ...(canUpdate.value
    ? [{ title: '操作', key: 'actions', width: 230, fixed: 'right' as const }]
    : [])
])

const rowSelection = computed(() =>
  canUpdate.value
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

    <div v-if="canUpdate && selectedKeys.length > 0" class="batch-bar">
      <span class="batch-count">已选择 {{ selectedKeys.length }} 项</span>
      <div class="batch-actions">
        <a-button size="small" @click="openBatchAge">批量设置年龄等级</a-button>
        <a-button size="small" @click="openBatchCover">批量设置敏感封面</a-button>
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
      :scroll="{ x: 1100 }"
      @change="
        (pagination: { current?: number }) => {
          query.page = pagination.current ?? 1
        }
      "
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'rating'">
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
          <div v-if="canUpdate" class="table-actions">
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
            <a-button size="small" @click="openEdit(record)">
              编辑
            </a-button>
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
</style>
