<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { storeToRefs } from 'pinia'
import { listRoles } from '~/api/generated/roles/roles'
import {
  createAdminUser,
  deleteAdminUser,
  getAdminUser,
  listAdminUsers,
  listUserRoles,
  updateAdminUser,
  updateUserRoles
} from '~/api/generated/users/users'
import type {
  DtoAdminUserData,
  DtoCreateAdminUserRequest,
  DtoRoleResponse,
  DtoUpdateAdminUserRequest
} from '~/api/generated/models'

useSeoMeta({ title: '用户管理 - Koyomi' })

const { load: loadPermissions, has } = usePermissions()
const { user: currentUser } = storeToRefs(useUserStore())
const canList = computed(() => has('user:list'))
const canRead = computed(() => has('user:read'))
const canCreate = computed(() => has('user:create'))
const canUpdate = computed(() => has('user:update'))
const canDelete = computed(() => has('user:delete'))
const canManageRoles = computed(() => has('role:list') && has('role:assign'))
const hasManualActions = computed(() =>
  canRead.value || canUpdate.value || canDelete.value || canManageRoles.value
)

const items = ref<DtoAdminUserData[]>([])
const total = ref(0)
const page = ref(1)
const limit = 20
const keywordInput = ref('')
const keyword = ref('')
const listLoading = ref(false)

const detailOpen = ref(false)
const detailLoading = ref(false)
const detailUser = ref<DtoAdminUserData | null>(null)

const userModalOpen = ref(false)
const editingUserId = ref<number | null>(null)
const editingFromSnapshot = ref(false)
const userSaving = ref(false)
const deletingId = ref<number | null>(null)
const emptyUserForm = () => ({
  username: '',
  email: '',
  password: '',
  is_banned: false as boolean | undefined
})
const userForm = reactive(emptyUserForm())
const isEditingSelf = computed(() => isSelf(editingUserId.value))

const manualUserId = ref<number>()

const roleDrawerOpen = ref(false)
const roleUserId = ref<number | null>(null)
const roleUserLabel = ref('')
const allRoles = ref<DtoRoleResponse[]>([])
const selectedRoleIds = ref<number[]>([])
const roleOptionsLoaded = ref(false)
const userRolesLoaded = ref(false)
const roleLoading = ref(false)
const roleSaving = ref(false)
let roleLoadRequest = 0
const roleDataReady = computed(() =>
  roleOptionsLoaded.value && userRolesLoaded.value
)

const hasTableActions = computed(() =>
  canRead.value || canUpdate.value || canDelete.value || canManageRoles.value
)
const userColumns = computed<TableColumnsType>(() => {
  const columns: TableColumnsType = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '用户名', dataIndex: 'username', width: 160 },
    { title: '邮箱', dataIndex: 'email', ellipsis: true },
    { title: '状态', key: 'status', width: 90 },
    { title: '创建时间', dataIndex: 'created_at', width: 180 }
  ]
  if (hasTableActions.value) {
    columns.push({ title: '操作', key: 'actions', width: 280 })
  }
  return columns
})

onMounted(async () => {
  await loadPermissions()
  if (canList.value) {
    await loadUsers()
  }
})

async function loadUsers(): Promise<void> {
  if (!canList.value) return

  listLoading.value = true
  try {
    const data = unwrapApiData(await listAdminUsers({
      page: page.value,
      limit,
      keyword: keyword.value || undefined
    }))
    items.value = data.items ?? []
    total.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '用户列表加载失败'))
  } finally {
    listLoading.value = false
  }
}

function search(): void {
  keyword.value = keywordInput.value.trim()
  page.value = 1
  void loadUsers()
}

function resetSearch(): void {
  keywordInput.value = ''
  keyword.value = ''
  page.value = 1
  void loadUsers()
}

function changePage(pagination: { current?: number }): void {
  page.value = pagination.current ?? 1
  void loadUsers()
}

async function openDetail(id?: number): Promise<void> {
  if (!id || !canRead.value) return

  detailOpen.value = true
  detailUser.value = null
  detailLoading.value = true
  try {
    detailUser.value = unwrapApiData(await getAdminUser(id))
  } catch (error) {
    message.error(getApiErrorMessage(error, '用户详情加载失败'))
  } finally {
    detailLoading.value = false
  }
}

function openCreate(): void {
  if (!canCreate.value) return

  editingUserId.value = null
  editingFromSnapshot.value = false
  Object.assign(userForm, emptyUserForm())
  userModalOpen.value = true
}

