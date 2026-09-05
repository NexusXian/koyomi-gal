<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  listAdminNovelVolumes,
  listAdminNovels,
  reviewNovel,
  reviewNovelVolume
} from '~/api/generated/admin/admin'
import type {
  DtoAdminVolumeListItem,
  DtoNovelListItem,
  ListAdminNovelVolumesStatus,
  ListAdminNovelsStatus
} from '~/api/generated/models'
import {
  GALGAME_STATUS,
  NOVEL_RELEASE_STATUS,
  NOVEL_STATUS,
  domainLabel,
  domainSlug,
  formatDate
} from '~/constants/domain'

const activeTab = ref<'novels' | 'volumes'>('novels')

// ---- 小说列表 ----
const novelItems = ref<DtoNovelListItem[]>([])
const novelTotal = ref(0)
const novelPage = ref(1)
const novelStatus = ref<number | undefined>(undefined)
const novelKeyword = ref('')
const novelLoading = ref(false)
const reviewingId = ref<number | null>(null)
const rejectReason = ref('')
const rejectTarget = ref<{ type: 'novel' | 'volume'; id: number; title: string } | null>(null)
const rejectOpen = ref(false)

const novelColumns: TableColumnsType = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '封面', dataIndex: 'cover', width: 60 },
  { title: '标题', dataIndex: 'title' },
  { title: '作者', dataIndex: 'author', width: 120 },
  { title: '出版社', dataIndex: 'publisher', width: 130 },
  { title: '状态', dataIndex: 'status', width: 90 },
  { title: '连载', dataIndex: 'release_status', width: 90 },
  { title: '卷数', dataIndex: 'volume_count', width: 70 },
  { title: '更新时间', dataIndex: 'updated_at', width: 160 },
  { title: '操作', dataIndex: 'actions', width: 260 }
]

async function loadNovels(): Promise<void> {
  novelLoading.value = true
  try {
    const data = unwrapApiData(
      await listAdminNovels({
        status: novelStatus.value as ListAdminNovelsStatus | undefined,
        keyword: novelKeyword.value.trim() || undefined,
        page: novelPage.value,
        limit: 20
      })
    )
    novelItems.value = data.items ?? []
    novelTotal.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '查询小说失败'))
  } finally {
    novelLoading.value = false
  }
}

async function approveNovel(record: DtoNovelListItem): Promise<void> {
  reviewingId.value = record.id ?? null
  try {
    await reviewNovel(record.id ?? 0, { status: 1 })
    message.success('已通过')
    await loadNovels()
  } catch (error) {
    message.error(getApiErrorMessage(error, '审核失败'))
  } finally {
    reviewingId.value = null
  }
}

function openRejectNovel(record: DtoNovelListItem): void {
  rejectTarget.value = { type: 'novel', id: record.id ?? 0, title: record.title ?? '' }
  rejectReason.value = ''
  rejectOpen.value = true
}

// ---- 卷册列表 ----
const volumeItems = ref<DtoAdminVolumeListItem[]>([])
const volumeTotal = ref(0)
const volumePage = ref(1)
const volumeStatus = ref<number | undefined>(undefined)
const volumeLoading = ref(false)
const reviewingVolumeId = ref<number | null>(null)

const volumeColumns: TableColumnsType = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '小说', dataIndex: 'novel_title', ellipsis: true },
  { title: '卷号', dataIndex: 'volume_number', width: 80 },
  { title: '标题', dataIndex: 'title' },
  { title: 'ISBN', dataIndex: 'isbn', width: 140 },
  { title: '发售日', dataIndex: 'release_date', width: 110 },
  { title: '状态', dataIndex: 'status', width: 90 },
  { title: '更新时间', dataIndex: 'updated_at', width: 160 },
  { title: '操作', dataIndex: 'actions', width: 260 }
]

async function loadVolumes(): Promise<void> {
  volumeLoading.value = true
  try {
    const data = unwrapApiData(
      await listAdminNovelVolumes({
        status: volumeStatus.value as ListAdminNovelVolumesStatus | undefined,
        page: volumePage.value,
        limit: 20
      })
    )
    volumeItems.value = data.items ?? []
    volumeTotal.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '查询卷册失败'))
  } finally {
    volumeLoading.value = false
  }
}

