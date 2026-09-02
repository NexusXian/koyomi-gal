<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  createRole,
  deleteRole,
  getRolePermissions,
  listRoles,
  updateRole,
  updateRolePermissions
} from '~/api/generated/roles/roles'
import { listPermissions } from '~/api/generated/permissions/permissions'
import type { DtoRoleResponse } from '~/api/generated/models'

useSeoMeta({ title: '角色管理 - Koyomi' })

const { load: loadPermissions, has } = usePermissions()
const canListRoles = computed(() => has('role:list'))
const canCreateRole = computed(() => has('role:create'))
const canUpdateRole = computed(() => has('role:update'))
const canDeleteRole = computed(() => has('role:delete'))
const canAssignPermissions = computed(() =>
  has('role:list') && has('permission:list') && has('permission:assign')
)
const roles = ref<DtoRoleResponse[]>([])
const permissions = ref<{ id?: number; name?: string; code?: string }[]>([])
const roleLoading = ref(false)
const permissionsLoading = ref(false)
const permissionsLoaded = ref(false)
let permissionOptionsRequest = 0

const roleModalOpen = ref(false)
const editingRole = ref<DtoRoleResponse | null>(null)
const roleSaving = ref(false)
const roleForm = reactive({ code: '', name: '', description: '' })

const permDrawerOpen = ref(false)
const permDrawerRole = ref<DtoRoleResponse | null>(null)
const selectedPermissionIds = ref<number[]>([])
const permDrawerLoading = ref(false)
const rolePermissionsLoaded = ref(false)
const permSaving = ref(false)
let rolePermissionRequest = 0

async function loadRoles(): Promise<void> {
  if (!canListRoles.value) return

  roleLoading.value = true
  try {
    roles.value = unwrapApiData(await listRoles()) ?? []
  } catch (error) {
    message.error(getApiErrorMessage(error, '角色列表加载失败'))
  } finally {
    roleLoading.value = false
  }
}

async function loadPermissionOptions(): Promise<void> {
  if (!canAssignPermissions.value) return

  const request = ++permissionOptionsRequest
  permissionsLoading.value = true
  permissionsLoaded.value = false
  try {
    const data = unwrapApiData(await listPermissions()) ?? []
    if (request !== permissionOptionsRequest) return
    permissions.value = data
    permissionsLoaded.value = true
  } catch (error) {
    if (request !== permissionOptionsRequest) return
    permissions.value = []
    message.error(getApiErrorMessage(error, '权限列表加载失败'))
  } finally {
    if (request === permissionOptionsRequest) {
      permissionsLoading.value = false
    }
  }
}

onMounted(async () => {
  await loadPermissions()
  await Promise.all([loadRoles(), loadPermissionOptions()])
})

function openRoleCreate(): void {
  if (!canCreateRole.value) return

  editingRole.value = null
  Object.assign(roleForm, { code: '', name: '', description: '' })
  roleModalOpen.value = true
}

function openRoleEdit(role: DtoRoleResponse): void {
  if (!canUpdateRole.value) return

  editingRole.value = role
  Object.assign(roleForm, {
    code: role.code ?? '',
    name: role.name ?? '',
    description: role.description ?? ''
  })
  roleModalOpen.value = true
}

async function submitRole(): Promise<void> {
  if (
    (editingRole.value && !canUpdateRole.value) ||
    (!editingRole.value && !canCreateRole.value)
  ) {
    return
  }
  if (!roleForm.code.trim() || !roleForm.name.trim()) {
    message.warning('请填写角色代码和名称')
    return
  }

  roleSaving.value = true
  try {
    if (editingRole.value?.id) {
      await updateRole(editingRole.value.id, {
        name: roleForm.name.trim(),
        description: roleForm.description.trim() || undefined
      })
    } else {
      await createRole({
        code: roleForm.code.trim(),
        name: roleForm.name.trim(),
        description: roleForm.description.trim() || undefined
      })
    }
    message.success(editingRole.value ? '角色已更新' : '角色已创建')
    roleModalOpen.value = false
    await loadRoles()
  } catch (error) {
    message.error(getApiErrorMessage(error, '保存失败'))
  } finally {
    roleSaving.value = false
  }
}

async function removeRole(role: DtoRoleResponse): Promise<void> {
  if (!role.id || !canDeleteRole.value) {
    return
  }

  try {
    await deleteRole(role.id)
    message.success(`角色「${role.name}」已删除`)
    await loadRoles()
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除失败'))
  }
}

async function openPermissions(role: DtoRoleResponse): Promise<void> {
  if (!role.id || !canAssignPermissions.value) return

  const request = ++rolePermissionRequest
  permDrawerRole.value = role
  permDrawerOpen.value = true
  selectedPermissionIds.value = []
  rolePermissionsLoaded.value = false
  permDrawerLoading.value = true
  try {
    const [response] = await Promise.all([
      getRolePermissions(role.id),
      permissionsLoaded.value ? Promise.resolve() : loadPermissionOptions()
    ])
    if (request !== rolePermissionRequest) return
    const data = unwrapApiData(response)
    selectedPermissionIds.value = (data ?? [])
      .map((item) => item.id)
      .filter((id): id is number => id !== undefined)
    rolePermissionsLoaded.value = true
  } catch (error) {
    if (request !== rolePermissionRequest) return
    message.error(getApiErrorMessage(error, '加载角色权限失败'))
  } finally {
    if (request === rolePermissionRequest) {
      permDrawerLoading.value = false
    }
  }
}

