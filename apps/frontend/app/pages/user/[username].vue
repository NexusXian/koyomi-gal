<script setup lang="ts">
import { message } from 'ant-design-vue'
import {
  getPublicUserProfile,
  listUserActivities,
  listUserComments,
  listUserFavorites,
  listUserPosts,
  listUserRatings
} from '~/api/generated/users/users'
import type {
  DtoProfileCommentData,
  DtoProfileGalgameData,
  DtoProfilePostData,
  DtoPublicUserProfile,
  DtoUserActivityData
} from '~/api/generated/models'
import { formatDate } from '~/constants/domain'

type ProfileTab = 'home' | 'posts' | 'comments' | 'ratings' | 'favorites' | 'activity'

const route = useRoute()
const router = useRouter()
const { initialized } = storeToRefs(useUserStore())
const username = computed(() => String(route.params.username ?? ''))
const profile = ref<DtoPublicUserProfile | null>(null)
const profileLoading = ref(true)
const profileError = ref('')
const contentLoading = ref(false)
const posts = ref<DtoProfilePostData[]>([])
const comments = ref<DtoProfileCommentData[]>([])
const galgames = ref<DtoProfileGalgameData[]>([])
const activities = ref<DtoUserActivityData[]>([])
const total = ref(0)
const limit = 12
let contentSequence = 0

const requestedTab = computed<ProfileTab>(() => {
  const value = typeof route.query.tab === 'string' ? route.query.tab : 'home'
  return ['home', 'posts', 'comments', 'ratings', 'favorites', 'activity'].includes(value)
    ? value as ProfileTab
    : 'home'
})
const page = computed(() => Math.max(1, Number(route.query.page ?? 1) || 1))

const tabs = computed(() => {
  const access = profile.value?.access
  const result = [{ key: 'home', label: '主页', icon: 'lucide:house' }]
  if (access?.can_view_posts) result.push({ key: 'posts', label: '帖子', icon: 'lucide:messages-square' })
  if (access?.can_view_comments) result.push({ key: 'comments', label: '评论', icon: 'lucide:message-circle' })
  if (access?.can_view_ratings) result.push({ key: 'ratings', label: '评分', icon: 'lucide:star' })
  if (access?.can_view_favorites) result.push({ key: 'favorites', label: '收藏', icon: 'lucide:heart' })
  if (access?.can_view_activity) result.push({ key: 'activity', label: '动态', icon: 'lucide:activity' })
  return result
})
const activeTab = computed<ProfileTab>(() =>
  tabs.value.some((tab) => tab.key === requestedTab.value) ? requestedTab.value : 'home'
)
const totalPage = computed(() => Math.max(1, Math.ceil(total.value / limit)))

useSeoMeta({
  title: () => `${profile.value?.display_name || profile.value?.username || username.value}的个人空间 - Koyomi`,
  description: () => profile.value?.bio || `查看 ${username.value} 的个人空间`
})

async function loadProfile(): Promise<void> {
  if (import.meta.client && !initialized.value) return
  profileLoading.value = true
  profileError.value = ''
  profile.value = null
  try {
    profile.value = unwrapApiData(
      await getPublicUserProfile(username.value),
      '用户资料加载失败'
    )
    if (!tabs.value.some((tab) => tab.key === requestedTab.value)) {
      await router.replace({ path: route.path, query: {} })
    }
  } catch (error) {
    profileError.value = getApiErrorMessage(error, '用户不存在或资料加载失败')
  } finally {
    profileLoading.value = false
  }
}

