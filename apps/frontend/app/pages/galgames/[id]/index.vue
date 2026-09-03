<script setup lang="ts">
import { message } from 'ant-design-vue'
import {
  addGalgameFavorite,
  deleteGalgame,
  deleteGalgameRating,
  deleteGalgameUserState,
  getGalgame,
  getMyGalgameRelation,
  removeGalgameFavorite,
  upsertGalgameRating,
  upsertGalgameUserState
} from '~/api/generated/galgames/galgames'
import {
  deleteResource,
  listGalgameResources
} from '~/api/generated/resources/resources'
import { listPosts } from '~/api/generated/posts/posts'
import type {
  DtoGalgameListData,
  DtoGalgameResponse,
  DtoPostListData,
  DtoResourceData,
  DtoResourceListData
} from '~/api/generated/models'
import {
  AGE_RATINGS,
  RESOURCE_STATUS,
  RESOURCE_TYPES,
  USER_STATES,
  domainLabel
} from '~/constants/domain'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { isAuthenticated } = storeToRefs(userStore)
const { has } = usePermissions()

const galgameId = computed(() => Number(route.params.id))

const { data: galgame, error } = await useAsyncData<DtoGalgameResponse, Error>(
  `galgame-${galgameId.value}`,
  async () =>
    unwrapApiData(
      await getGalgame(galgameId.value),
      '加载 Galgame 失败'
    )
)

if (error.value || !galgame.value) {
  throw createError({
    statusCode: 404,
    statusMessage: 'Galgame 不存在或尚未发布',
    fatal: true
  })
}

useSeoMeta({
  title: () => `${galgame.value?.title ?? 'Galgame'} - Koyomi`,
  description: () => galgame.value?.description ?? 'Galgame 详情'
})

const resources = ref<DtoResourceData[]>([])
const resourceTotal = ref(0)
const resourcePage = ref(1)
const resourceLimit = 20
const resourcesLoading = ref(false)
const deletingGalgame = ref(false)
const deletingResourceId = ref<number | null>(null)
const editingResource = ref<DtoResourceData | null>(null)
const editResourceOpen = ref(false)
const relatedPosts = ref<DtoPostListData['items']>([])
const relationLoaded = ref(false)
const favorited = ref(false)
const favoritePending = ref(false)
const myScore = ref(0)
const myState = ref<number | undefined>(undefined)
const myPlayHours = ref<number>(0)
const statePending = ref(false)

const ratingAverage = computed(() =>
  galgame.value?.rating?.average !== undefined
    ? Number(galgame.value.rating.average.toFixed(1))
    : null
)

const postsLink = computed(
  () => `/posts?galgame_id=${galgameId.value}`
)

const resourceTotalPage = computed(() =>
  Math.max(1, Math.ceil(resourceTotal.value / resourceLimit))
)

async function loadResources(): Promise<void> {
  resourcesLoading.value = true
  try {
    const data = unwrapApiData<DtoResourceListData>(
      await listGalgameResources(galgameId.value, {
        page: resourcePage.value,
        limit: resourceLimit
      })
    )
    resources.value = data.items ?? []
    resourceTotal.value = data.total ?? 0
  } catch {
    resources.value = []
    resourceTotal.value = 0
  } finally {
    resourcesLoading.value = false
  }
}

function updateResourcePage(next: number): void {
  resourcePage.value = next
  void loadResources()
}

function handleResourceCreated(): void {
  resourcePage.value = 1
  void loadResources()
}

function isResourceOwner(resource: DtoResourceData): boolean {
  return Boolean(
    isAuthenticated.value &&
      userStore.getUser?.id &&
      resource.uploader_id === userStore.getUser.id
  )
}

function canEditResource(resource: DtoResourceData): boolean {
  return isResourceOwner(resource) || has('resource:update')
}

function canDeleteResource(resource: DtoResourceData): boolean {
  return isResourceOwner(resource) || has('resource:delete')
}

function openResourceEdit(resource: DtoResourceData): void {
  editingResource.value = resource
  editResourceOpen.value = true
}

