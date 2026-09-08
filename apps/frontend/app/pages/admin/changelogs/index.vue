<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { createSiteChangelogService } from '~/services/siteChangelog'
import type {
  ChangelogItem,
  ChangelogItemType,
  SiteChangelog
} from '~/types/siteChangelog'

useSeoMeta({ title: '更新日志管理 - Koyomi' })

const changelogService = createSiteChangelogService(useNuxtApp().$api)
const { has } = usePermissions()
const items = ref<SiteChangelog[]>([])
const total = ref(0)
const page = ref(1)
const limit = 20
const loading = ref(false)
const saving = ref(false)
const actionKey = ref('')
const modalOpen = ref(false)
const editing = ref<SiteChangelog | null>(null)

const typeLabels: Record<ChangelogItemType, string> = {
  new: '新增',
  improve: '改进',
  fix: '修复'
}

const typeOptions: { label: string; value: ChangelogItemType }[] = (
  Object.keys(typeLabels) as ChangelogItemType[]
).map((value) => ({ label: typeLabels[value], value }))

const emptyForm = () => ({
  version: '',
  title: '',
  publishedAt: '',
  items: [{ type: 'new' as ChangelogItemType, text: '' }] as ChangelogItem[]
})
const formState = reactive(emptyForm())

function typeLabel(value: ChangelogItemType): string {
  return typeLabels[value]
}

function itemCountSummary(record: SiteChangelog): string[] {
  const counts: Record<string, number> = {}
  for (const item of record.items ?? []) {
    counts[item.type] = (counts[item.type] ?? 0) + 1
  }
  return (Object.keys(typeLabels) as ChangelogItemType[])
    .filter((type) => counts[type])
    .map((type) => `${typeLabels[type]} ${counts[type]}`)
}

function formatDate(value?: string | null): string {
  return value
    ? new Date(value).toLocaleString('zh-CN', { hour12: false })
    : '-'
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

function addItem(): void {
  formState.items.push({ type: 'new', text: '' })
}

function removeItem(index: number): void {
  formState.items.splice(index, 1)
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = await changelogService.listAdmin({
      page: page.value,
      limit
    })
    items.value = data.items
    total.value = data.total
  } catch (error) {
    message.error(getApiErrorMessage(error, '更新日志列表加载失败'))
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())
watch(page, () => void load())

function openCreate(): void {
  if (!has('changelog:create')) return
  editing.value = null
  Object.assign(formState, emptyForm())
  formState.items = [{ type: 'new', text: '' }]
  modalOpen.value = true
}

function openEdit(item: SiteChangelog): void {
  if (!has('changelog:update')) return
  editing.value = item
  Object.assign(formState, {
    version: item.version,
    title: item.title,
    publishedAt: toDateInput(item.publishedAt),
    items: item.items.map((entry) => ({ ...entry }))
  })
  modalOpen.value = true
}

async function submit(): Promise<void> {
  if (
    (editing.value && !has('changelog:update')) ||
    (!editing.value && !has('changelog:create'))
  ) {
    return
  }
  const entries = formState.items
    .map((item) => ({ type: item.type, text: item.text.trim() }))
    .filter((item) => item.text !== '')
  if (!formState.version.trim() || !formState.title.trim()) {
    message.warning('请填写版本号和标题')
    return
  }
  if (entries.length === 0) {
    message.warning('请至少填写一条更新内容')
    return
  }

  saving.value = true
  try {
    const payload = {
      version: formState.version.trim(),
      title: formState.title.trim(),
      publishedAt: toIsoDate(formState.publishedAt),
      items: entries
    }
    if (editing.value) {
      await changelogService.update(editing.value.id, payload)
    } else {
      await changelogService.create(payload)
    }
    message.success(editing.value ? '更新日志已更新' : '更新日志已创建')
    modalOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '更新日志保存失败'))
  } finally {
    saving.value = false
  }
}

