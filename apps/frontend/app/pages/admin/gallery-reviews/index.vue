<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  approveGalleryImage,
  batchReviewGalleryImages,
  listGalleryReviews,
  rejectGalleryImage
} from '~/api/generated/admin/admin'
import type { DtoGalleryReviewItemData } from '~/api/generated/models'
import {
  GALLERY_IMAGE_STATUS,
  GALLERY_IMAGE_TYPES,
  GALLERY_SOURCE_TYPES,
  domainLabel
} from '~/constants/domain'

useSeoMeta({ title: '插图审核 - Koyomi' })

const { has } = usePermissions()
const canReview = computed(() => has('galgame_gallery:review'))

const query = reactive({
  status: 0 as number | undefined,
  sourceType: undefined as number | undefined,
  galgameId: undefined as number | undefined,
  page: 1
})
const limit = 20
const items = ref<DtoGalleryReviewItemData[]>([])
const total = ref(0)
const loading = ref(false)
const actingId = ref<number | null>(null)
const selectedIds = ref<number[]>([])
const batchLoading = ref<'approve' | 'reject' | null>(null)

const rejectOpen = ref(false)
const rejectSaving = ref(false)
const rejectReason = ref('')
const rejectTargetIds = ref<number[]>([])

const STATUS_OPTIONS = GALLERY_IMAGE_STATUS.map((item) => ({
  value: item.value,
  label: item.label
}))

const SOURCE_OPTIONS = GALLERY_SOURCE_TYPES.map((item) => ({
  value: item.value,
  label: item.label
}))

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = unwrapApiData(
      await listGalleryReviews({
        status: query.status,
        source_type: query.sourceType,
        galgame_id: query.galgameId || undefined,
        page: query.page,
        limit
      })
    )
    items.value = data.items ?? []
    total.value = data.total ?? 0
    selectedIds.value = []
  } catch (error) {
    message.error(getApiErrorMessage(error, '查询插图审核列表失败'))
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())

watch(
  () => [query.status, query.sourceType, query.galgameId],
  () => {
    if (query.page !== 1) {
      query.page = 1
    } else {
      void load()
    }
  }
)

watch(
  () => query.page,
  () => void load()
)

async function approve(item: DtoGalleryReviewItemData): Promise<void> {
  if (!item.id) {
    return
  }
  actingId.value = item.id
  try {
    await approveGalleryImage(item.id)
    message.success('已通过该插图')
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '通过失败'))
  } finally {
    actingId.value = null
  }
}

function askReject(item: DtoGalleryReviewItemData): void {
  if (!item.id) {
    return
  }
  rejectTargetIds.value = [item.id]
  rejectReason.value = ''
  rejectOpen.value = true
}

function askBatchReject(): void {
  rejectTargetIds.value = [...selectedIds.value]
  rejectReason.value = ''
  rejectOpen.value = true
}

async function submitReject(): Promise<void> {
  if (rejectTargetIds.value.length === 0) {
    return
  }
  rejectSaving.value = true
  try {
    if (rejectTargetIds.value.length === 1) {
      await rejectGalleryImage(rejectTargetIds.value[0]!, {
        reason: rejectReason.value.trim() || undefined
      })
    } else {
      await batchReviewGalleryImages({
        ids: rejectTargetIds.value,
        action: 'reject',
        reason: rejectReason.value.trim() || undefined
      })
    }
    message.success(`已拒绝 ${rejectTargetIds.value.length} 张插图`)
    rejectOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '拒绝失败'))
  } finally {
    rejectSaving.value = false
  }
}

async function runBatchApprove(): Promise<void> {
  if (selectedIds.value.length === 0) {
    return
  }
  batchLoading.value = 'approve'
  try {
    const reviewed = unwrapApiData(
      await batchReviewGalleryImages({
        ids: selectedIds.value,
        action: 'approve'
      })
    )
    message.success(`已批量通过 ${reviewed ?? 0} 张插图`)
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '批量通过失败'))
  } finally {
    batchLoading.value = null
  }
}

const rowSelection = computed(() =>
  query.status === 0
    ? {
        selectedRowKeys: selectedIds.value,
        onChange: (keys: (number | string)[]) => {
          selectedIds.value = keys.map(Number)
        }
      }
    : undefined
)

function formatDateTime(value?: string): string {
  return value ? value.replace('T', ' ').slice(0, 19) : '-'
}

function sourceHost(item: DtoGalleryReviewItemData): string {
  if (item.source_type !== 1 || !item.external_url) {
    return ''
  }
  try {
    return new URL(item.external_url).host
  } catch {
    return ''
  }
}

const columns: TableColumnsType = [
  { title: '预览', key: 'thumb', width: 96 },
  { title: '游戏', key: 'galgame', ellipsis: true },
  { title: '来源', key: 'source', width: 130 },
  { title: '类型', key: 'type', width: 90 },
  { title: '提交者', key: 'submitter', width: 120 },
  { title: '状态', key: 'status', width: 100 },
  { title: '提交时间', key: 'created', width: 165 },
  { title: '操作', key: 'actions', width: 210, fixed: 'right' as const }
]
</script>

