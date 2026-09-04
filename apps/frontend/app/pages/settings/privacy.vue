<script setup lang="ts">
import { message } from 'ant-design-vue'
import { getMyPrivacy, updateMyPrivacy } from '~/api/generated/me/me'
import type {
  DtoPrivacySettingsData,
  DtoUpdatePrivacyRequest,
  DtoUpdatePrivacyRequestProfileVisibility
} from '~/api/generated/models'

useSeoMeta({ title: '隐私设置 - Koyomi' })

const router = useRouter()
const userStore = useUserStore()
const { user, initialized, isAuthenticated } = storeToRefs(userStore)
const { mode: sensitiveCoverMode, setMode: setSensitiveCoverMode } = useSensitiveCover()
const sensitiveCoverSaving = ref(false)
const loading = ref(false)
const saving = ref(false)
const loaded = ref(false)
const settings = reactive<Required<DtoUpdatePrivacyRequest>>({
  profile_visibility: 'public',
  show_activity: true,
  show_posts: true,
  show_comments: true,
  show_ratings: true,
  show_favorites: false,
  show_location: false,
  show_birthday: false
})

function applySettings(data: DtoPrivacySettingsData): void {
  settings.profile_visibility = (data.profile_visibility as DtoUpdatePrivacyRequestProfileVisibility) || 'public'
  settings.show_activity = data.show_activity ?? true
  settings.show_posts = data.show_posts ?? true
  settings.show_comments = data.show_comments ?? true
  settings.show_ratings = data.show_ratings ?? true
  settings.show_favorites = data.show_favorites ?? false
  settings.show_location = data.show_location ?? false
  settings.show_birthday = data.show_birthday ?? false
}

async function loadPrivacy(): Promise<void> {
  if (loaded.value || !isAuthenticated.value) return
  loading.value = true
  try {
    applySettings(unwrapApiData(await getMyPrivacy(), '隐私设置加载失败'))
    loaded.value = true
  } catch (error) {
    message.error(getApiErrorMessage(error, '隐私设置加载失败'))
  } finally {
    loading.value = false
  }
}

async function savePrivacy(): Promise<void> {
  if (!loaded.value || saving.value) return
  saving.value = true
  try {
    const data = unwrapApiData(await updateMyPrivacy({ ...settings }), '隐私设置更新失败')
    applySettings(data)
    message.success('隐私设置已立即生效')
  } catch (error) {
    message.error(getApiErrorMessage(error, '隐私设置更新失败'))
    loaded.value = false
    await loadPrivacy()
  } finally {
    saving.value = false
  }
}

watch(
  [initialized, isAuthenticated],
  ([ready, authenticated]) => {
    if (!ready) return
    if (!authenticated) {
      void router.replace('/login')
      return
    }
    void loadPrivacy()
  },
  { immediate: true }
)

async function changeSensitiveCoverMode(event: { target: { value: 'blur' | 'show' } }): Promise<void> {
  const next = event.target.value
  if (sensitiveCoverSaving.value || next === sensitiveCoverMode.value) return
  sensitiveCoverSaving.value = true
  const ok = await setSensitiveCoverMode(next)
  if (ok) {
    message.success(next === 'show' ? '敏感封面将始终显示' : '敏感封面已恢复默认模糊')
  } else {
    message.error('敏感封面设置保存失败')
  }
  sensitiveCoverSaving.value = false
}
</script>

