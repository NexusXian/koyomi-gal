<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  createDeveloper,
  listDevelopers,
  updateDeveloper
} from '~/api/generated/developers/developers'
import type { DtoDeveloperResponse } from '~/api/generated/models'

useSeoMeta({ title: '开发商管理 - Koyomi' })

const { has } = usePermissions()
const items = ref<DtoDeveloperResponse[]>([])
const loading = ref(false)
const modalOpen = ref(false)
const editing = ref<DtoDeveloperResponse | null>(null)
const saving = ref(false)

const formState = reactive({
  name: '',
  slug: '',
  original_name: '',
  logo_url: '',
  website: '',
  description: ''
})

async function load(): Promise<void> {
  loading.value = true
  try {
    items.value = unwrapApiData(await listDevelopers()) ?? []
  } catch (error) {
    message.error(getApiErrorMessage(error, '加载开发商失败'))
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
  Object.assign(formState, {
    name: '',
    slug: '',
    original_name: '',
    logo_url: '',
    website: '',
    description: ''
  })
  modalOpen.value = true
}

function openEdit(developer: DtoDeveloperResponse): void {
  if (!has('galgame:update')) {
    return
  }

  editing.value = developer
  Object.assign(formState, {
    name: developer.name ?? '',
    slug: developer.slug ?? '',
    original_name: developer.original_name ?? '',
    logo_url: developer.logo_url ?? '',
    website: developer.website ?? '',
    description: developer.description ?? ''
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
      original_name: formState.original_name.trim() || undefined,
      logo_url: formState.logo_url.trim() || undefined,
      website: formState.website.trim() || undefined,
      description: formState.description.trim() || undefined
    }
    if (editing.value?.id) {
      await updateDeveloper(editing.value.id, payload)
    } else {
      await createDeveloper(payload)
    }
    message.success(editing.value ? '开发商已更新' : '开发商已创建')
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
  { title: 'Slug', dataIndex: 'slug', width: 150, ellipsis: true },
  { title: '原名', dataIndex: 'original_name', width: 160, ellipsis: true },
  { title: '官网', dataIndex: 'website', ellipsis: true },
  { title: '简介', dataIndex: 'description', ellipsis: true },
  ...(has('galgame:update')
    ? [{ title: '操作', key: 'actions', width: 100 }]
    : [])
])
</script>

<template>
  <div>
    <div v-if="has('galgame:create')" class="table-toolbar">
      <a-button type="primary" @click="openCreate">
        新建开发商
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
        <template v-if="column.dataIndex === 'website'">
          <a
            v-if="record.website"
            :href="record.website"
            target="_blank"
            rel="noopener noreferrer nofollow"
          >
            {{ record.website }}
          </a>
          <template v-else>-</template>
        </template>

        <template v-else-if="column.dataIndex === 'description'">
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
      :title="editing ? '编辑开发商' : '新建开发商'"
      :confirm-loading="saving"
      ok-text="保存"
      cancel-text="取消"
      @ok="submit"
    >
      <a-form layout="vertical">
        <a-form-item label="名称" required>
          <a-input v-model:value="formState.name" :maxlength="255" />
        </a-form-item>
        <a-form-item label="Slug" required>
          <a-input v-model:value="formState.slug" :maxlength="255" />
        </a-form-item>
        <a-form-item label="原名">
          <a-input v-model:value="formState.original_name" :maxlength="255" />
        </a-form-item>
        <a-form-item label="Logo 地址">
          <a-input v-model:value="formState.logo_url" placeholder="https://" />
        </a-form-item>
        <a-form-item label="官网">
          <a-input v-model:value="formState.website" placeholder="https://" />
        </a-form-item>
        <a-form-item label="简介">
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