<template>
  <div class="gallery-review">
    <KunNull v-if="!canReview" text="没有插图审核权限" />

    <template v-else>
      <div class="gallery-review-toolbar">
        <KunHeader
          name="插图审核"
          description="上传与外部链接导入的游戏画面在此审核，通过后才会展示在游戏详情页。"
          scale="h3"
        />
        <a-space wrap>
          <template v-if="query.status === 0">
            <span v-if="selectedIds.length > 0" class="gallery-review-selected">
              已选 {{ selectedIds.length }} 项
            </span>
            <a-popconfirm
              :title="`确认批量通过所选 ${selectedIds.length} 张插图？`"
              ok-text="确认"
              cancel-text="取消"
              :disabled="selectedIds.length === 0"
              @confirm="runBatchApprove"
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
            <a-button
              size="small"
              danger
              :disabled="selectedIds.length === 0"
              @click="askBatchReject"
            >
              批量拒绝
            </a-button>
          </template>
          <a-select
            v-model:value="query.status"
            class="gallery-review-filter"
            :options="STATUS_OPTIONS"
            placeholder="全部状态"
            allow-clear
          />
          <a-select
            v-model:value="query.sourceType"
            class="gallery-review-filter"
            :options="SOURCE_OPTIONS"
            placeholder="全部来源"
            allow-clear
          />
          <a-input-number
            v-model:value="query.galgameId"
            class="gallery-review-galgame"
            placeholder="游戏 ID"
            :min="1"
            :precision="0"
          />
        </a-space>
      </div>

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
        :row-selection="rowSelection"
        row-key="id"
        size="middle"
        :scroll="{ x: 1180 }"
        @change="(pagination: { current?: number }) => { query.page = pagination.current ?? 1 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'thumb'">
            <a
              v-if="record.url"
              :href="record.url"
              target="_blank"
              rel="noopener noreferrer"
            >
              <img
                :src="record.url"
                class="gallery-review-thumb"
                loading="lazy"
                referrerpolicy="no-referrer"
                alt=""
              />
            </a>
            <span v-else>-</span>
          </template>
          <template v-else-if="column.key === 'galgame'">
            <div class="gallery-review-title-cell">
              <span class="gallery-review-title-main">
                {{ record.galgame_title || `#${record.galgame_id}` }}
              </span>
              <a-button
                type="link"
                size="small"
                class="gallery-review-link"
                @click="navigateTo(`/galgames/${record.galgame_id}`)"
              >
                站内 #{{ record.galgame_id }}
              </a-button>
            </div>
          </template>
          <template v-else-if="column.key === 'source'">
            <a-tag :color="record.source_type === 1 ? 'blue' : 'default'">
              {{ domainLabel(GALLERY_SOURCE_TYPES, record.source_type) }}
            </a-tag>
            <span v-if="sourceHost(record)" class="gallery-review-host">
              {{ sourceHost(record) }}
            </span>
          </template>
          <template v-else-if="column.key === 'type'">
            {{ domainLabel(GALLERY_IMAGE_TYPES, record.image_type) }}
          </template>
          <template v-else-if="column.key === 'submitter'">
            <div class="gallery-review-title-cell">
              <span>{{ record.created_by_username || '-' }}</span>
              <span v-if="record.is_spoiler" class="gallery-review-spoiler">
                剧透
              </span>
            </div>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tooltip :title="record.reject_reason">
              <a-tag
                :color="
                  GALLERY_IMAGE_STATUS.find((s) => s.value === record.status)?.color
                "
              >
                {{ domainLabel(GALLERY_IMAGE_STATUS, record.status) }}
              </a-tag>
            </a-tooltip>
          </template>
          <template v-else-if="column.key === 'created'">
            {{ formatDateTime(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <template v-if="record.status === 0">
              <a-button
                size="small"
                type="primary"
                :loading="actingId === record.id"
                @click="approve(record)"
              >
                通过
              </a-button>
              <a-button
                size="small"
                danger
                class="gallery-review-action"
                @click="askReject(record)"
              >
                拒绝
              </a-button>
            </template>
            <span
              v-else
              class="gallery-review-reviewed"
            >
              {{ record.reviewed_by_username || '-' }}
              {{ record.reviewed_at ? `· ${formatDateTime(record.reviewed_at)}` : '' }}
            </span>
          </template>
        </template>
        <template #emptyText>
          <KunNull text="没有符合条件的插图" />
        </template>
      </a-table>

      <a-modal
        v-model:open="rejectOpen"
        title="拒绝插图"
        :confirm-loading="rejectSaving"
        ok-text="确认拒绝"
        ok-type="danger"
        cancel-text="取消"
        @ok="submitReject"
      >
        <p class="gallery-review-reject-count">
          将拒绝 {{ rejectTargetIds.length }} 张插图
        </p>
        <a-textarea
          v-model:value="rejectReason"
          :rows="3"
          :maxlength="500"
          placeholder="拒绝理由（可选，会展示给提交者）"
        />
      </a-modal>
    </template>
  </div>
</template>

<style scoped>
.gallery-review-toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.gallery-review-filter {
  width: 120px;
  flex: none;
}

.gallery-review-galgame {
  width: 110px;
  flex: none;
}

.gallery-review-selected {
  font-size: 12px;
  opacity: 0.75;
}

.gallery-review-thumb {
  display: block;
  width: 72px;
  height: 40px;
  object-fit: cover;
  border-radius: 4px;
}

.gallery-review-title-cell {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.gallery-review-title-main {
  font-weight: 600;
}

.gallery-review-link {
  padding: 0;
  height: auto;
  font-size: 12px;
  justify-content: flex-start;
}

.gallery-review-host {
  display: block;
  margin-top: 2px;
  color: var(--color-default-400);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 110px;
}

.gallery-review-spoiler {
  color: var(--color-default-400);
  font-size: 12px;
}

.gallery-review-action {
  margin-left: 8px;
}

.gallery-review-reviewed {
  color: var(--color-default-400);
  font-size: 12px;
}

.gallery-review-reject-count {
  margin: 0 0 10px;
  color: var(--color-default-500);
  font-size: 13px;
}
</style>