function openEdit(user: DtoAdminUserData): void {
  if (!user.id || !canUpdate.value) return

  editingUserId.value = user.id
  editingFromSnapshot.value = true
  Object.assign(userForm, {
    username: user.username ?? '',
    email: user.email ?? '',
    password: '',
    is_banned: user.is_banned ?? false
  })
  userModalOpen.value = true
}

function openManualEdit(): void {
  if (!canUpdate.value) return

  if (!manualUserId.value) {
    message.warning('请输入用户 ID')
    return
  }

  editingUserId.value = manualUserId.value
  editingFromSnapshot.value = false
  Object.assign(userForm, {
    username: '',
    email: '',
    password: '',
    is_banned: undefined
  })
  userModalOpen.value = true
}

async function submitUser(): Promise<void> {
  const username = userForm.username.trim()
  const email = userForm.email.trim()
  const password = userForm.password

  if (!editingUserId.value && (!username || !email || !password)) {
    message.warning('请填写用户名、邮箱和密码')
    return
  }
  if (editingFromSnapshot.value && (!username || !email)) {
    message.warning('请填写用户名和邮箱')
    return
  }
  if (password && password.length < 8) {
    message.warning('密码至少需要 8 个字符')
    return
  }

  userSaving.value = true
  try {
    if (editingUserId.value) {
      const payload: DtoUpdateAdminUserRequest = {}
      if (username) payload.username = username
      if (email) payload.email = email
      if (password) payload.password = password
      if (!isEditingSelf.value && userForm.is_banned !== undefined) {
        payload.is_banned = userForm.is_banned
      }
      if (!Object.keys(payload).length) {
        message.warning('请至少填写一个要更新的字段')
        return
      }
      await updateAdminUser(editingUserId.value, payload)
      message.success('用户已更新')
    } else {
      const payload: DtoCreateAdminUserRequest = {
        username,
        email,
        password,
        is_banned: userForm.is_banned ?? false
      }
      await createAdminUser(payload)
      message.success('用户已创建')
    }
    userModalOpen.value = false
    if (canList.value) await loadUsers()
  } catch (error) {
    message.error(getApiErrorMessage(error, '用户保存失败'))
  } finally {
    userSaving.value = false
  }
}