async function submitPermissions(): Promise<void> {
  if (
    !permDrawerRole.value?.id ||
    !canAssignPermissions.value ||
    !permissionsLoaded.value ||
    !rolePermissionsLoaded.value
  ) {
    return
  }

  permSaving.value = true
  try {
    await updateRolePermissions(permDrawerRole.value.id, {
      permission_ids: selectedPermissionIds.value
    })
    message.success('角色权限已更新')
    permDrawerOpen.value = false
  } catch (error) {
    message.error(getApiErrorMessage(error, '保存权限失败'))
  } finally {
    permSaving.value = false
  }
}

const hasRoleActions = computed(() =>
  canUpdateRole.value || canDeleteRole.value || canAssignPermissions.value
)
const roleColumns = computed<TableColumnsType>(() => {
  const columns: TableColumnsType = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '代码', dataIndex: 'code', width: 140 },
    { title: '名称', dataIndex: 'name', width: 140 },
    { title: '描述', dataIndex: 'description', ellipsis: true }
  ]
  if (hasRoleActions.value) {
    columns.push({ title: '操作', key: 'actions', width: 250 })
  }
  return columns
})
</script>

<template>
  <div>
    <KunHeader
      name="角色"
      description="管理系统角色及其权限。"
      scale="h3"
      class="section-heading"
    />
    <div class="table-toolbar">
      <a-button v-if="canCreateRole" type="primary" @click="openRoleCreate">新建角色</a-button>
    </div>
    <a-table
      v-if="canListRoles"
      :columns="roleColumns"
      :data-source="roles"
      :loading="roleLoading"
      :pagination="false"
      row-key="id"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'description'">
          {{ record.description || '-' }}
        </template>

        <template v-else-if="column.key === 'actions'">
          <div class="table-actions">
            <a-button v-if="canUpdateRole" size="small" @click="openRoleEdit(record)">
              编辑
            </a-button>
            <a-button v-if="canAssignPermissions" size="small" @click="openPermissions(record)">
              权限
            </a-button>
            <a-popconfirm
              v-if="canDeleteRole"
              :title="`确定删除角色「${record.name}」吗？`"
              ok-text="删除"
              cancel-text="取消"
              @confirm="removeRole(record)"
            >
              <a-button size="small" danger>删除</a-button>
            </a-popconfirm>
          </div>
        </template>
      </template>
    </a-table>

    <a-modal
      v-model:open="roleModalOpen"
      :title="editingRole ? '编辑角色' : '新建角色'"
      :confirm-loading="roleSaving"
      ok-text="保存"
      cancel-text="取消"
      @ok="submitRole"
    >
      <a-form layout="vertical">
        <a-form-item label="角色代码" required>
          <a-input v-model:value="roleForm.code" :maxlength="64" :disabled="Boolean(editingRole)" />
        </a-form-item>
        <a-form-item label="名称" required>
          <a-input v-model:value="roleForm.name" :maxlength="64" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="roleForm.description" :rows="3" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-drawer
      v-model:open="permDrawerOpen"
      :title="`角色权限 - ${permDrawerRole?.name ?? ''}`"
      width="420"
    >
      <a-spin :spinning="permDrawerLoading || permissionsLoading">
        <a-alert
          v-if="!permissionsLoading && (!permissionsLoaded || !rolePermissionsLoaded)"
          type="error"
          show-icon
          message="权限数据未完整加载，已禁用保存以避免覆盖现有权限。"
          class="permission-alert"
        />
        <a-checkbox-group
          v-model:value="selectedPermissionIds"
          class="permission-group"
          :disabled="!permissionsLoaded || !rolePermissionsLoaded"
        >
          <div class="permission-list">
            <a-checkbox
              v-for="permission in permissions"
              :key="permission.id"
              :value="permission.id"
              class="permission-item"
            >
              <span class="permission-name">{{ permission.name }}</span>
              <span class="permission-code">{{ permission.code }}</span>
            </a-checkbox>
          </div>
        </a-checkbox-group>
      </a-spin>

      <template #footer>
        <div class="drawer-footer">
          <a-button
            type="primary"
            :loading="permSaving"
            :disabled="permDrawerLoading || permissionsLoading || !permissionsLoaded || !rolePermissionsLoaded"
            @click="submitPermissions"
          >
            保存
          </a-button>
        </div>
      </template>
    </a-drawer>
  </div>
</template>

<style scoped>
.section-heading {
  margin: 10px 0 12px;
}

.table-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 14px;
}

.table-actions {
  display: flex;
  gap: 6px;
}

.permission-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.permission-alert {
  margin-bottom: 14px;
}

.permission-item {
  display: flex;
  align-items: center;
  margin-inline-start: 0;
  padding: 8px 10px;
  border: 1px solid var(--color-default-200);
  border-radius: var(--radius-kun-md);
}

.permission-name {
  font-weight: 600;
}

.permission-code {
  margin-left: 8px;
  color: var(--color-default-400);
  font-size: 12px;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
}
</style>