<template>
  <AppPageContainer title="隐私设置" description="控制谁可以查看你的个人空间和社区记录。">
    <nav class="settings-nav" aria-label="设置导航">
      <NuxtLink to="/settings/profile">个人资料</NuxtLink>
      <NuxtLink class="active" to="/settings/privacy">隐私设置</NuxtLink>
      <NuxtLink v-if="user?.username" :to="`/user/${user.username}`">查看个人空间</NuxtLink>
    </nav>

    <KunCard padding="lg">
      <a-spin :spinning="loading">
        <div v-if="loaded" class="privacy-form">
          <div class="privacy-row visibility-row">
            <div><strong>个人空间可见范围</strong><p>决定谁可以访问你的个人资料页。</p></div>
            <a-select v-model:value="settings.profile_visibility" :disabled="saving" style="width: 160px" @change="savePrivacy">
              <a-select-option value="public">所有人</a-select-option>
              <a-select-option value="registered">仅登录用户</a-select-option>
              <a-select-option value="private">仅自己</a-select-option>
            </a-select>
          </div>

          <div v-for="item in [
            { key: 'show_activity', title: '公开动态', detail: '展示最近的社区活动' },
            { key: 'show_posts', title: '公开帖子', detail: '允许他人查看你的帖子列表' },
            { key: 'show_comments', title: '公开评论', detail: '允许他人查看你的评论列表' },
            { key: 'show_ratings', title: '公开评分', detail: '允许他人查看你的 Galgame 评分' },
            { key: 'show_favorites', title: '公开收藏', detail: '允许他人查看你的 Galgame 收藏' },
            { key: 'show_location', title: '公开所在地', detail: '在个人资料中展示所在地' },
            { key: 'show_birthday', title: '公开生日', detail: '在个人资料中展示生日' }
          ]" :key="item.key" class="privacy-row">
            <div><strong>{{ item.title }}</strong><p>{{ item.detail }}</p></div>
            <a-switch
              v-model:checked="settings[item.key as keyof typeof settings]"
              :disabled="saving"
              @change="savePrivacy"
            />
          </div>
          <span v-if="saving" class="saving-hint">正在保存...</span>
        </div>
      </a-spin>
    </KunCard>

    <KunCard padding="lg" class-name="sensitive-card">
      <KunHeader
        name="敏感内容"
        description="控制被标记为敏感的 Galgame 封面如何展示"
        scale="h3"
      />
      <div class="sensitive-form">
        <div class="privacy-row">
          <div>
            <strong>敏感封面显示方式</strong>
            <p>选择「始终显示」后，敏感封面将不再模糊。</p>
          </div>
          <a-radio-group
            :value="sensitiveCoverMode"
            :disabled="sensitiveCoverSaving"
            @change="changeSensitiveCoverMode"
          >
            <a-radio value="blur">默认模糊</a-radio>
            <a-radio value="show">始终显示</a-radio>
          </a-radio-group>
        </div>
        <p class="sensitive-hint">
          未登录时敏感封面始终默认模糊；点击单张模糊封面可临时查看，刷新后恢复。
        </p>
        <span v-if="sensitiveCoverSaving" class="saving-hint">正在保存...</span>
      </div>
    </KunCard>
  </AppPageContainer>
</template>

<style scoped>
.settings-nav { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 16px; }
.settings-nav a { padding: 8px 14px; border-radius: var(--radius-kun-md); color: var(--color-default-500); }
.settings-nav a:hover, .settings-nav a.active { background: color-mix(in srgb, var(--color-primary) 12%, transparent); color: var(--color-primary); }
.privacy-form { display: flex; flex-direction: column; }
.privacy-row { display: flex; align-items: center; justify-content: space-between; gap: 24px; padding: 17px 0; border-bottom: 1px solid var(--color-default-200); }
.privacy-row:last-of-type { border-bottom: 0; }
.privacy-row strong { font-size: 15px; }
.privacy-row p { margin: 4px 0 0; color: var(--color-default-500); font-size: 13px; }
.saving-hint { margin-top: 8px; color: var(--color-default-400); font-size: 12px; text-align: right; }
.sensitive-card { margin-top: 18px; }
.sensitive-form { display: flex; flex-direction: column; }
.sensitive-hint { margin: 12px 0 0; color: var(--color-default-400); font-size: 12px; }
@media (max-width: 480px) {
  .visibility-row { align-items: flex-start; flex-direction: column; }
  .visibility-row :deep(.ant-select) { width: 100% !important; }
}
</style>
