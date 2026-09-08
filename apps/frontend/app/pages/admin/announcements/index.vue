<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { createAnnouncementService } from '~/services/announcement'
import type {
  Announcement,
  AnnouncementDisplayMode,
  AnnouncementPayload,
  AnnouncementTarget,
  AnnouncementType
} from '~/types/announcement'

useSeoMeta({ title: '公告管理 - Koyomi' })

const announcementService = createAnnouncementService(useNuxtApp().$api)
const { has } = usePermissions()
const items = ref<Announcement[]>([])
const total = ref(0)
const page = ref(1)
const limit = 20
const loading = ref(false)
const saving = ref(false)
const actionKey = ref('')
const modalOpen = ref(false)
const editing = ref<Announcement | null>(null)

const typeOptions: { label: string; value: AnnouncementType }[] = [
  { label: '普通', value: 'normal' },
  { label: '更新', value: 'update' },
  { label: '维护', value: 'maintenance' },
  { label: '警告', value: 'warning' },
  { label: '活动', value: 'event' },
  { label: '系统', value: 'system' }
]
const displayModeOptions: {
  label: string
  value: AnnouncementDisplayMode
}[] = [
  { label: '普通展示', value: 'normal' },
  { label: '横幅', value: 'banner' },
  { label: '弹窗', value: 'modal' },
  { label: '启动弹窗', value: 'startup_modal' }
]
const targetOptions: { label: string; value: AnnouncementTarget }[] = [
  { label: '全部平台', value: 'all' },
  { label: '网页端', value: 'web' },
  { label: 'Android', value: 'android' },
  { label: 'iOS', value: 'ios' },
  { label: '桌面端', value: 'desktop' }
]

const emptyForm = () => ({
  title: '',
  content: '',
  type: 'normal' as AnnouncementType,
  displayMode: 'startup_modal' as AnnouncementDisplayMode,
  target: 'all' as AnnouncementTarget,
  priority: 0,
  dismissible: true,
  startsAt: '',
  endsAt: ''
})
const formState = reactive(emptyForm())

function optionLabel<T extends string>(
  options: { label: string; value: T }[],
  value: T
): string {
  return options.find((option) => option.value === value)?.label ?? value
}

function toDateInput(value?: string | null): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000)
  return local.toISOString().slice(0, 16)
}

function toIsoDate(value: string): string | null {
  if (!value) return null
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? null : date.toISOString()
}

function formatDate(value?: string | null): string {
  return value
    ? new Date(value).toLocaleString('zh-CN', { hour12: false })
    : '-'
}

function payload(): AnnouncementPayload {
  return {
    title: formState.title.trim(),
    content: formState.content.trim(),
    type: formState.type,
    displayMode: formState.displayMode,
    target: formState.target,
    priority: formState.priority,
    dismissible: formState.dismissible,
    published: editing.value?.published ?? false,
    startsAt: toIsoDate(formState.startsAt),
    endsAt: toIsoDate(formState.endsAt)
  }
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = await announcementService.listAdmin({ page: page.value, limit })
    items.value = data.items
    total.value = data.total
  } catch (error) {
    message.error(getApiErrorMessage(error, '公告列表加载失败'))
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())
watch(page, () => void load())

function openCreate(): void {
  if (!has('announcement:create')) return
  editing.value = null
  Object.assign(formState, emptyForm())
  modalOpen.value = true
}

async function openEdit(item: Announcement): Promise<void> {
  if (!has('announcement:update')) return
  actionKey.value = `edit:${item.id}`
  try {
    const detail = await announcementService.getAdmin(item.id)
    editing.value = detail
    Object.assign(formState, {
      title: detail.title,
      content: detail.content,
      type: detail.type,
      displayMode: detail.displayMode,
      target: detail.target,
      priority: detail.priority,
      dismissible: detail.dismissible,
      startsAt: toDateInput(detail.startsAt),
      endsAt: toDateInput(detail.endsAt)
    })
    modalOpen.value = true
  } catch (error) {
    message.error(getApiErrorMessage(error, '公告加载失败'))
  } finally {
    actionKey.value = ''
  }
}

async function submit(): Promise<void> {
  if (
    (editing.value && !has('announcement:update')) ||
    (!editing.value && !has('announcement:create'))
  ) {
    return
  }
  if (!formState.title.trim() || !formState.content.trim()) {
    message.warning('请填写公告标题和内容')
    return
  }
  if (
    formState.startsAt &&
    formState.endsAt &&
    new Date(formState.endsAt) < new Date(formState.startsAt)
  ) {
    message.warning('结束时间不能早于开始时间')
    return
  }

  saving.value = true
  try {
    if (editing.value) {
      await announcementService.update(editing.value.id, payload())
    } else {
      await announcementService.create(payload())
    }
    message.success(editing.value ? '公告已更新' : '公告已创建')
    modalOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '公告保存失败'))
  } finally {
    saving.value = false
  }
}

async function changeStatus(
  item: Announcement,
  action: 'publish' | 'withdraw'
): Promise<void> {
  if (!has('announcement:publish')) return
  actionKey.value = `${action}:${item.id}`
  try {
    await announcementService[action](item.id)
    message.success(action === 'publish' ? '公告已发布' : '公告已撤回')
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '公告状态更新失败'))
  } finally {
    actionKey.value = ''
  }
}

