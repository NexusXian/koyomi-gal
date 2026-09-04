<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  listImportBatches,
  listImportProviders,
  searchImportGames
} from '~/api/generated/admin-import/admin-import'
import type {
  DtoImportJobData,
  DtoImportSearchItem
} from '~/api/generated/models'

useSeoMeta({ title: '外部数据导入 - Koyomi' })

const { has } = usePermissions()
const canImport = computed(() => has('galgame:import'))
const canBatch = computed(() => has('galgame:import:batch'))

const providers = ref<string[]>([])
const providersLoading = ref(false)

const searchState = reactive({
  provider: 'vndb',
  q: ''
})
const searching = ref(false)
const searched = ref(false)
const results = ref<DtoImportSearchItem[]>([])

const previewState = reactive({
  open: false,
  provider: 'vndb',
  externalId: ''
})

const jobsQuery = reactive({
  status: undefined as number | undefined,
  page: 1
})
const jobsLimit = 10
const jobs = ref<DtoImportJobData[]>([])
const jobsTotal = ref(0)
const jobsLoading = ref(false)

const JOB_STATUS_OPTIONS = [
  { value: 0, label: '待处理' },
  { value: 1, label: '运行中' },
  { value: 2, label: '成功' },
  { value: 3, label: '失败' },
  { value: 4, label: '已取消' }
]

let jobsTimer: ReturnType<typeof setInterval> | null = null

async function loadProviders(): Promise<void> {
  providersLoading.value = true
  try {
    const data = unwrapApiData(await listImportProviders())
    providers.value = data.providers ?? []
    if (
      providers.value.length > 0 &&
      !providers.value.includes(searchState.provider)
    ) {
      searchState.provider = providers.value[0] ?? 'vndb'
    }
  } catch (error) {
    message.error(getApiErrorMessage(error, '获取数据源失败'))
  } finally {
    providersLoading.value = false
  }
}

async function search(payload: { provider: string; q: string }): Promise<void> {
  searchState.provider = payload.provider
  searchState.q = payload.q
  searching.value = true
  try {
    const data = unwrapApiData(
      await searchImportGames({
        provider: payload.provider,
        q: payload.q,
        limit: 20
      })
    )
    results.value = data.items ?? []
    searched.value = true
  } catch (error) {
    message.error(getApiErrorMessage(error, '搜索外部作品失败'))
  } finally {
    searching.value = false
  }
}

async function loadJobs(): Promise<void> {
  if (!canBatch.value) {
    return
  }
  jobsLoading.value = true
  try {
    const data = unwrapApiData(
      await listImportBatches({
        status: jobsQuery.status,
        page: jobsQuery.page,
        limit: jobsLimit
      })
    )
    jobs.value = data.items ?? []
    jobsTotal.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '查询导入任务失败'))
  } finally {
    jobsLoading.value = false
  }
}

function hasActiveJob(): boolean {
  return jobs.value.some((job) => job.status === 0 || job.status === 1)
}

function syncJobsPolling(): void {
  if (hasActiveJob() && jobsTimer === null) {
    jobsTimer = setInterval(() => {
      if (!hasActiveJob()) {
        return
      }
      void loadJobs()
    }, 5000)
  } else if (!hasActiveJob() && jobsTimer !== null) {
    clearInterval(jobsTimer)
    jobsTimer = null
    void loadJobs()
  }
}

watch(jobs, () => syncJobsPolling(), { deep: true })

onMounted(() => {
  if (canImport.value) {
    void loadProviders()
  }
  void loadJobs()
})

onUnmounted(() => {
  if (jobsTimer !== null) {
    clearInterval(jobsTimer)
    jobsTimer = null
  }
})

watch(
  () => [jobsQuery.status, jobsQuery.page],
  () => {
    void loadJobs()
  }
)

function openPreview(item: DtoImportSearchItem): void {
  previewState.provider = item.game?.source ?? searchState.provider
  previewState.externalId = item.game?.external_id ?? ''
  if (previewState.externalId) {
    previewState.open = true
  }
}

async function quickImport(item: DtoImportSearchItem): Promise<void> {
  const game = item.game
  if (!game?.external_id) {
    return
  }
  previewState.provider = game.source ?? searchState.provider
  previewState.externalId = game.external_id
  previewState.open = true
}

function onBatchCreated(job: DtoImportJobData): void {
  jobsQuery.page = 1
  jobsQuery.status = undefined
  void loadJobs()
  void navigateTo(`/admin/import/jobs/${job.id}`)
}

const resultColumns = computed<TableColumnsType>(() => [
  { title: '封面', key: 'cover', width: 70 },
  { title: '标题', key: 'title', ellipsis: true },
  { title: '原始标题', key: 'original_title', ellipsis: true },
  { title: '开发商', key: 'developer', width: 140, ellipsis: true },
  { title: '发行时间', key: 'release_date', width: 110 },
  { title: 'VNDB 评分', key: 'rating', width: 110 },
  { title: '状态', key: 'duplicate', width: 90 },
  { title: '操作', key: 'actions', width: 150 }
])

const jobColumns = computed<TableColumnsType>(() => [
  { title: 'ID', dataIndex: 'id', width: 70 },
  { title: '数据源', dataIndex: 'provider', width: 90 },
  { title: '进度', key: 'progress', width: 220 },
  { title: '创建时间', key: 'created_at', width: 170 },
  { title: '操作', key: 'actions', width: 90 }
])

function formatDateTime(value?: string): string {
  return value ? value.replace('T', ' ').slice(0, 19) : '-'
}

function jobsTotalText(count: number): string {
  return `共 ${count} 条`
}