async function reloadResourcePage(): Promise<void> {
  await loadResources()
  if (resources.value.length === 0 && resourcePage.value > 1) {
    resourcePage.value -= 1
    await loadResources()
  }
}

async function removeResource(resource: DtoResourceData): Promise<void> {
  if (!resource.id) {
    return
  }

  deletingResourceId.value = resource.id
  try {
    await deleteResource(resource.id)
    message.success('资源已删除')
    await reloadResourcePage()
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除资源失败'))
  } finally {
    deletingResourceId.value = null
  }
}

async function removeGalgame(): Promise<void> {
  deletingGalgame.value = true
  try {
    await deleteGalgame(galgameId.value)
    message.success('Galgame 已删除')
    await router.push('/galgames')
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除 Galgame 失败'))
  } finally {
    deletingGalgame.value = false
  }
}

async function loadRelation(): Promise<void> {
  if (!isAuthenticated.value) {
    relationLoaded.value = true
    return
  }

  try {
    const relation = unwrapApiData(
      await getMyGalgameRelation(galgameId.value)
    )
    favorited.value = Boolean(relation.favorite?.favorited)
    myScore.value = relation.rating?.score ?? 0
    myState.value = relation.state?.state
    myPlayHours.value = Math.floor(
      (relation.state?.play_time_minutes ?? 0) / 60
    )
  } catch {
    /* 未登录或接口失败时忽略 */
  } finally {
    relationLoaded.value = true
  }
}

async function loadPosts(): Promise<void> {
  try {
    relatedPosts.value =
      unwrapApiData(await listPosts({ galgame_id: galgameId.value, limit: 5 }))
        ?.items ?? []
  } catch {
    relatedPosts.value = []
  }
}

async function toggleFavorite(): Promise<void> {
  if (!isAuthenticated.value) {
    message.warning('登录后才能收藏')
    void router.push('/login')
    return
  }

  favoritePending.value = true
  try {
    if (favorited.value) {
      await removeGalgameFavorite(galgameId.value)
      favorited.value = false
      message.success('已取消收藏')
    } else {
      const data = unwrapApiData(await addGalgameFavorite(galgameId.value))
      favorited.value = Boolean(data.favorited)
      message.success('已收藏')
    }
  } catch (error) {
    message.error(getApiErrorMessage(error, '操作失败'))
  } finally {
    favoritePending.value = false
  }
}

async function submitScore(score: number): Promise<void> {
  if (!isAuthenticated.value) {
    message.warning('登录后才能评分')
    return
  }

  try {
    if (score === myScore.value) {
      return
    }

    const data = unwrapApiData(
      await upsertGalgameRating(galgameId.value, {
        score: score as 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10
      })
    )
    myScore.value = data.score ?? score
    message.success(`已评分 ${score} 分`)
    void refreshNuxtData(`galgame-${galgameId.value}`)
  } catch (error) {
    message.error(getApiErrorMessage(error, '评分失败'))
  }
}

async function clearScore(): Promise<void> {
  try {
    await deleteGalgameRating(galgameId.value)
    myScore.value = 0
    message.success('已删除评分')
    void refreshNuxtData(`galgame-${galgameId.value}`)
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除评分失败'))
  }
}

async function submitState(): Promise<void> {
  if (myState.value === undefined) {
    return
  }

  statePending.value = true
  try {
    await upsertGalgameUserState(galgameId.value, {
      state: myState.value as 1 | 2 | 3 | 4 | 5,
      play_time_minutes: Math.max(0, Math.round(myPlayHours.value * 60))
    })
    message.success('游玩状态已保存')
  } catch (error) {
    message.error(getApiErrorMessage(error, '保存失败'))
  } finally {
    statePending.value = false
  }
}

async function clearState(): Promise<void> {
  try {
    await deleteGalgameUserState(galgameId.value)
    myState.value = undefined
    myPlayHours.value = 0
    message.success('已清除游玩状态')
  } catch (error) {
    message.error(getApiErrorMessage(error, '清除失败'))
  }
}

const uploadOpen = ref(false)
const reportResourceId = ref<number | null>(null)
const reportOpen = ref(false)

function openReport(resourceId: number): void {
  reportResourceId.value = resourceId
  reportOpen.value = true
}