async function remove(item: SiteChangelog): Promise<void> {
  if (!has('changelog:delete')) return
  actionKey.value = `delete:${item.id}`
  try {
    await changelogService.remove(item.id)
    message.success('更新日志已删除')
    if (items.value.length === 1 && page.value > 1) page.value -= 1
    else await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '更新日志删除失败'))
  } finally {
    actionKey.value = ''
  }
}

const columns: TableColumnsType = [
  { title: '版本', dataIndex: 'version', width: 110 },
  { title: '标题', dataIndex: 'title', ellipsis: true },
  { title: '发布时间', dataIndex: 'publishedAt', width: 180 },
  { title: '条目数', key: 'itemCount', width: 90 },
  { title: '操作', key: 'actions', width: 180, fixed: 'right' }
]
</script>

<template>
  <div>
    <div class="table-toolbar">
      <KunHeader
        name="更新日志"
        description="管理站点版本更新日志，保存后立即对用户可见。"
        scale="h3"
      />
      <a-button
        v-if="has('changelog:create')"
        type="primary"
        @click="openCreate"
      >
        新建日志
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
      :scroll="{ x: 900 }"
      @change="(pagination: { current?: number }) => { page = pagination.current ?? 1 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'publishedAt'">
          {{ formatDate(record.publishedAt) }}
        </template>
        <template v-else-if="column.key === 'itemCount'">
          <a-tag v-for="entry in itemCountSummary(record)" :key="entry" class="item-count-tag">
            {{ entry }}
          </a-tag>
        </template>
        <template v-else-if="column.key === 'actions'">
          <div class="table-actions">
            <a-button
              v-if="has('changelog:update')"
              size="small"
              @click="openEdit(record)"
            >
              编辑
            </a-button>
            <a-popconfirm
              v-if="has('changelog:delete')"
              :title="`确定删除「${record.version}」的更新日志吗？`"
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
      :title="editing ? '编辑更新日志' : '新建更新日志'"
      :confirm-loading="saving"
      width="720px"
      ok-text="保存"
      cancel-text="取消"
      destroy-on-close
      :body-style="{ maxHeight: '72vh', overflowY: 'auto' }"
      @ok="submit"
    >
      <a-form layout="vertical">
        <div class="form-grid">
          <a-form-item label="版本号" required>
            <a-input
              v-model:value="formState.version"
              :maxlength="32"
              placeholder="例如 v0.5.0"
            />
          </a-form-item>
          <a-form-item label="发布时间">
            <a-input
              v-model:value="formState.publishedAt"
              type="datetime-local"
              placeholder="留空表示现在"
            />
          </a-form-item>
        </div>
        <a-form-item label="标题" required>
          <a-input v-model:value="formState.title" :maxlength="255" />
        </a-form-item>
        <a-form-item label="更新内容" required>
          <div class="item-editor">
            <div
              v-for="(item, index) in formState.items"
              :key="index"
              class="item-row"
            >
              <a-select
                v-model:value="item.type"
                :options="typeOptions"
                class="item-type"
              />
              <a-input
                v-model:value="item.text"
                :maxlength="255"
                placeholder="更新内容"
                class="item-text"
              />
              <a-button
                size="small"
                danger
                :disabled="formState.items.length <= 1"
                @click="removeItem(index)"
              >
                删除
              </a-button>
            </div>
            <a-button
              size="small"
              type="dashed"
              block
              :disabled="formState.items.length >= 50"
              @click="addItem"
            >
              添加条目
            </a-button>
          </div>
        </a-form-item>
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

.item-count-tag {
  margin-right: 4px;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.item-editor {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.item-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.item-type {
  width: 100px;
  flex-shrink: 0;
}

.item-text {
  flex: 1;
}

@media (max-width: 639px) {
  .table-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .form-grid {
    grid-template-columns: 1fr;
    gap: 0;
  }

  .item-row {
    align-items: stretch;
    flex-direction: column;
  }

  .item-type {
    width: 100%;
  }
}
</style>