function onJobsTableChange(pagination: { current?: number }): void {
  jobsQuery.page = pagination.current ?? 1
}

const DUPLICATE_LABELS: Record<string, { text: string; color: string }> = {
  none: { text: '可导入', color: 'green' },
  possible: { text: '疑似重复', color: 'orange' },
  already_imported: { text: '已导入', color: 'blue' }
}
</script>

<template>
  <div class="admin-import">
    <KunNull v-if="!canImport && !canBatch" text="没有外部数据导入权限" />

    <template v-else>
      <KunCard v-if="canImport" padding="md" class="admin-import-section">
        <h3 class="admin-import-title">搜索外部作品</h3>
        <ImportSearch
          :providers="providers.length > 0 ? providers : ['vndb']"
          :loading="searching"
          @search="search"
        />
        <a-alert
          class="admin-import-hint"
          type="info"
          show-icon
          :message="`当前数据源：${searchState.provider.toUpperCase()}（VNDB）`"
        />

        <a-table
          v-if="searched"
          class="admin-import-results"
          :columns="resultColumns"
          :data-source="results"
          :loading="searching"
          :pagination="false"
          row-key="(item) => item.game?.external_id"
          size="middle"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'cover'">
              <img
                v-if="record.game?.cover_url"
                :src="record.game.cover_url"
                class="admin-import-cover"
                loading="lazy"
                referrerpolicy="no-referrer"
                alt=""
              />
              <span v-else>-</span>
            </template>
            <template v-else-if="column.key === 'title'">
              <div class="admin-import-title-cell">
                <span class="admin-import-title-main">
                  {{ record.game?.title ?? '-' }}
                </span>
                <span class="admin-import-title-sub">
                  {{ record.game?.external_id }}
                </span>
              </div>
            </template>
            <template v-else-if="column.key === 'original_title'">
              {{ record.game?.original_title || '-' }}
            </template>
            <template v-else-if="column.key === 'developer'">
              {{ record.game?.developer || '-' }}
            </template>
            <template v-else-if="column.key === 'release_date'">
              {{ record.game?.release_date || '-' }}
            </template>
            <template v-else-if="column.key === 'rating'">
              <template v-if="record.game?.rating != null">
                {{ record.game.rating.toFixed(2) }}（{{ record.game.rating_count ?? 0 }} 票）
              </template>
              <span v-else>暂无</span>
            </template>
            <template v-else-if="column.key === 'duplicate'">
              <a-tag
                :color="DUPLICATE_LABELS[record.duplicate_status ?? 'none']?.color"
              >
                {{ DUPLICATE_LABELS[record.duplicate_status ?? 'none']?.text }}
              </a-tag>
            </template>
            <template v-else-if="column.key === 'actions'">
              <a-button size="small" @click="openPreview(record)">预览</a-button>
              <a-button
                size="small"
                type="primary"
                class="admin-import-action"
                @click="quickImport(record)"
              >
                导入
              </a-button>
            </template>
          </template>
          <template #emptyText>
            <KunNull text="没有搜索结果" />
          </template>
        </a-table>
      </KunCard>

      <KunCard v-if="canBatch" padding="md" class="admin-import-section">
        <h3 class="admin-import-title">批量导入</h3>
        <ImportBatchForm
          :providers="providers.length > 0 ? providers : ['vndb']"
          @created="onBatchCreated"
        />
      </KunCard>

      <KunCard v-if="canBatch" padding="md" class="admin-import-section">
        <div class="admin-import-jobs-head">
          <h3 class="admin-import-title">导入任务</h3>
          <a-select
            v-model:value="jobsQuery.status"
            class="admin-import-jobs-filter"
            allow-clear
            placeholder="任务状态"
            :options="JOB_STATUS_OPTIONS"
          />
        </div>
        <a-table
          :columns="jobColumns"
          :data-source="jobs"
          :loading="jobsLoading"
          :pagination="{
            current: jobsQuery.page,
            pageSize: jobsLimit,
            total: jobsTotal,
            showSizeChanger: false,
            showTotal: jobsTotalText
          }"
          row-key="id"
          size="middle"
          @change="onJobsTableChange"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'progress'">
              <ImportJobProgress :job="record" compact />
            </template>
            <template v-else-if="column.key === 'created_at'">
              {{ formatDateTime(record.created_at) }}
            </template>
            <template v-else-if="column.key === 'actions'">
              <a-button
                size="small"
                @click="navigateTo(`/admin/import/jobs/${record.id}`)"
              >
                详情
              </a-button>
            </template>
          </template>
        </a-table>
      </KunCard>

      <ImportPreview
        v-model:open="previewState.open"
        :provider="previewState.provider"
        :external-id="previewState.externalId"
        @imported="search({ provider: searchState.provider, q: searchState.q })"
      />
    </template>
  </div>
</template>

<style scoped>
.admin-import-section {
  margin-bottom: 18px;
}

.admin-import-title {
  margin: 0 0 14px;
  font-size: 16px;
}

.admin-import-hint {
  margin-top: 14px;
}

.admin-import-results {
  margin-top: 14px;
}

.admin-import-cover {
  width: 44px;
  height: 60px;
  object-fit: cover;
  border-radius: 4px;
  display: block;
}

.admin-import-title-cell {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.admin-import-title-main {
  font-weight: 600;
}

.admin-import-title-sub {
  font-size: 12px;
  opacity: 0.65;
}

.admin-import-action {
  margin-left: 8px;
}

.admin-import-jobs-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.admin-import-jobs-head .admin-import-title {
  margin: 0;
}

.admin-import-jobs-filter {
  width: 140px;
}
</style>