onMounted(() => {
  void loadResources()
  void loadRelation()
  void loadPosts()
})
</script>

<template>
  <AppPageContainer width="wide">
    <nav class="breadcrumb" aria-label="面包屑">
      <NuxtLink to="/galgames">Galgame</NuxtLink>
      <KunIcon name="lucide:chevron-right" />
      <span>{{ galgame?.title }}</span>
    </nav>

    <KunCard padding="none" class-name="detail-card" content-class="detail-content">
      <div
        v-if="galgame?.banner_url"
        class="detail-banner"
        :style="{ backgroundImage: `url(${galgame.banner_url})` }"
      />

      <div class="detail-layout">
        <div class="detail-cover">
          <img
            :src="galgame?.cover_url"
            :alt="galgame?.title || '封面'"
          />
        </div>

        <div class="detail-info">
          <div class="detail-title-row">
            <h1>{{ galgame?.title }}</h1>
            <KunChip
              v-if="galgame?.age_rating"
              :color="galgame.age_rating === 3 ? 'danger' : 'primary'"
              variant="flat"
              size="sm"
            >
              {{ domainLabel(AGE_RATINGS, galgame?.age_rating) }}
            </KunChip>
          </div>

          <p v-if="galgame?.original_title" class="detail-subtitle">
            {{ galgame.original_title }}
            <template v-if="galgame.romaji_title">
              · {{ galgame.romaji_title }}
            </template>
          </p>

          <dl class="detail-meta">
            <div v-if="galgame?.developer?.name" class="meta-row">
              <dt><KunIcon name="lucide:building-2" />开发商</dt>
              <dd>{{ galgame.developer.name }}</dd>
            </div>
            <div v-if="galgame?.release_date" class="meta-row">
              <dt><KunIcon name="lucide:calendar" />发行日期</dt>
              <dd>{{ galgame.release_date.slice(0, 10) }}</dd>
            </div>
            <div class="meta-row">
              <dt><KunIcon name="lucide:star" />评分</dt>
              <dd>
                <template v-if="ratingAverage !== null">
                  {{ ratingAverage }} / 10（{{ galgame?.rating?.count ?? 0 }} 人）
                </template>
                <template v-else>暂无评分</template>
              </dd>
            </div>
            <div class="meta-row">
              <dt><KunIcon name="lucide:heart" />收藏</dt>
              <dd>{{ galgame?.statistics?.favorite_count ?? 0 }}</dd>
            </div>
          </dl>

          <div v-if="galgame?.tags?.length" class="detail-tags">
            <KunChip
              v-for="tag in galgame.tags"
              :key="tag.id"
              variant="flat"
              size="sm"
            >
              {{ tag.name }}
            </KunChip>
          </div>

          <div class="detail-actions">
            <KunButton
              :color="favorited ? 'danger' : 'primary'"
              :variant="favorited ? 'solid' : 'bordered'"
              :disabled="favoritePending"
              @click="toggleFavorite"
            >
              <KunIcon :name="favorited ? 'lucide:heart' : 'lucide:heart'" />
              {{ favorited ? '已收藏' : '收藏' }}
            </KunButton>

            <KunButton color="primary" variant="light" :href="postsLink">
              <KunIcon name="lucide:message-square-text" />
              相关帖子
            </KunButton>

            <KunButton
              v-if="has('galgame:update')"
              color="default"
              variant="bordered"
              :href="`/galgames/${galgameId}/edit`"
            >
              <KunIcon name="lucide:pencil" />
              编辑
            </KunButton>

            <a-popconfirm
              v-if="has('galgame:delete')"
              :title="`确定删除「${galgame?.title ?? '该 Galgame'}」吗？此操作无法撤销。`"
              ok-text="删除"
              cancel-text="取消"
              @confirm="removeGalgame"
            >
              <KunButton
                color="danger"
                variant="bordered"
                :disabled="deletingGalgame"
              >
                <KunIcon name="lucide:trash-2" />
                删除
              </KunButton>
            </a-popconfirm>
          </div>
        </div>
      </div>

      <div class="detail-body">
        <KunHeader name="简介" scale="h3" class="section-heading" />
        <p class="detail-description">
          {{ galgame?.description || '暂无简介' }}
        </p>
        <p v-if="galgame?.aliases?.length" class="detail-aliases">
          别名：{{ galgame.aliases.join('、') }}
        </p>
      </div>
    </KunCard>

    <GalgameGallery :galgame-id="galgameId" :game-title="galgame?.title" />

    <div class="side-grid">
      <KunCard padding="lg" class-name="relation-card">
        <KunHeader name="我的评分" scale="h3" class="section-heading" />
        <template v-if="isAuthenticated">
          <div class="rating-row">
            <KunRating
              :model-value="myScore"
              :max="10"
              :readonly="false"
              @set="submitScore"
            />
            <span class="rating-current">
              {{ myScore > 0 ? `${myScore} 分` : '未评分' }}
            </span>
          </div>
          <a-button
            v-if="myScore > 0"
            type="link"
            size="small"
            danger
            @click="clearScore"
          >
            删除评分
          </a-button>
        </template>
        <p v-else class="login-hint">
          <NuxtLink to="/login">登录</NuxtLink>
          后即可评分
        </p>

        <KunDivider />

        <KunHeader name="游玩状态" scale="h3" class="section-heading" />
        <template v-if="isAuthenticated && relationLoaded">
          <div class="state-form">
            <a-select
              v-model:value="myState"
              class="state-select"
              placeholder="选择状态"
              allow-clear
              :options="
                USER_STATES.map((item) => ({
                  value: item.value,
                  label: item.label
                }))
              "
            />
            <a-input-number
              v-model:value="myPlayHours"
              class="state-hours"
              :min="0"
              :precision="1"
              addon-after="小时"
            />
            <a-button
              type="primary"
              :loading="statePending"
              :disabled="myState === undefined"
              @click="submitState"
            >
              保存
            </a-button>
            <a-button
              v-if="myState !== undefined"
              danger
              @click="clearState"
            >
              清除
            </a-button>
          </div>
        </template>
        <p v-else class="login-hint">
          <NuxtLink to="/login">登录</NuxtLink>
          后可记录游玩状态
        </p>
      </KunCard>

      <KunCard padding="lg" class-name="posts-card">
        <div class="section-head-row">
          <KunHeader name="相关帖子" scale="h3" class="section-heading" />
          <NuxtLink class="section-more" :to="postsLink">
            更多
            <KunIcon name="lucide:arrow-right" />
          </NuxtLink>
        </div>

        <KunNull v-if="!relatedPosts?.length" message="还没有相关帖子" />

        <ul v-else class="related-post-list">
          <li v-for="post in relatedPosts" :key="post.id">
            <NuxtLink :to="`/posts/${post.id}`" class="related-post-link">
              <span class="related-post-title">{{ post.title }}</span>
              <span class="related-post-meta">
                <KunIcon name="lucide:message-circle" />
                {{ post.comment_count ?? 0 }}
                <KunIcon name="lucide:thumbs-up" />
                {{ post.like_count ?? 0 }}
              </span>
            </NuxtLink>
            <UserLink
              v-if="post.author?.username"
              class="related-post-author"
              :username="post.author.username"
              :display-name="post.author.display_name || post.author_name"
              :user-id="post.author.id || post.author_id"
            >
              <UserAvatar
                :avatar-url="post.author.avatar_url || post.author_avatar"
                :display-name="post.author.display_name || post.author_name"
                :username="post.author.username"
                size="sm"
              />
              {{ post.author.display_name || post.author_name || post.author.username }}
            </UserLink>
          </li>
        </ul>

        <KunButton
          v-if="isAuthenticated"
          color="primary"
          variant="flat"
          class="new-post-button"
          :href="`/posts/new?galgame_id=${galgameId}`"
        >
          <KunIcon name="lucide:plus" />
          发表相关帖子
        </KunButton>
      </KunCard>
    </div>

    <KunCard padding="lg" class-name="resource-card">
      <div class="section-head-row">
        <KunHeader
          name="资源"
          :description="`共 ${resourceTotal} 个资源`"
          scale="h3"
          class="section-heading"
        />
        <KunButton
          v-if="isAuthenticated"
          color="primary"
          size="sm"
          @click="uploadOpen = true"
        >
          <KunIcon name="lucide:upload" />
          上传资源
        </KunButton>
      </div>

      <a-spin :spinning="resourcesLoading">
        <KunNull v-if="resources.length === 0" message="暂无资源" />

        <div v-else class="resource-list">
          <div v-for="resource in resources" :key="resource.id" class="resource-item">
            <div class="resource-head">
              <KunChip variant="flat" size="sm">
                {{ domainLabel(RESOURCE_TYPES, resource.type) }}
              </KunChip>
              <h4 class="resource-title">{{ resource.title }}</h4>
              <a-tag
                v-if="resource.status !== undefined && resource.status !== 1"
                :color="RESOURCE_STATUS[resource.status]?.color"
              >
                {{ domainLabel(RESOURCE_STATUS, resource.status) }}
              </a-tag>
            </div>

            <p v-if="resource.description" class="resource-description">
              {{ resource.description }}
            </p>

            <div v-if="resource.uploader?.username" class="resource-uploader">
              <span>贡献者</span>
              <UserLink
                :username="resource.uploader.username"
                :display-name="resource.uploader.display_name"
                :user-id="resource.uploader.id || resource.uploader_id"
              >
                <UserAvatar
                  :avatar-url="resource.uploader.avatar_url"
                  :display-name="resource.uploader.display_name"
                  :username="resource.uploader.username"
                  size="sm"
                />
                {{ resource.uploader.display_name || resource.uploader.username }}
              </UserLink>
            </div>

            <ul class="resource-links">
              <li v-for="link in resource.links" :key="link.id">
                <KunCopy :text="link.url ?? ''">
                  <span class="resource-link-url">{{ link.url }}</span>
                </KunCopy>
                <a
                  v-if="link.url"
                  :href="link.url"
                  target="_blank"
                  rel="noopener noreferrer nofollow"
                  class="resource-link-open"
                >
                  <KunIcon name="lucide:external-link" />
                  打开
                </a>
              </li>
            </ul>

            <div class="resource-actions">
              <a-button
                v-if="canEditResource(resource)"
                size="small"
                @click="openResourceEdit(resource)"
              >
                <template #icon><KunIcon name="lucide:pencil" /></template>
                编辑
              </a-button>
              <a-popconfirm
                v-if="canDeleteResource(resource)"
                :title="`确定删除资源「${resource.title ?? resource.id}」吗？`"
                ok-text="删除"
                cancel-text="取消"
                @confirm="removeResource(resource)"
              >
                <a-button
                  size="small"
                  danger
                  :loading="deletingResourceId === resource.id"
                >
                  <template #icon><KunIcon name="lucide:trash-2" /></template>
                  删除
                </a-button>
              </a-popconfirm>
              <a-button size="small" danger @click="openReport(resource.id ?? 0)">
                <template #icon><KunIcon name="lucide:flag" /></template>
                举报
              </a-button>
            </div>
          </div>
        </div>

        <div v-if="resourceTotalPage > 1" class="resource-pagination">
          <KunPagination
            :current-page="resourcePage"
            :total-page="resourceTotalPage"
            :is-loading="resourcesLoading"
            @update:current-page="updateResourcePage"
          />
        </div>
      </a-spin>
    </KunCard>

    <ResourceUploadModal
      v-model:open="uploadOpen"
      :galgame-id="galgameId"
      @created="handleResourceCreated"
    />

    <ResourceEditModal
      v-model:open="editResourceOpen"
      :resource="editingResource"
      @updated="reloadResourcePage"
    />

    <ResourceReportModal
      v-model:open="reportOpen"
      :resource-id="reportResourceId"
    />
  </AppPageContainer>
