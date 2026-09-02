<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  createTag,
  listTags,
  updateTag
} from '~/api/generated/tags/tags'
import type { DtoTagResponse } from '~/api/generated/models'

useSeoMeta({ title: 'Tag 管理 - Koyomi' })

const { has } = usePermissions()
const items = ref<DtoTagResponse[]>([])
const loading = ref(false)
const modalOpen = ref(false)
const editing = ref<DtoTagResponse | null>(null)
const saving = ref(false)

const formState = reactive({
  name: '',
  slug: '',
  description: ''
})

async function load(): Promise<void> {
  loading.value = true
  try {
    items.value = unwrapApiData(await listTags()) ?? []
  } catch (error) {
    message.error(getApiErrorMessage(error, '加载 Tag 失败'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void load()
})

function openCreate(): void {
  if (!has('galgame:create')) {
    return
  }

  editing.value = null
  Object.assign(formState, { name: '', slug: '', description: '' })
  modalOpen.value = true
}

function openEdit(tag: DtoTagResponse): void {
  if (!has('galgame:update')) {
    return
  }

  editing.value = tag
  Object.assign(formState, {
    name: tag.name ?? '',
    slug: tag.slug ?? '',
    description: tag.description ?? ''
  })
  modalOpen.value = true
}

async function submit(): Promise<void> {
  if (
    (editing.value && !has('galgame:update')) ||
    (!editing.value && !has('galgame:create'))
  ) {
    return
  }

  if (!formState.name.trim() || !formState.slug.trim()) {
    message.warning('请填写名称和 Slug')
    return
  }

  saving.value = true
  try {
    const payload = {
      name: formState.name.trim(),
      slug: formState.slug.trim(),
      description: formState.description.trim() || undefined
    }
    if (editing.value?.id) {
      await updateTag(editing.value.id, payload)
    } else {
      await createTag(payload)
    }
    message.success(editing.value ? 'Tag 已更新' : 'Tag 已创建')
    modalOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '保存失败'))
  } finally {
    saving.value = false
  }
}

const columns = computed<TableColumnsType>(() => [
  { title: 'ID', dataIndex: 'id', width: 70 },
  { title: '名称', dataIndex: 'name', width: 160 },
  { title: 'Slug', dataIndex: 'slug', width: 160 },
  { title: '描述', dataIndex: 'description', ellipsis: true },
  ...(has('galgame:update')
    ? [{ title: '操作', key: 'actions', width: 100 }]
    : [])
])
</script>

<template>
  <div>
    <div v-if="has('galgame:create')" class="table-toolbar">
      <a-button type="primary" @click="openCreate">
        新建 Tag
      </a-button>
    </div>

    <a-table
      :columns="columns"
      :data-source="items"
      :loading="loading"
      :pagination="false"
      row-key="id"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'description'">
          {{ record.description || '-' }}
        </template>

        <template v-else-if="column.key === 'actions'">
          <div class="table-actions">
            <a-button
              v-if="has('galgame:update')"
              size="small"
              @click="openEdit(record)"
            >
              编辑
            </a-button>
          </div>
        </template>
      </template>
    </a-table>

    <a-modal
      v-model:open="modalOpen"
      :title="editing ? '编辑 Tag' : '新建 Tag'"
      :confirm-loading="saving"
      ok-text="保存"
      cancel-text="取消"
      @ok="submit"
    >
      <a-form layout="vertical">
        <a-form-item label="名称" required>
          <a-input v-model:value="formState.name" :maxlength="100" />
        </a-form-item>
        <a-form-item label="Slug" required>
          <a-input v-model:value="formState.slug" :maxlength="100" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="formState.description" :rows="3" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.table-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 14px;
}

.table-actions {
  display: flex;
  gap: 6px;
}
</style>