async function loadContent(): Promise<void> {
  if (!profile.value || profile.value.is_restricted) return
  const sequence = ++contentSequence
  contentLoading.value = true
  posts.value = []
  comments.value = []
  galgames.value = []
  activities.value = []
  total.value = 0
  try {
    if (activeTab.value === 'posts') {
      const data = unwrapApiData(await listUserPosts(username.value, { page: page.value, limit }))
      if (sequence === contentSequence) {
        posts.value = data.items ?? []
        total.value = data.total ?? 0
      }
    } else if (activeTab.value === 'comments') {
      const data = unwrapApiData(await listUserComments(username.value, { page: page.value, limit }))
      if (sequence === contentSequence) {
        comments.value = data.items ?? []
        total.value = data.total ?? 0
      }
    } else if (activeTab.value === 'ratings' || activeTab.value === 'favorites') {
      const response = activeTab.value === 'ratings'
        ? await listUserRatings(username.value, { page: page.value, limit })
        : await listUserFavorites(username.value, { page: page.value, limit })
      const data = unwrapApiData(response)
      if (sequence === contentSequence) {
        galgames.value = data.items ?? []
        total.value = data.total ?? 0
      }
    } else if (profile.value.access?.can_view_activity) {
      const activityLimit = activeTab.value === 'home' ? 8 : limit
      const data = unwrapApiData(await listUserActivities(username.value, {
        page: activeTab.value === 'home' ? 1 : page.value,
        limit: activityLimit
      }))
      if (sequence === contentSequence) {
        activities.value = data.items ?? []
        total.value = data.total ?? 0
      }
    }
  } catch (error) {
    if (sequence === contentSequence) {
      message.error(getApiErrorMessage(error, '内容加载失败'))
    }
  } finally {
    if (sequence === contentSequence) contentLoading.value = false
  }
}

function changeTab(tab: string): void {
  void router.push({
    path: route.path,
    query: tab === 'home' ? {} : { tab }
  })
}

function changePage(next: number): void {
  void router.push({
    path: route.path,
    query: { tab: activeTab.value === 'home' ? undefined : activeTab.value, page: next > 1 ? next : undefined }
  })
}

watch([username, initialized], () => void loadProfile(), { immediate: true })
watch([activeTab, page, profile], () => void loadContent())
</script>

<template>
  <AppPageContainer width="wide">
    <div v-if="profileLoading" class="profile-loading">
      <KunSkeleton class="header-skeleton" />
      <KunSkeleton class="content-skeleton" />
    </div>

    <a-alert
      v-else-if="profileError"
      type="error"
      show-icon
      :message="profileError"
    >
      <template #action>
        <KunButton size="sm" color="danger" variant="light" @click="loadProfile">重试</KunButton>
      </template>
    </a-alert>

    <div v-else-if="profile" class="profile-page">
      <UserProfileHeader :profile="profile" />
      <PrivateProfile v-if="profile.is_restricted || !profile.access?.can_view_profile" :is-private="profile.is_private" />

      <template v-else>
        <UserProfileTabs :tabs="tabs" :active="activeTab" @change="changeTab" />

        <a-spin :spinning="contentLoading">
          <section v-if="activeTab === 'home'" class="profile-home">
            <UserProfileStats :profile="profile" />
            <KunCard v-if="profile.access?.can_view_activity" padding="lg">
              <KunHeader name="最近动态" description="近期公开的社区活动" scale="h3" />
              <KunNull v-if="!contentLoading && activities.length === 0" message="暂无公开动态" />
              <div v-else class="activity-list">
                <UserActivityItem v-for="item in activities" :key="item.id" :activity="item" />
              </div>
            </KunCard>
          </section>

          <section v-else-if="activeTab === 'posts'" class="content-list">
            <KunNull v-if="!contentLoading && posts.length === 0" message="暂无公开帖子" />
            <KunCard v-for="post in posts" :key="post.id" padding="md" :is-hoverable="true">
              <NuxtLink :to="`/posts/${post.id}`" class="content-row">
                <h2>{{ post.title || '未命名帖子' }}</h2>
                <p>{{ post.content }}</p>
                <div class="row-meta">
                  <span>{{ formatDate(post.created_at) }}</span>
                  <span><KunIcon name="lucide:thumbs-up" />{{ post.like_count ?? 0 }}</span>
                  <span><KunIcon name="lucide:message-circle" />{{ post.comment_count ?? 0 }}</span>
                  <span><KunIcon name="lucide:heart" />{{ post.favorite_count ?? 0 }}</span>
                </div>
              </NuxtLink>
            </KunCard>
          </section>

          <section v-else-if="activeTab === 'comments'" class="content-list">
            <KunNull v-if="!contentLoading && comments.length === 0" message="暂无公开评论" />
            <KunCard v-for="comment in comments" :key="comment.id" padding="md" :is-hoverable="true">
              <NuxtLink :to="`/posts/${comment.post_id}`" class="content-row">
                <span class="context-title">评论于 {{ comment.post_title || `帖子 #${comment.post_id}` }}</span>
                <p>{{ comment.content }}</p>
                <div class="row-meta"><span>{{ formatDate(comment.created_at) }}</span><span><KunIcon name="lucide:thumbs-up" />{{ comment.like_count ?? 0 }}</span></div>
              </NuxtLink>
            </KunCard>
          </section>

          <section v-else-if="activeTab === 'ratings' || activeTab === 'favorites'" class="galgame-grid">
            <KunNull v-if="!contentLoading && galgames.length === 0" :message="activeTab === 'ratings' ? '暂无公开评分' : '暂无公开收藏'" />
            <NuxtLink v-for="game in galgames" :key="game.id" :to="`/galgames/${game.id}`" class="galgame-row">
              <div class="game-cover">
                <SensitiveImage
                  v-if="game.cover_url"
                  :src="game.cover_url"
                  :alt="game.title || 'Galgame 封面'"
                  :sensitive="game.cover_sensitive"
                />
                <KunIcon v-else name="lucide:image" />
              </div>
              <div>
                <h2>{{ game.title || `Galgame #${game.id}` }}</h2>
                <span v-if="game.score != null" class="game-score"><KunIcon name="lucide:star" />{{ game.score }} 分</span>
                <span class="game-date">{{ formatDate(game.updated_at || game.created_at) }}</span>
              </div>
            </NuxtLink>
          </section>

          <section v-else class="activity-panel">
            <KunNull v-if="!contentLoading && activities.length === 0" message="暂无公开动态" />
            <UserActivityItem v-for="item in activities" v-else :key="item.id" :activity="item" />
          </section>
        </a-spin>

        <div v-if="activeTab !== 'home' && totalPage > 1" class="pagination-row">
          <KunPagination :current-page="page" :total-page="totalPage" :is-loading="contentLoading" @update:current-page="changePage" />
        </div>
      </template>
    </div>
  </AppPageContainer>