</template>

<style scoped>
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 14px;
  color: var(--color-default-500);
  font-size: 14px;
}

.breadcrumb a {
  color: var(--color-default-500);
}

.breadcrumb a:hover {
  color: var(--color-primary);
}

.detail-card {
  overflow: hidden;
  margin-bottom: 18px;
}

:deep(.detail-content) {
  padding: 0;
}

.detail-banner {
  height: 200px;
  background-color: var(--color-content2);
  background-position: center;
  background-size: cover;
}

.detail-layout {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 20px;
}

.detail-cover {
  flex: 0 0 auto;
}

.detail-cover img {
  display: block;
  width: 200px;
  max-width: 100%;
  border-radius: var(--radius-kun-lg);
  box-shadow: var(--shadow-kun-lg);
  object-fit: cover;
}

.detail-info {
  min-width: 0;
}

.detail-title-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.detail-title-row h1 {
  margin: 0;
  font-size: clamp(22px, 3vw, 30px);
  font-weight: 800;
  letter-spacing: -0.02em;
}

.detail-subtitle {
  margin: 6px 0 0;
  color: var(--color-default-500);
  font-size: 14px;
}

.detail-meta {
  display: grid;
  margin: 16px 0 0;
  gap: 8px;
}

.meta-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.meta-row dt {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-width: 84px;
  color: var(--color-default-500);
}