async function approveVolume(record: DtoAdminVolumeListItem): Promise<void> {
  reviewingVolumeId.value = record.id ?? null
  try {
    await reviewNovelVolume(record.id ?? 0, { status: 1 })
    message.success('已通过')
    await loadVolumes()
  } catch (error) {
    message.error(getApiErrorMessage(error, '审核失败'))
  } finally {
    reviewingVolumeId.value = null
  }
}

function openRejectVolume(record: DtoAdminVolumeListItem): void {
  rejectTarget.value = { type: 'volume', id: record.id ?? 0, title: record.title || `Vol.${record.volume_number}` }
  rejectReason.value = ''
  rejectOpen.value = true
}

async function submitReject(): Promise<void> {
  if (!rejectTarget.value) {
    return
  }
  const { type, id } = rejectTarget.value
  try {
    if (type === 'novel') {
      await reviewNovel(id, { status: 2, reason: rejectReason.value.trim() || undefined })
    } else {
      await reviewNovelVolume(id, { status: 2, reason: rejectReason.value.trim() || undefined })
    }
    message.success('已拒绝')
    rejectOpen.value = false
    if (type === 'novel') {
      await loadNovels()
    } else {
      await loadVolumes()
    }
  } catch (error) {
    message.error(getApiErrorMessage(error, '审核失败'))
  }
}

watch(activeTab, () => {
  novelPage.value = 1
  volumePage.value = 1
  if (activeTab.value === 'volumes') {
    void loadVolumes()
  }
})

watch([novelStatus, novelPage], () => loadNovels())
watch([volumeStatus, volumePage], () => loadVolumes())

function novelLink(record: DtoNovelListItem): string {
  return `/novels/${record.id}`
}

onMounted(() => {
  void loadNovels()
})
</script>

