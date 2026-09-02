<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { message } from 'ant-design-vue'
import { createAuthService } from '~/services/auth'
import type { ImageAsset } from '~/types/image'
import { useUserStore } from '~/stores/user'

useSeoMeta({ title: '个人设置 - Koyomi' })

const router = useRouter()
const userStore = useUserStore()
const { user, isAuthenticated } = storeToRefs(userStore)
const authService = createAuthService(useNuxtApp().$api)
const saving = ref(false)

onMounted(() => {
  if (!userStore.getInitialized) {
    return
  }
  if (!isAuthenticated.value) {
    void router.replace('/login')
  }
})

watch(isAuthenticated, (authenticated) => {
  if (!authenticated && userStore.getInitialized) {
    void router.replace('/login')
  }
})

const avatarUser = computed(() =>
  user.value
    ? { id: user.value.id, name: user.value.username, avatar: user.value.avatar }
    : null
)

async function handleAvatarSuccess(asset: ImageAsset): Promise<void> {
  saving.value = true
  try {
    const updated = await authService.updateMe({
      avatar_asset_id: asset.id
    })
    userStore.setUser(updated)
    message.success('头像已更新')
  } catch (error) {
    message.error(getApiErrorMessage(error, '头像更新失败'))
  } finally {
    saving.value = false
  }
}

async function removeAvatar(): Promise<void> {
  saving.value = true
  try {
    const updated = await authService.updateMe({ avatar_asset_id: null })
    userStore.setUser(updated)
    message.success('头像已移除')
  } catch (error) {
    message.error(getApiErrorMessage(error, '头像移除失败'))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <AppPageContainer
    title="个人设置"
    description="管理账号资料与个性化配置。"
  >
    <KunCard padding="lg">
      <a-skeleton :loading="!userStore.getInitialized" :paragraph="{ rows: 3 }">
        <div v-if="user" class="profile-grid">
          <div class="profile-field">
            <span class="field-label">用户名</span>
            <span class="field-value">{{ user.username }}</span>
          </div>
          <div class="profile-field">
            <span class="field-label">邮箱</span>
            <span class="field-value">{{ user.email }}</span>
          </div>

          <a-divider />

          <div class="avatar-section">
            <div class="avatar-current">
              <KunAvatar :user="avatarUser" :is-navigation="false" />
              <div class="avatar-meta">
                <span class="field-label">头像</span>
                <span class="field-hint">
                  上传后自动替换，旧头像会被删除；支持 JPG、PNG、WebP、AVIF、GIF，最大 5MB
                </span>
              </div>
            </div>
            <a-spin :spinning="saving">
              <ImageUploader
                category="avatars"
                :width="'112px'"
                :height="'112px'"
                @success="handleAvatarSuccess"
                @remove="removeAvatar"
              />
            </a-spin>
          </div>
        </div>
      </a-skeleton>
    </KunCard>
  </AppPageContainer>
</template>

<style scoped>
.profile-grid {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.profile-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.field-label {
  color: var(--color-default-500);
  font-size: 13px;
}

.field-value {
  font-size: 15px;
}

.field-hint {
  color: var(--color-default-400);
  font-size: 12px;
}

.avatar-section {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.avatar-current {
  display: flex;
  align-items: center;
  gap: 14px;
}

.avatar-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
</style>
