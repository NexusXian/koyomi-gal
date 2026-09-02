<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { listAdminResources, reviewResource } from '~/api/generated/admin/admin'
import { deleteResource } from '~/api/generated/resources/resources'
import type { DtoResourceData } from '~/api/generated/models'
import {
  RESOURCE_STATUS,
  RESOURCE_TYPES,
  domainLabel,
  formatDate
} from '~/constants/domain'

useSeoMeta({ title: '资源审核 - Koyomi' })

const activeStatus = ref(0)
const userStore = useUserStore()
const { has } = usePermissions()
const page = ref(1)
const limit = 20
const items = ref<DtoResourceData[]>([])
const total = ref(0)
const loading = ref(false)
const reviewing = ref<number | null>(null)
const deletingId = ref<number | null>(null)
const editing = ref<DtoResourceData | null>(null)
const editOpen = ref(false)

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
  if (page.value !== 1) {
    page.value = 1
  } else {
    void load()
  }
})

watch(page, () => {
  void load()
})

async function review(
  resource: DtoResourceData,
  status: 1 | 2 | 3
): Promise<void> {
  if (!resource.id || !has('resource:review')) {
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

function isOwner(resource: DtoResourceData): boolean {
  return Boolean(
    userStore.getUser?.id && resource.uploader_id === userStore.getUser.id
  )
}

function canEdit(resource: DtoResourceData): boolean {
  return isOwner(resource) || has('resource:update')
}

function canDelete(resource: DtoResourceData): boolean {
  return isOwner(resource) || has('resource:delete')
}

function openEdit(resource: DtoResourceData): void {
  editing.value = resource
  editOpen.value = true
}

async function remove(resource: DtoResourceData): Promise<void> {
  if (!resource.id) {
    return
  }

  deletingId.value = resource.id
  try {
    await deleteResource(resource.id)
    message.success('资源已删除')
    if (items.value.length === 1 && page.value > 1) {
      page.value -= 1
    } else {
      await load()
    }
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除资源失败'))
  } finally {
    deletingId.value = null
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
    width: 280
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
              v-if="canEdit(record)"
              size="small"
              @click="openEdit(record)"
            >
              编辑
            </a-button>
            <a-button
              v-if="has('resource:review') && record.status !== 1"
              size="small"
              type="primary"
              :loading="reviewing === record.id"
              @click="review(record, 1)"
            >
              通过
            </a-button>
            <a-button
              v-if="has('resource:review') && record.status === 0"
              size="small"
              danger
              :loading="reviewing === record.id"
              @click="review(record, 2)"
            >
              拒绝
            </a-button>
            <a-button
              v-if="has('resource:review') && record.status === 1"
              size="small"
              :loading="reviewing === record.id"
              @click="review(record, 3)"
            >
              隐藏
            </a-button>
            <a-popconfirm
              v-if="canDelete(record)"
              :title="`确定删除资源「${record.title ?? record.id}」吗？`"
              ok-text="删除"
              cancel-text="取消"
              @confirm="remove(record)"
            >
              <a-button
                size="small"
                danger
                :loading="deletingId === record.id"
              >
                删除
              </a-button>
            </a-popconfirm>
          </div>
        </template>
      </template>
    </a-table>

    <ResourceEditModal
      v-model:open="editOpen"
      :resource="editing"
      @updated="load"
    />
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
