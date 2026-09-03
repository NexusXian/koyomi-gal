<script setup lang="ts">
import { message } from 'ant-design-vue'
import { getMyProfile, updateMyProfile } from '~/api/generated/me/me'
import type {
  DtoPublicUserProfile,
  DtoUpdateProfileRequest,
  DtoUpdateProfileRequestGender
} from '~/api/generated/models'
import type { ImageAsset } from '~/types/image'

type NullableAssetPayload = Omit<DtoUpdateProfileRequest, 'avatar_asset_id' | 'banner_asset_id'> & {
  avatar_asset_id?: number | null
  banner_asset_id?: number | null
}

useSeoMeta({ title: '个人资料设置 - Koyomi' })

const router = useRouter()
const userStore = useUserStore()
const { user, initialized, isAuthenticated } = storeToRefs(userStore)
const loading = ref(false)
const saving = ref(false)
const loaded = ref(false)
const profile = ref<DtoPublicUserProfile | null>(null)
const avatarAsset = ref<ImageAsset | null>(null)
const bannerAsset = ref<ImageAsset | null>(null)
const avatarTouched = ref(false)
const bannerTouched = ref(false)
const avatarAssetId = ref<number | null>(null)
const bannerAssetId = ref<number | null>(null)
const form = reactive({
  display_name: '',
  bio: '',
  gender: 'undisclosed' as DtoUpdateProfileRequestGender,
  location: '',
  birthday: '',
  website_url: ''
})

async function loadProfile(): Promise<void> {
  if (loaded.value || !isAuthenticated.value) return
  loading.value = true
  try {
    const data = unwrapApiData(await getMyProfile(), '个人资料加载失败')
    profile.value = data
    form.display_name = data.display_name ?? ''
    form.bio = data.bio ?? ''
    form.gender = (data.gender as DtoUpdateProfileRequestGender) || 'undisclosed'
    form.location = data.location ?? ''
    form.birthday = data.birthday?.slice(0, 10) ?? ''
    form.website_url = data.website_url ?? ''
    loaded.value = true
  } catch (error) {
    message.error(getApiErrorMessage(error, '个人资料加载失败'))
  } finally {
    loading.value = false
  }
}

function setAvatar(asset: ImageAsset): void {
  avatarAsset.value = asset
  avatarAssetId.value = asset.id
  avatarTouched.value = true
}

function clearAvatar(): void {
  avatarAsset.value = null
  avatarAssetId.value = null
  avatarTouched.value = true
}

function setBanner(asset: ImageAsset): void {
  bannerAsset.value = asset
  bannerAssetId.value = asset.id
  bannerTouched.value = true
}

function clearBanner(): void {
  bannerAsset.value = null
  bannerAssetId.value = null
  bannerTouched.value = true
}

async function save(): Promise<void> {
  if (!isAuthenticated.value) return
  saving.value = true
  try {
    const payload: NullableAssetPayload = {
      display_name: form.display_name.trim(),
      bio: form.bio.trim(),
      gender: form.gender,
      location: form.location.trim(),
      birthday: form.birthday,
      website_url: form.website_url.trim()
    }
    if (avatarTouched.value) payload.avatar_asset_id = avatarAssetId.value
    if (bannerTouched.value) payload.banner_asset_id = bannerAssetId.value

    const data = unwrapApiData(
      await updateMyProfile(payload as DtoUpdateProfileRequest),
      '个人资料保存失败'
    )
    profile.value = data
    avatarTouched.value = false
    bannerTouched.value = false
    if (user.value) {
      userStore.setUser({ ...user.value, avatar: data.avatar_url ?? '' })
    }
    message.success('个人资料已保存')
  } catch (error) {
    message.error(getApiErrorMessage(error, '个人资料保存失败'))
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
    void loadProfile()
  },
  { immediate: true }
)
</script>

<template>
  <AppPageContainer title="个人设置" description="完善公开资料和个人空间外观。">
    <nav class="settings-nav" aria-label="设置导航">
      <NuxtLink class="active" to="/settings/profile">个人资料</NuxtLink>
      <NuxtLink to="/settings/privacy">隐私设置</NuxtLink>
      <NuxtLink v-if="user?.username" :to="`/user/${user.username}`">查看个人空间</NuxtLink>
    </nav>

    <KunCard padding="lg">
      <a-spin :spinning="loading || saving">
        <a-form v-if="loaded" layout="vertical" @submit.prevent="save">
          <div class="asset-grid">
            <a-form-item label="头像">
              <ImageUploader
                v-model="avatarAsset"
                category="avatars"
                :preview-url="avatarTouched ? null : profile?.avatar_url"
                width="112px"
                height="112px"
                :disabled="saving"
                @success="setAvatar"
                @remove="clearAvatar"
              />
            </a-form-item>
            <a-form-item label="个人空间横幅">
              <ImageUploader
                v-model="bannerAsset"
                category="profile-banners"
                :preview-url="bannerTouched ? null : profile?.banner_url"
                width="min(100%, 420px)"
                height="140px"
                :disabled="saving"
                @success="setBanner"
                @remove="clearBanner"
              />
            </a-form-item>
          </div>

          <div class="form-grid">
            <a-form-item label="显示名称">
              <a-input v-model:value="form.display_name" :maxlength="100" placeholder="展示给其他用户的名称" />
            </a-form-item>
            <a-form-item label="性别">
              <a-select v-model:value="form.gender">
                <a-select-option value="undisclosed">不公开</a-select-option>
                <a-select-option value="male">男</a-select-option>
                <a-select-option value="female">女</a-select-option>
                <a-select-option value="non_binary">非二元</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item label="所在地">
              <a-input v-model:value="form.location" :maxlength="100" placeholder="城市或地区" />
            </a-form-item>
            <a-form-item label="生日">
              <a-input v-model:value="form.birthday" type="date" />
            </a-form-item>
            <a-form-item label="个人网站">
              <a-input v-model:value="form.website_url" :maxlength="2048" placeholder="https://example.com" />
            </a-form-item>
          </div>
          <a-form-item label="个人简介">
            <a-textarea v-model:value="form.bio" :rows="5" :maxlength="1000" show-count placeholder="介绍一下自己" />
          </a-form-item>
          <div class="save-row">
            <a-button type="primary" html-type="submit" :loading="saving">保存资料</a-button>
          </div>
        </a-form>
      </a-spin>
    </KunCard>
  </AppPageContainer>
</template>

<style scoped>
.settings-nav { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 16px; }
.settings-nav a { padding: 8px 14px; border-radius: var(--radius-kun-md); color: var(--color-default-500); }
.settings-nav a:hover, .settings-nav a.active { background: color-mix(in srgb, var(--color-primary) 12%, transparent); color: var(--color-primary); }
.asset-grid { display: grid; grid-template-columns: minmax(150px, 1fr) minmax(280px, 2fr); gap: 24px; }
.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 18px; }
.save-row { display: flex; justify-content: flex-end; }
@media (max-width: 639px) {
  .asset-grid, .form-grid { grid-template-columns: 1fr; }
}
</style>
