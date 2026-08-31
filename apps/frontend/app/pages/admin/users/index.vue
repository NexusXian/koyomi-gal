<script setup lang="ts">
import { message } from 'ant-design-vue'
import { listRoles } from '~/api/generated/roles/roles'
import {
  listUserRoles,
  updateUserRoles
} from '~/api/generated/users/users'
import type { DtoRoleResponse } from '~/api/generated/models'

useSeoMeta({ title: '用户角色 - Koyomi' })

const allRoles = ref<DtoRoleResponse[]>([])
const userId = ref<number>()
const currentRoleIds = ref<number[]>([])
const loading = ref(false)
const saving = ref(false)
const loadedUserId = ref<number | null>(null)

const { user } = storeToRefs(useUserStore())

onMounted(async () => {
  try {
    allRoles.value = unwrapApiData(await listRoles()) ?? []
  } catch (error) {
    message.error(getApiErrorMessage(error, '加载角色列表失败'))
  }

  if (user.value?.id) {
    userId.value = user.value.id
    await load()
  }
})

async function load(): Promise<void> {
  if (!userId.value) {
    message.warning('请输入用户 ID')
    return
  }

  loading.value = true
  try {
    const roles = unwrapApiData(await listUserRoles(userId.value))
    currentRoleIds.value = (roles ?? [])
      .map((role) => role.id)
      .filter((id): id is number => Boolean(id))
    loadedUserId.value = userId.value
  } catch (error) {
    message.error(getApiErrorMessage(error, '查询用户角色失败'))
    loadedUserId.value = null
  } finally {
    loading.value = false
  }
}

async function save(): Promise<void> {
  if (!loadedUserId.value) {
    return
  }

  saving.value = true
  try {
    await updateUserRoles(loadedUserId.value, {
      role_ids: currentRoleIds.value
    })
    message.success('用户角色已更新')
  } catch (error) {
    message.error(getApiErrorMessage(error, '保存失败'))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div>
    <KunCard padding="md" class-name="lookup-card">
      <div class="lookup-row">
        <a-input-number
          v-model:value="userId"
          class="lookup-input"
          :min="1"
          :precision="0"
          placeholder="输入用户 ID"
        />
        <a-button type="primary" :loading="loading" @click="load">
          查询角色
        </a-button>
      </div>
      <p class="lookup-hint">
        输入要调整角色的用户 ID，勾选目标角色后保存。
      </p>
    </KunCard>

    <KunCard v-if="loadedUserId" padding="lg">
      <KunHeader
        :name="`用户 #${loadedUserId} 的角色`"
        :description="`共 ${currentRoleIds.length} 个角色`"
        scale="h3"
        class="section-heading"
      />

      <a-spin :spinning="loading">
        <a-checkbox-group
          v-model:value="currentRoleIds"
          class="role-group"
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

      <div class="save-row">
        <a-button
          type="primary"
          :loading="saving"
          @click="save"
        >
          保存角色
        </a-button>
      </div>
    </KunCard>
  </div>
</template>

<style scoped>
.lookup-card {
  margin-bottom: 16px;
}

.lookup-row {
  display: flex;
  gap: 10px;
}

.lookup-input {
  width: 220px;
}

.lookup-hint {
  margin: 10px 0 0;
  color: var(--color-default-500);
  font-size: 13px;
}

.section-heading {
  margin-bottom: 14px;
}

.role-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.role-item {
  display: flex;
  align-items: center;
  margin-inline-start: 0;
  padding: 8px 10px;
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

.save-row {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
