<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  createPermission,
  deletePermission,
  listPermissions,
  updatePermission
} from '~/api/generated/permissions/permissions'
import type { DtoPermissionResponse } from '~/api/generated/models'

const emit = defineEmits<{
  changed: []
}>()

const { load: loadPermissions, has } = usePermissions()
const canList = computed(() => has('permission:list'))
const canCreate = computed(() => has('permission:create'))
const canUpdate = computed(() => has('permission:update'))
const canDelete = computed(() => has('permission:delete'))
const items = ref<DtoPermissionResponse[]>([])
const loading = ref(false)
const modalOpen = ref(false)
const editing = ref<DtoPermissionResponse | null>(null)
const saving = ref(false)

const formState = reactive({ code: '', name: '', description: '' })

async function load(): Promise<void> {
  if (!canList.value) return

  loading.value = true
  try {
    items.value = unwrapApiData(await listPermissions()) ?? []
  } catch (error) {
    message.error(getApiErrorMessage(error, '加载权限失败'))
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadPermissions()
  await load()
})

function openCreate(): void {
  if (!canCreate.value) return

  editing.value = null
  Object.assign(formState, { code: '', name: '', description: '' })
  modalOpen.value = true
}

function openEdit(permission: DtoPermissionResponse): void {
  if (!canUpdate.value) return

  editing.value = permission
  Object.assign(formState, {
    code: permission.code ?? '',
    name: permission.name ?? '',
    description: permission.description ?? ''
  })
  modalOpen.value = true
}

async function submit(): Promise<void> {
  if ((editing.value && !canUpdate.value) || (!editing.value && !canCreate.value)) {
    return
  }
  if (!formState.code.trim() || !formState.name.trim()) {
    message.warning('请填写权限代码和名称')
    return
  }

  saving.value = true
  try {
    if (editing.value?.id) {
      await updatePermission(editing.value.id, {
        name: formState.name.trim(),
        description: formState.description.trim() || undefined
      })
    } else {
      await createPermission({
        code: formState.code.trim(),
        name: formState.name.trim(),
        description: formState.description.trim() || undefined
      })
    }
    message.success(editing.value ? '权限已更新' : '权限已创建')
    modalOpen.value = false
    emit('changed')
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '保存失败'))
  } finally {
    saving.value = false
  }
}

async function remove(permission: DtoPermissionResponse): Promise<void> {
  if (!permission.id || !canDelete.value) {
    return
  }

  try {
    await deletePermission(permission.id)
    message.success(`权限「${permission.name}」已删除`)
    emit('changed')
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除失败'))
  }
}

const hasActions = computed(() => canUpdate.value || canDelete.value)
const columns = computed<TableColumnsType>(() => {
  const result: TableColumnsType = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '代码', dataIndex: 'code', width: 170 },
    { title: '名称', dataIndex: 'name', width: 170 },
    { title: '描述', dataIndex: 'description', ellipsis: true }
  ]
  if (hasActions.value) {
    result.push({ title: '操作', key: 'actions', width: 150 })
  }
  return result
})

defineExpose({ reload: load })
</script>

<template>
  <div>
    <div class="table-toolbar">
      <a-button v-if="canCreate" type="primary" @click="openCreate">新建权限</a-button>
    </div>

    <a-table
      v-if="canList"
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
            <a-button v-if="canUpdate" size="small" @click="openEdit(record)">编辑</a-button>
            <a-popconfirm
              v-if="canDelete"
              :title="`确定删除权限「${record.name}」吗？`"
              ok-text="删除"
              cancel-text="取消"
              @confirm="remove(record)"
            >
              <a-button size="small" danger>删除</a-button>
            </a-popconfirm>
          </div>
        </template>
      </template>
    </a-table>

    <a-modal
      v-model:open="modalOpen"
      :title="editing ? '编辑权限' : '新建权限'"
      :confirm-loading="saving"
      ok-text="保存"
      cancel-text="取消"
      @ok="submit"
    >
      <a-form layout="vertical">
        <a-form-item label="权限代码" required>
          <a-input v-model:value="formState.code" :maxlength="64" :disabled="Boolean(editing)" />
        </a-form-item>
        <a-form-item label="名称" required>
          <a-input v-model:value="formState.name" :maxlength="64" />
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