async function removeUser(user: DtoAdminUserData): Promise<void> {
  if (!user.id || isSelf(user.id) || !canDelete.value) return

  deletingId.value = user.id
  try {
    await deleteAdminUser(user.id)
    message.success(`用户「${user.username || `#${user.id}`}」已删除`)
    if (canList.value) {
      if (items.value.length === 1 && page.value > 1) page.value -= 1
      await loadUsers()
    }
  } catch (error) {
    message.error(getApiErrorMessage(error, '用户删除失败'))
  } finally {
    deletingId.value = null
  }
}

function requireManualId(): number | null {
  if (!manualUserId.value) {
    message.warning('请输入用户 ID')
    return null
  }
  return manualUserId.value
}

function openManualDetail(): void {
  const id = requireManualId()
  if (id) void openDetail(id)
}

function removeManualUser(): void {
  const id = requireManualId()
  if (id && !isSelf(id)) void removeUser({ id })
}

function openManualRoles(): void {
  const id = requireManualId()
  if (id && !isSelf(id)) void openRoles({ id })
}

async function openRoles(user: DtoAdminUserData): Promise<void> {
  if (!user.id || isSelf(user.id) || !canManageRoles.value) return

  const request = ++roleLoadRequest
  roleUserId.value = user.id
  roleUserLabel.value = user.username || `用户 #${user.id}`
  allRoles.value = []
  selectedRoleIds.value = []
  roleOptionsLoaded.value = false
  userRolesLoaded.value = false
  roleDrawerOpen.value = true
  roleLoading.value = true

  const [roleResult, userRoleResult] = await Promise.allSettled([
    listRoles(),
    listUserRoles(user.id)
  ])
  if (request !== roleLoadRequest) return

  if (roleResult.status === 'fulfilled') {
    try {
      allRoles.value = unwrapApiData(roleResult.value) ?? []
      roleOptionsLoaded.value = true
    } catch (error) {
      message.error(getApiErrorMessage(error, '角色列表加载失败'))
    }
  } else {
    message.error(getApiErrorMessage(roleResult.reason, '角色列表加载失败'))
  }

  if (userRoleResult.status === 'fulfilled') {
    try {
      const roles = unwrapApiData(userRoleResult.value) ?? []
      selectedRoleIds.value = roles
        .map((role) => role.id)
        .filter((id): id is number => id !== undefined)
      userRolesLoaded.value = true
    } catch (error) {
      message.error(getApiErrorMessage(error, '用户角色加载失败'))
    }
  } else {
    message.error(getApiErrorMessage(userRoleResult.reason, '用户角色加载失败'))
  }

  roleLoading.value = false
}

async function saveRoles(): Promise<void> {
  if (
    !roleUserId.value ||
    isSelf(roleUserId.value) ||
    !roleDataReady.value ||
    !canManageRoles.value
  ) return

  roleSaving.value = true
  try {
    await updateUserRoles(roleUserId.value, { role_ids: selectedRoleIds.value })
    message.success('用户角色已更新')
    roleDrawerOpen.value = false
  } catch (error) {
    message.error(getApiErrorMessage(error, '角色保存失败'))
  } finally {
    roleSaving.value = false
  }
}

function formatDate(value?: string): string {
  return value
    ? new Date(value).toLocaleString('zh-CN', { hour12: false })
    : '-'
}

function isSelf(id?: number | null): boolean {
  return Boolean(id && currentUser.value?.id === id)
}
</script>

<template>
  <div>
    <div class="page-toolbar">
      <KunHeader
        name="用户管理"
        description="查询用户、维护账户状态并分配角色。"
        scale="h3"
      />
      <a-button v-if="canCreate" type="primary" @click="openCreate">
        新建用户
      </a-button>
    </div>

    <template v-if="canList">
      <KunCard padding="md" class-name="search-card">
        <div class="search-row">
          <a-input-search
            v-model:value="keywordInput"
            allow-clear
            placeholder="搜索用户名、邮箱或精确用户 ID"
            :loading="listLoading"
            @search="search"
          />
          <a-button :disabled="listLoading" @click="resetSearch">重置</a-button>
        </div>
      </KunCard>

      <a-table
        :columns="userColumns"
        :data-source="items"
        :loading="listLoading"
        :pagination="{
          current: page,
          pageSize: limit,
          total,
          showSizeChanger: false,
          showTotal: (count: number) => `共 ${count} 位用户`
        }"
        row-key="id"
        :scroll="{ x: 980 }"
        @change="changePage"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="record.is_banned ? 'error' : 'success'">
              {{ record.is_banned ? '已封禁' : '正常' }}
            </a-tag>
          </template>
          <template v-else-if="column.dataIndex === 'created_at'">
            {{ formatDate(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'actions'">
            <div class="table-actions">
              <a-button v-if="canRead" size="small" @click="openDetail(record.id)">
                详情
              </a-button>
              <a-button v-if="canUpdate" size="small" @click="openEdit(record)">
                编辑
              </a-button>
              <a-button v-if="canManageRoles && !isSelf(record.id)" size="small" @click="openRoles(record)">
                角色
              </a-button>
              <a-popconfirm
                v-if="canDelete && !isSelf(record.id)"
                :title="`确定删除用户「${record.username || `#${record.id}`}」吗？`"
                ok-text="删除"
                cancel-text="取消"
                @confirm="removeUser(record)"
              >
                <a-button size="small" danger :loading="deletingId === record.id">
                  删除
                </a-button>
              </a-popconfirm>
            </div>
          </template>
        </template>
      </a-table>
    </template>

    <KunCard v-else-if="hasManualActions" padding="md">
      <KunHeader
        name="按用户 ID 管理"
        description="当前权限不能浏览用户列表，可直接操作已知用户 ID。"
        scale="h3"
        class="manual-heading"
      />
      <div class="manual-row">
        <a-input-number
          v-model:value="manualUserId"
          class="id-input"
          :min="1"
          :precision="0"
          placeholder="用户 ID"
        />
        <a-button v-if="canRead" @click="openManualDetail">查看详情</a-button>
        <a-button v-if="canUpdate" @click="openManualEdit">编辑</a-button>
        <a-button v-if="canManageRoles && !isSelf(manualUserId)" type="primary" @click="openManualRoles">
          配置角色
        </a-button>
        <a-popconfirm
          v-if="canDelete && manualUserId && !isSelf(manualUserId)"
          :title="`确定删除用户 #${manualUserId} 吗？`"
          ok-text="删除"
          cancel-text="取消"
          @confirm="removeManualUser"
        >
          <a-button danger :loading="deletingId === manualUserId">删除</a-button>
        </a-popconfirm>
      </div>
    </KunCard>

    <a-modal
      v-model:open="detailOpen"
      title="用户详情"
      :footer="null"
      destroy-on-close
    >
      <a-spin :spinning="detailLoading">
        <a-descriptions v-if="detailUser" :column="1" bordered size="small">
          <a-descriptions-item label="ID">{{ detailUser.id }}</a-descriptions-item>
          <a-descriptions-item label="用户名">{{ detailUser.username }}</a-descriptions-item>
          <a-descriptions-item label="邮箱">{{ detailUser.email }}</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="detailUser.is_banned ? 'error' : 'success'">
              {{ detailUser.is_banned ? '已封禁' : '正常' }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ formatDate(detailUser.created_at) }}</a-descriptions-item>
          <a-descriptions-item label="更新时间">{{ formatDate(detailUser.updated_at) }}</a-descriptions-item>
        </a-descriptions>
      </a-spin>
    </a-modal>

    <a-modal
      v-model:open="userModalOpen"
      :title="editingUserId ? `编辑用户 #${editingUserId}` : '新建用户'"
      :confirm-loading="userSaving"
      ok-text="保存"
      cancel-text="取消"
      destroy-on-close
      @ok="submitUser"
    >
      <a-form layout="vertical">
        <a-form-item label="用户名" :required="!editingUserId || editingFromSnapshot">
          <a-input
            v-model:value="userForm.username"
            :maxlength="50"
            :placeholder="editingFromSnapshot ? '' : '留空则保持不变'"
          />
        </a-form-item>
        <a-form-item label="邮箱" :required="!editingUserId || editingFromSnapshot">
          <a-input
            v-model:value="userForm.email"
            type="email"
            :maxlength="254"
            :placeholder="editingFromSnapshot ? '' : '留空则保持不变'"
          />
        </a-form-item>
        <a-form-item :label="editingUserId ? '新密码' : '密码'" :required="!editingUserId">
          <a-input-password
            v-model:value="userForm.password"
            :maxlength="255"
            :placeholder="editingUserId ? '留空则保持不变' : '至少 8 个字符'"
          />
        </a-form-item>
        <a-form-item v-if="!isEditingSelf" label="账户状态">
          <a-select v-if="editingUserId && !editingFromSnapshot" v-model:value="userForm.is_banned">
            <a-select-option :value="undefined">保持不变</a-select-option>
            <a-select-option :value="false">正常</a-select-option>
            <a-select-option :value="true">已封禁</a-select-option>
          </a-select>
          <a-switch v-else v-model:checked="userForm.is_banned" checked-children="封禁" un-checked-children="正常" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-drawer
      v-model:open="roleDrawerOpen"
      :title="`分配角色 - ${roleUserLabel}`"
      width="440"
    >
      <a-spin :spinning="roleLoading">
        <a-alert
          v-if="!roleLoading && !roleDataReady"
          type="error"
          show-icon
          message="角色数据未完整加载，已禁用保存以避免覆盖现有角色。"
          class="role-alert"
        />
        <a-checkbox-group
          v-model:value="selectedRoleIds"
          class="role-group"
          :disabled="!roleDataReady"
        >
          <div class="role-list">
            <a-checkbox
              v-for="role in allRoles"
              :key="role.id"
              :value="role.id"
              class="role-item"
            >
              <span class="role-name">{{ role.name }}</span>
              <span class="role-code">{{ role.code }}</span>
            </a-checkbox>
          </div>
        </a-checkbox-group>
      </a-spin>

      <template #footer>
        <div class="drawer-footer">
          <a-button @click="roleDrawerOpen = false">取消</a-button>
          <a-button
            type="primary"
            :loading="roleSaving"
            :disabled="roleLoading || !roleDataReady"
            @click="saveRoles"
          >
            保存角色
          </a-button>
        </div>
      </template>
    </a-drawer>
  </div>
</template>

<style scoped>
.page-toolbar,
.search-row,
.manual-row,
.table-actions,
.drawer-footer {
  display: flex;
  align-items: center;
  gap: 8px;
}

.page-toolbar {
  justify-content: space-between;
  margin-bottom: 16px;
}

.search-card {
  margin-bottom: 16px;
}

.search-row :deep(.ant-input-search) {
  max-width: 520px;
}

.manual-heading {
  margin-bottom: 14px;
}

.manual-row {
  flex-wrap: wrap;
}

.id-input {
  width: 200px;
}

.table-actions {
  flex-wrap: wrap;
}

.role-alert {
  margin-bottom: 14px;
}

.role-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.role-item {
  display: flex;
  align-items: center;
  margin-inline-start: 0;
  padding: 9px 10px;
  border: 1px solid var(--color-default-200);
  border-radius: var(--radius-kun-md);
}

.role-name {
  font-weight: 600;
}

.role-code {
  margin-left: 8px;
  color: var(--color-default-400);
  font-size: 12px;
}

.drawer-footer {
  justify-content: flex-end;
}

@media (max-width: 639px) {
  .page-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .search-row {
    align-items: stretch;
    flex-direction: column;
  }

  .id-input {
    width: 100%;
  }
}
</style>
