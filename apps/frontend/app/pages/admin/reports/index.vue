<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  handleResourceReport,
  listResourceReports
} from '~/api/generated/admin/admin'
import type { DtoResourceReportData } from '~/api/generated/models'
import {
  REPORT_REASONS,
  REPORT_STATUS,
  RESOURCE_TYPES,
  domainLabel,
  formatDate
} from '~/constants/domain'

useSeoMeta({ title: '资源举报 - Koyomi' })

const { has } = usePermissions()
const activeStatus = ref(0)
const page = ref(1)
const limit = 20
const items = ref<DtoResourceReportData[]>([])
const total = ref(0)
const loading = ref(false)
const handling = ref<number | null>(null)

const TABS = REPORT_STATUS.map((item) => ({
  key: item.value,
  label: item.label
}))

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = unwrapApiData(
      await listResourceReports({
        status: activeStatus.value,
        page: page.value,
        limit
      })
    )
    items.value = data.items ?? []
    total.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '查询举报失败'))
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

async function handle(
  report: DtoResourceReportData,
  status: 1 | 2
): Promise<void> {
  if (!report.id || !has('resource_report:handle')) {
    return
  }

  handling.value = report.id
  try {
    await handleResourceReport(report.id, { status })
    message.success(
      status === 1 ? '已标记为解决' : '已驳回该举报'
    )
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '处理失败'))
  } finally {
    handling.value = null
  }
}

const columns = computed<TableColumnsType>(() => [
  { title: 'ID', dataIndex: 'id', width: 70 },
  {
    title: '资源',
    key: 'resource',
    ellipsis: true
  },
  {
    title: '原因',
    dataIndex: 'reason',
    width: 120
  },
  {
    title: '补充说明',
    dataIndex: 'description',
    ellipsis: true
  },
  {
    title: '提交时间',
    dataIndex: 'created_at',
    width: 170
  },
  {
    title: '处理时间',
    dataIndex: 'handled_at',
    width: 170
  },
  ...(has('resource_report:handle')
    ? [{ title: '操作', key: 'actions', width: 170 }]
    : [])
])
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
          <template v-if="record.resource">
            {{ record.resource.title }}
            <a-tag class="resource-type-tag">
              {{ domainLabel(RESOURCE_TYPES, record.resource.type) }}
            </a-tag>
          </template>
          <template v-else>#{{ record.resource_id }}</template>
        </template>

        <template v-else-if="column.dataIndex === 'reason'">
          <a-tag color="warning">
            {{ domainLabel(REPORT_REASONS, record.reason) }}
          </a-tag>
        </template>

        <template v-else-if="column.dataIndex === 'description'">
          {{ record.description || '-' }}
        </template>

        <template v-else-if="column.dataIndex === 'created_at'">
          {{ formatDate(record.created_at) }}
        </template>

        <template v-else-if="column.dataIndex === 'handled_at'">
          {{ record.handled_at ? formatDate(record.handled_at) : '-' }}
        </template>

        <template v-else-if="column.key === 'actions'">
          <div
            v-if="has('resource_report:handle') && record.status === 0"
            class="table-actions"
          >
            <a-button
              size="small"
              type="primary"
              :loading="handling === record.id"
              @click="handle(record, 1)"
            >
              标记解决
            </a-button>
            <a-button
              size="small"
              danger
              :loading="handling === record.id"
              @click="handle(record, 2)"
            >
              驳回
            </a-button>
          </div>
          <span v-else class="handled-text">
            {{ domainLabel(REPORT_STATUS, record.status) }}
          </span>
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

.handled-text {
  color: var(--color-default-400);
  font-size: 13px;
}
</style>
