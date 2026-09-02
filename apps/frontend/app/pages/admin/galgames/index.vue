<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  getAdminGalgame,
  listAdminGalgames,
  reviewGalgame
} from '~/api/generated/admin/admin'
import { updateGalgame } from '~/api/generated/galgames/galgames'
import type { DtoGalgameListItem } from '~/api/generated/models'
import { GALGAME_SORTS, domainSortSlug } from '~/constants/domain'

useSeoMeta({ title: 'Galgame 审核 - Koyomi' })

const { has } = usePermissions()
const query = reactive({
  status: undefined as number | undefined,
  keyword: '',
  sort: 0,
  page: 1
})

const limit = 20
const items = ref<DtoGalgameListItem[]>([])
const total = ref(0)
const loading = ref(false)
const statusUpdating = ref<number | null>(null)

const GALGAME_STATUS_OPTIONS = [
  { value: 0, label: '待审核' },
  { value: 1, label: '已发布' },
  { value: 2, label: '已拒绝' },
  { value: 3, label: '已隐藏' }
]

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = unwrapApiData(
      await listAdminGalgames({
        status: query.status,
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
  () => [query.status, query.sort, query.page],
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
      age_rating: (detail.age_rating ?? 0) as 0 | 1 | 2 | 3,
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
    width: 140,
    ellipsis: true
  },
  {
    title: '状态',
    dataIndex: 'status',
    width: 100
  },
  {
    title: '评分',
    key: 'rating',
    width: 100
  },
  {
    title: '收藏',
    key: 'favorite',
    width: 80
  },
  ...(has('galgame:update')
    ? [{ title: '操作', key: 'actions', width: 230 }]
    : [])
])
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
          <div v-if="has('galgame:update')" class="table-actions">
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
            <a-button size="small" :href="`/galgames/${record.id}/edit`">
              编辑
            </a-button>
          </div>
        </template>
      </template>
    </a-table>
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

.table-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
