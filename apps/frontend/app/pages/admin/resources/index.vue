<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { listAdminResources, reviewResource } from '~/api/generated/admin/admin'
import type { DtoResourceData } from '~/api/generated/models'
import {
  RESOURCE_STATUS,
  RESOURCE_TYPES,
  domainLabel,
  formatDate
} from '~/constants/domain'

useSeoMeta({ title: '资源审核 - Koyomi' })

const activeStatus = ref(0)
const page = ref(1)
const limit = 20
const items = ref<DtoResourceData[]>([])
const total = ref(0)
const loading = ref(false)
const reviewing = ref<number | null>(null)

const TABS = RESOURCE_STATUS.map((item) => ({
  key: item.value,
  label: item.label
}))

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = unwrapApiData(
      await listAdminResources({
        status: activeStatus.value,
        page: page.value,
        limit
      })
    )
    items.value = data.items ?? []
    total.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '查询资源失败'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void load()
})

watch(activeStatus, () => {
  page.value = 1
  void load()
})

watch(page, () => {
  void load()
})

async function review(
  resource: DtoResourceData,
  status: 1 | 2 | 3
): Promise<void> {
  if (!resource.id) {
    return
  }

  reviewing.value = resource.id
  try {
    await reviewResource(resource.id, { status })
    message.success(
      status === 1 ? '资源已发布' : status === 2 ? '资源已拒绝' : '资源已隐藏'
    )
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '审核失败'))
  } finally {
    reviewing.value = null
  }
}

const columns: TableColumnsType = [
  { title: 'ID', dataIndex: 'id', width: 70 },
  {
    title: '资源',
    key: 'resource',
    ellipsis: true
  },
  {
    title: 'Galgame',
    dataIndex: 'galgame_id',
    width: 100
  },
  {
    title: '上传者',
    dataIndex: 'uploader_id',
    width: 90
  },
  {
    title: '链接',
    key: 'links',
    width: 70
  },
  {
    title: '状态',
    dataIndex: 'status',
    width: 100
  },
  {
    title: '提交时间',
    dataIndex: 'created_at',
    width: 170
  },
  {
    title: '操作',
    key: 'actions',
    width: 170
  }
]
</script>

<template>
  <div>
    <a-tabs v-model:active-key="activeStatus">
      <a-tab-pane
        v-for="tab in TABS"
        :key="tab.key"
        :tab="tab.label"
      />
    </a-tabs>

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
      @change="
        (pagination: { current?: number }) => {
          page = pagination.current ?? 1
        }
      "
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'resource'">
          {{ record.title }}
          <a-tag class="resource-type-tag">
            {{ domainLabel(RESOURCE_TYPES, record.type) }}
          </a-tag>
        </template>

        <template v-else-if="column.dataIndex === 'galgame_id'">
          <NuxtLink :to="`/galgames/${record.galgame_id}`">
            #{{ record.galgame_id }}
          </NuxtLink>
        </template>

        <template v-else-if="column.dataIndex === 'uploader_id'">
          {{ record.uploader_id ? `#${record.uploader_id}` : '-' }}
        </template>

        <template v-else-if="column.key === 'links'">
          {{ record.links?.length ?? 0 }}
        </template>

        <template v-else-if="column.dataIndex === 'status'">
          <a-tag :color="RESOURCE_STATUS[record.status]?.color">
            {{ domainLabel(RESOURCE_STATUS, record.status) }}
          </a-tag>
        </template>

        <template v-else-if="column.dataIndex === 'created_at'">
          {{ formatDate(record.created_at) }}
        </template>

        <template v-else-if="column.key === 'actions'">
          <div class="table-actions">
            <a-button
              v-if="record.status !== 1"
              size="small"
              type="primary"
              :loading="reviewing === record.id"
              @click="review(record, 1)"
            >
              通过
            </a-button>
            <a-button
              v-if="record.status === 0"
              size="small"
              danger
              :loading="reviewing === record.id"
              @click="review(record, 2)"
            >
              拒绝
            </a-button>
            <a-button
              v-if="record.status === 1"
              size="small"
              :loading="reviewing === record.id"
              @click="review(record, 3)"
            >
              隐藏
            </a-button>
          </div>
        </template>
      </template>
    </a-table>
  </div>
</template>

<style scoped>
.table-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.resource-type-tag {
  margin-left: 6px;
}
</style>