<template>
  <AppPageContainer title="小说管理" description="管理小说条目与卷册审核" width="wide">
    <a-tabs v-model:active-key="activeTab" type="card">
      <a-tab-pane key="novels" :tab="`小说审核`">
        <div class="toolbar">
          <a-input-search
            v-model:value="novelKeyword"
            placeholder="搜索标题 / 作者"
            allow-clear
            class="toolbar-search"
            @search="novelPage = 1; loadNovels()"
          />
          <a-select
            v-model:value="novelStatus"
            class="toolbar-select"
            placeholder="状态"
            allow-clear
            :options="
              NOVEL_STATUS.map((item) => ({
                value: item.value,
                label: item.label
              }))
            "
          />
        </div>

        <a-table
          :columns="novelColumns"
          :data-source="novelItems"
          :loading="novelLoading"
          :row-key="(record: DtoNovelListItem) => record.id ?? 0"
          :pagination="{
            current: novelPage,
            pageSize: 20,
            total: novelTotal,
            showSizeChanger: false,
            showTotal: (count: number) => `共 ${count} 条`
          }"
          @change="(pagination: { current?: number }) => { novelPage = pagination.current ?? 1 }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'cover'">
              <img
                class="cover-thumb"
                :src="record.cover_url"
                :alt="record.title"
                loading="lazy"
              >
            </template>
            <template v-else-if="column.dataIndex === 'title'">
              <NuxtLink :to="novelLink(record)">{{ record.title }}</NuxtLink>
              <p v-if="record.original_title" class="cell-subtitle">
                {{ record.original_title }}
              </p>
            </template>
            <template v-else-if="column.dataIndex === 'release_status'">
              {{ domainSlug(NOVEL_RELEASE_STATUS, record.release_status ?? '')?.label ?? '-' }}
            </template>
            <template v-else-if="column.dataIndex === 'volume_count'">
              {{ record.statistics?.volume_count ?? 0 }}
            </template>
            <template v-else-if="column.dataIndex === 'status'">
              <a-tag :color="GALGAME_STATUS[record.status ?? 0]?.color">
                {{ domainLabel(NOVEL_STATUS, record.status) }}
              </a-tag>
            </template>
            <template v-else-if="column.dataIndex === 'updated_at'">
              {{ formatDate(record.updated_at) }}
            </template>
            <template v-else-if="column.dataIndex === 'actions'">
              <a-button
                type="link"
                size="small"
                :href="`/novels/${record.id}/edit`"
              >
                编辑
              </a-button>
              <a-button
                v-if="record.status !== 1"
                type="link"
                size="small"
                :loading="reviewingId === record.id"
                @click="approveNovel(record)"
              >
                通过
              </a-button>
              <a-button
                v-if="record.status === 0 || record.status === 3"
                type="link"
                size="small"
                danger
                @click="openRejectNovel(record)"
              >
                拒绝
              </a-button>
              <a-button
                v-if="record.status === 1"
                type="link"
                size="small"
                danger
                @click="openRejectNovel(record)"
              >
                下架
              </a-button>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="volumes" :tab="`卷册审核`">
        <div class="toolbar">
          <a-select
            v-model:value="volumeStatus"
            class="toolbar-select"
            placeholder="状态"
            allow-clear
            :options="
              NOVEL_STATUS.map((item) => ({
                value: item.value,
                label: item.label
              }))
            "
          />
        </div>

        <a-table
          :columns="volumeColumns"
          :data-source="volumeItems"
          :loading="volumeLoading"
          :row-key="(record: DtoAdminVolumeListItem) => record.id ?? 0"
          :pagination="{
            current: volumePage,
            pageSize: 20,
            total: volumeTotal,
            showSizeChanger: false,
            showTotal: (count: number) => `共 ${count} 条`
          }"
          @change="(pagination: { current?: number }) => { volumePage = pagination.current ?? 1 }"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'novel_title'">
              <NuxtLink :to="`/novels/${record.novel_id}`">
                {{ record.novel_title }}
              </NuxtLink>
            </template>
            <template v-else-if="column.dataIndex === 'volume_number'">
              {{ record.volume_number !== undefined && record.volume_number !== null ? `Vol.${record.volume_number}` : '-' }}
            </template>
            <template v-else-if="column.dataIndex === 'release_date'">
              {{ record.release_date ? record.release_date.slice(0, 10) : '-' }}
            </template>
            <template v-else-if="column.dataIndex === 'status'">
              <a-tag :color="GALGAME_STATUS[record.status ?? 0]?.color">
                {{ domainLabel(NOVEL_STATUS, record.status) }}
              </a-tag>
              <p v-if="record.status === 2 && record.reject_reason" class="cell-reject">
                {{ record.reject_reason }}
              </p>
            </template>
            <template v-else-if="column.dataIndex === 'updated_at'">
              {{ formatDate(record.updated_at) }}
            </template>
            <template v-else-if="column.dataIndex === 'actions'">
              <a-button
                v-if="record.status !== 1"
                type="link"
                size="small"
                :loading="reviewingVolumeId === record.id"
                @click="approveVolume(record)"
              >
                通过
              </a-button>
              <a-button
                v-if="record.status === 0"
                type="link"
                size="small"
                danger
                @click="openRejectVolume(record)"
              >
                拒绝
              </a-button>
            </template>
          </template>
        </a-table>
      </a-tab-pane>
    </a-tabs>

    <a-modal
      v-model:open="rejectOpen"
      :title="`拒绝${rejectTarget?.type === 'novel' ? '小说' : '卷册'}：${rejectTarget?.title ?? ''}`"
      ok-text="确认拒绝"
      cancel-text="取消"
      @ok="submitReject"
    >
      <a-form layout="vertical">
        <a-form-item label="拒绝原因（将通知提交者）">
          <a-textarea
            v-model:value="rejectReason"
            :rows="3"
            :maxlength="1000"
            placeholder="请填写拒绝原因"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </AppPageContainer>
</template>

<style scoped>
.toolbar {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
}

.toolbar-search {
  width: 260px;
}

.toolbar-select {
  width: 160px;
}

.cover-thumb {
  width: 40px;
  height: 56px;
  border-radius: 4px;
  object-fit: cover;
}

.cell-subtitle {
  margin: 2px 0 0;
  color: var(--color-default-500);
  font-size: 12px;
}

.cell-reject {
  margin: 2px 0 0;
  color: var(--color-danger);
  font-size: 12px;
}
</style>