.meta-row dd {
  margin: 0;
  color: var(--color-foreground);
  font-weight: 500;
}

.detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 14px;
}

.detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 18px;
}

.detail-body {
  padding: 0 20px 22px;
}

.detail-description {
  margin: 8px 0 0;
  color: var(--color-default-600);
  font-size: 15px;
  line-height: 1.85;
  white-space: pre-wrap;
}

.detail-aliases {
  margin: 12px 0 0;
  color: var(--color-default-400);
  font-size: 13px;
}

.side-grid {
  display: grid;
  gap: 18px;
  margin-bottom: 18px;
}

.section-heading {
  margin-bottom: 4px;
}

.section-head-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.section-more {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--color-primary);
  font-size: 14px;
}

.rating-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 10px 0 4px;
}

.rating-current {
  color: var(--color-default-600);
  font-size: 14px;
}

.login-hint {
  margin: 10px 0 0;
  color: var(--color-default-500);
  font-size: 14px;
}

.login-hint a {
  color: var(--color-primary);
}

.state-form {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.state-select {
  width: 130px;
}

.state-hours {
  width: 130px;
}

.related-post-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 10px 0 0;
  padding: 0;
  list-style: none;
}

.related-post-link {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--radius-kun-md);
  transition: background var(--kun-dur-fast) var(--ease-kun-standard);
}