async function remove(item: Announcement): Promise<void> {
  if (!has('announcement:delete')) return
  actionKey.value = `delete:${item.id}`
  try {
    await announcementService.remove(item.id)
    message.success('公告已删除')
    if (items.value.length === 1 && page.value > 1) page.value -= 1
    else await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '公告删除失败'))
  } finally {
    actionKey.value = ''
  }
}

const columns: TableColumnsType = [
  { title: '标题', dataIndex: 'title', width: 220, ellipsis: true },
  { title: '类型', dataIndex: 'type', width: 90 },
  { title: '目标', dataIndex: 'target', width: 100 },
  { title: '展示方式', dataIndex: 'displayMode', width: 110 },
  { title: '开始时间', dataIndex: 'startsAt', width: 180 },
  { title: '结束时间', dataIndex: 'endsAt', width: 180 },
  { title: '状态', dataIndex: 'status', width: 90 },
  { title: '优先级', dataIndex: 'priority', width: 86 },
  { title: '操作', key: 'actions', width: 260, fixed: 'right' }
]
</script>

<template>
  <div>
    <div class="table-toolbar">
      <KunHeader
        name="公告管理"
        description="管理各平台公告、展示方式和生效时间。"
        scale="h3"
      />
      <a-button
        v-if="has('announcement:create')"
        type="primary"
        @click="openCreate"
      >
        新建公告
      </a-button>
    </div>

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
      :scroll="{ x: 1420 }"
      @change="(pagination: { current?: number }) => { page = pagination.current ?? 1 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'type'">
          <a-tag>{{ optionLabel(typeOptions, record.type) }}</a-tag>
        </template>
        <template v-else-if="column.dataIndex === 'target'">
          {{ optionLabel(targetOptions, record.target) }}
        </template>
        <template v-else-if="column.dataIndex === 'displayMode'">
          {{ optionLabel(displayModeOptions, record.displayMode) }}
        </template>
        <template v-else-if="column.dataIndex === 'startsAt'">
          {{ formatDate(record.startsAt) }}
        </template>
        <template v-else-if="column.dataIndex === 'endsAt'">
          {{ formatDate(record.endsAt) }}
        </template>
        <template v-else-if="column.dataIndex === 'status'">
          <a-tag :color="record.published ? 'success' : 'default'">
            {{ record.published ? '已发布' : '草稿' }}
          </a-tag>
        </template>
        <template v-else-if="column.key === 'actions'">
          <div class="table-actions">
            <a-button
              v-if="has('announcement:update')"
              size="small"
              :loading="actionKey === `edit:${record.id}`"
              @click="openEdit(record)"
            >
              编辑
            </a-button>
            <a-button
              v-if="!record.published && has('announcement:publish')"
              size="small"
              type="primary"
              :loading="actionKey === `publish:${record.id}`"
              @click="changeStatus(record, 'publish')"
            >
              发布
            </a-button>
            <a-button
              v-if="record.published && has('announcement:publish')"
              size="small"
              :loading="actionKey === `withdraw:${record.id}`"
              @click="changeStatus(record, 'withdraw')"
            >
              撤回
            </a-button>
            <a-popconfirm
              v-if="has('announcement:delete')"
              :title="`确定删除「${record.title}」吗？`"
              ok-text="删除"
              cancel-text="取消"
              @confirm="remove(record)"
            >
              <a-button
                size="small"
                danger
                :loading="actionKey === `delete:${record.id}`"
              >
                删除
              </a-button>
            </a-popconfirm>
          </div>
        </template>
      </template>
    </a-table>

    <a-modal
      v-model:open="modalOpen"
      :title="editing ? '编辑公告' : '新建公告'"
      :confirm-loading="saving"
      width="900px"
      ok-text="保存"
      cancel-text="取消"
      destroy-on-close
      :body-style="{ maxHeight: '72vh', overflowY: 'auto' }"
      @ok="submit"
    >
      <a-form layout="vertical">
        <a-form-item label="标题" required>
          <a-input v-model:value="formState.title" :maxlength="255" />
        </a-form-item>
        <a-form-item label="内容" required>
          <PostEditor
            v-model="formState.content"
            mode="markdown"
            :disabled="saving"
          />
        </a-form-item>
        <div class="form-grid form-grid-three">
          <a-form-item label="类型" required>
            <a-select v-model:value="formState.type" :options="typeOptions" />
          </a-form-item>
          <a-form-item label="展示方式" required>
            <a-select
              v-model:value="formState.displayMode"
              :options="displayModeOptions"
            />
          </a-form-item>
          <a-form-item label="目标平台" required>
            <a-select v-model:value="formState.target" :options="targetOptions" />
          </a-form-item>
        </div>
        <div class="form-grid">
          <a-form-item label="开始时间">
            <a-input v-model:value="formState.startsAt" type="datetime-local" />
          </a-form-item>
          <a-form-item label="结束时间">
            <a-input v-model:value="formState.endsAt" type="datetime-local" />
          </a-form-item>
        </div>
        <div class="form-grid">
          <a-form-item label="优先级">
            <a-input-number
              v-model:value="formState.priority"
              :precision="0"
              class="full-width"
            />
          </a-form-item>
          <a-form-item label="允许关闭">
            <a-switch v-model:checked="formState.dismissible" />
          </a-form-item>
        </div>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.table-toolbar {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.table-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.form-grid-three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.full-width {
  width: 100%;
}

@media (max-width: 639px) {
  .table-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .form-grid,
  .form-grid-three {
    grid-template-columns: 1fr;
    gap: 0;
  }
}
</style>