</template>

<style scoped>
.profile-page, .profile-loading, .profile-home, .content-list { display: flex; flex-direction: column; gap: 18px; }
.header-skeleton { height: 340px; }
.content-skeleton { height: 180px; }
.activity-list, .activity-panel { padding: 0 18px; border: 1px solid var(--app-glass-border); border-radius: var(--radius-kun-lg); background: var(--app-glass-background); }
.content-row { display: block; }
.content-row h2, .galgame-row h2 { margin: 0; font-size: 16px; }
.content-row p { display: -webkit-box; overflow: hidden; margin: 7px 0; color: var(--color-default-500); font-size: 14px; line-height: 1.65; white-space: pre-wrap; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
.row-meta { display: flex; flex-wrap: wrap; gap: 12px; color: var(--color-default-400); font-size: 12px; }
.row-meta span { display: inline-flex; align-items: center; gap: 4px; }
.context-title { color: var(--color-primary); font-size: 13px; font-weight: 600; }
.galgame-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 14px; }
.galgame-row { display: flex; min-width: 0; gap: 14px; padding: 14px; border: 1px solid var(--app-glass-border); border-radius: var(--radius-kun-lg); background: var(--app-glass-background); transition: border-color var(--kun-dur-fast); }
.galgame-row:hover { border-color: var(--color-primary); }
.game-cover { display: grid; overflow: hidden; width: 72px; height: 96px; flex: 0 0 72px; place-items: center; border-radius: var(--radius-kun-md); background: var(--color-default-100); color: var(--color-default-400); }
.game-score, .game-date { display: flex; align-items: center; gap: 4px; margin-top: 9px; color: var(--color-default-500); font-size: 13px; }
.game-score { color: var(--color-warning-600, #d97706); font-weight: 600; }
.pagination-row { display: flex; justify-content: center; }
</style>