.related-post-link:hover {
  background: color-mix(in srgb, var(--color-primary) 8%, transparent);
}

.related-post-title {
  overflow: hidden;
  color: var(--color-foreground);
  font-size: 14px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.related-post-meta {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 4px;
  color: var(--color-default-400);
  font-size: 12px;
}

.related-post-author {
  margin: 2px 10px 8px;
  color: var(--color-default-500);
  font-size: 12px;
}

.new-post-button {
  margin-top: 12px;
}

.resource-card {
  margin-bottom: 4px;
}

.resource-list {
  display: grid;
  gap: 12px;
  margin-top: 12px;
}

.resource-item {
  padding: 14px;
  border: 1px solid var(--color-default-200);
  border-radius: var(--radius-kun-lg);
}

.resource-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.resource-title {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
}

.resource-description {
  margin: 8px 0 0;
  color: var(--color-default-500);
  font-size: 13px;
  line-height: 1.7;
}

.resource-uploader {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  color: var(--color-default-400);
  font-size: 12px;
}

.resource-links {
  margin: 10px 0 0;
  padding: 0;
  list-style: none;
}

.resource-links li {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 0;
  border-bottom: 1px dashed var(--color-default-200);
}

.resource-links li:last-child {
  border-bottom: 0;
}

.resource-link-url {
  min-width: 0;
  overflow: hidden;
  color: var(--color-primary-600);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.resource-link-open {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 4px;
  color: var(--color-default-500);
  font-size: 13px;
}

.resource-link-open:hover {
  color: var(--color-primary);
}

.resource-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

.resource-pagination {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

@media (min-width: 768px) {
  .detail-layout {
    flex-direction: row;
    padding: 24px;
  }

  .side-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
