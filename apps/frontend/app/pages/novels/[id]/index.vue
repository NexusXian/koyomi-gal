<script setup lang="ts">
import { message } from 'ant-design-vue'
import {
  deleteNovel,
  getNovel
} from '~/api/generated/novels/novels'
import {
  deleteResource,
  listNovelResources
} from '~/api/generated/resources/resources'
import type {
  DtoNovelResponse,
  DtoRelatedWorkData,
  DtoResourceData,
  DtoResourceListData
} from '~/api/generated/models'
import {
  AGE_RATINGS,
  NOVEL_RELEASE_STATUS,
  RESOURCE_STATUS,
  RESOURCE_TYPES,
  domainLabel,
  domainSlug
} from '~/constants/domain'
import { stripMarkdownForExcerpt } from '~/utils/markdown'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { isAuthenticated } = storeToRefs(userStore)
const { has } = usePermissions()

const novelId = computed(() => Number(route.params.id))

const { data: novel, error } = await useAsyncData<DtoNovelResponse, Error>(
  `novel-${novelId.value}`,
  async () => unwrapApiData(await getNovel(novelId.value), '加载小说失败')
)

if (error.value || !novel.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '小说不存在或尚未发布',
    fatal: true
  })
}

const releaseStatusLabel = computed(() => {
  const option = domainSlug(NOVEL_RELEASE_STATUS, novel.value?.release_status ?? '')
  return option?.label ?? '未知'
})

const relationTypeLabels: Record<string, string> = {
  adaptation: '改编',
  original: '原作',
  spin_off: '衍生',
  sequel: '续作',
  prequel: '前作',
  same_series: '同系列',
  related: '相关'
}

function relationTypeLabel(relationType?: string): string {
  return relationTypeLabels[relationType ?? ''] ?? '相关'
}

useSeoMeta({
  title: () => `${novel.value?.title ?? '小说'} - Koyomi`,
  description: () =>
    stripMarkdownForExcerpt(novel.value?.summary ?? '', 160) || '小说详情',
  ogTitle: () => `${novel.value?.title ?? '小说'} - Koyomi`,
  ogDescription: () =>
    stripMarkdownForExcerpt(novel.value?.summary ?? '', 160) || '小说详情',
  ogImage: () => novel.value?.cover_url || undefined
})

const resources = ref<DtoResourceData[]>([])
const resourceTotal = ref(0)
const resourcePage = ref(1)
const resourceLimit = 20
const resourcesLoading = ref(false)
const deletingNovel = ref(false)
const deletingResourceId = ref<number | null>(null)
const editingResource = ref<DtoResourceData | null>(null)
const editResourceOpen = ref(false)
const uploadOpen = ref(false)
const reportResourceId = ref<number | null>(null)
const reportOpen = ref(false)

const resourceTotalPage = computed(() =>
  Math.max(1, Math.ceil(resourceTotal.value / resourceLimit))
)

async function loadResources(): Promise<void> {
  resourcesLoading.value = true
  try {
    const data = unwrapApiData<DtoResourceListData>(
      await listNovelResources(novelId.value, {
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

function canEditResource(resource: DtoResourceData): boolean {
  if (has('resource:update')) {
    return true
  }
  return (
    isAuthenticated.value &&
    resource.uploader_id !== undefined &&
    resource.uploader_id !== null &&
    resource.uploader_id === userStore.getUser?.id
  )
}

function canDeleteResource(resource: DtoResourceData): boolean {
  return canEditResource(resource) || has('resource:delete')
}

function openResourceEdit(resource: DtoResourceData): void {
  editingResource.value = resource
  editResourceOpen.value = true
}

function openReport(resourceId: number): void {
  reportResourceId.value = resourceId
  reportOpen.value = true
}

async function removeResource(resource: DtoResourceData): Promise<void> {
  deletingResourceId.value = resource.id ?? null
  try {
    await deleteResource(resource.id ?? 0)
    message.success('资源已删除')
    if (resources.value.length === 1 && resourcePage.value > 1) {
      resourcePage.value -= 1
    }
    await loadResources()
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除资源失败'))
  } finally {
    deletingResourceId.value = null
  }
}

async function removeNovel(): Promise<void> {
  deletingNovel.value = true
  try {
    await deleteNovel(novelId.value)
    message.success('小说已删除')
    void router.replace('/novels')
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除小说失败'))
  } finally {
    deletingNovel.value = false
  }
}

onMounted(() => {
  void loadResources()
})
</script>

<template>
  <AppPageContainer width="wide">
    <nav class="breadcrumb" aria-label="面包屑">
      <NuxtLink to="/novels">小说</NuxtLink>
      <KunIcon name="lucide:chevron-right" />
      <span>{{ novel?.title }}</span>
    </nav>

    <KunCard padding="none" class-name="detail-card" content-class="detail-content">
      <div class="detail-layout">
        <div class="detail-cover">
          <SensitiveImage
            :src="novel?.cover_url"
            :alt="novel?.title || '封面'"
            :sensitive="novel?.is_cover_sensitive"
          />
        </div>

        <div class="detail-info">
          <div class="detail-title-row">
            <h1>{{ novel?.title }}</h1>
            <KunChip
              v-if="novel?.age_rating"
              :color="novel.age_rating === 3 ? 'danger' : 'primary'"
              variant="flat"
              size="sm"
            >
              {{ domainLabel(AGE_RATINGS, novel?.age_rating) }}
            </KunChip>
            <KunChip
              v-if="novel?.release_status && novel.release_status !== 'unknown'"
              variant="flat"
              size="sm"
            >
              {{ releaseStatusLabel }}
            </KunChip>
          </div>

          <p v-if="novel?.original_title" class="detail-subtitle">
            {{ novel.original_title }}
          </p>

          <dl class="detail-meta">
            <div v-if="novel?.author" class="meta-row">
              <dt><KunIcon name="lucide:pen-line" />作者</dt>
              <dd>{{ novel.author }}</dd>
            </div>
            <div v-if="novel?.illustrator" class="meta-row">
              <dt><KunIcon name="lucide:brush" />插画师</dt>
              <dd>{{ novel.illustrator }}</dd>
            </div>
            <div v-if="novel?.publisher" class="meta-row">
              <dt><KunIcon name="lucide:book-marked" />出版社</dt>
              <dd>{{ novel.publisher }}</dd>
            </div>
            <div v-if="novel?.label" class="meta-row">
              <dt><KunIcon name="lucide:library" />文库</dt>
              <dd>{{ novel.label }}</dd>
            </div>
            <div v-if="novel?.first_release_date" class="meta-row">
              <dt><KunIcon name="lucide:calendar" />首次发售</dt>
              <dd>{{ novel.first_release_date.slice(0, 10) }}</dd>
            </div>
            <div class="meta-row">
              <dt><KunIcon name="lucide:layers" />卷数</dt>
              <dd>{{ novel?.statistics?.volume_count ?? 0 }}</dd>
            </div>
            <div v-if="novel?.language" class="meta-row">
              <dt><KunIcon name="lucide:languages" />语言</dt>
              <dd>{{ novel.language }}</dd>
            </div>
          </dl>

          <div v-if="novel?.tags?.length" class="detail-tags">
            <KunChip v-for="tag in novel.tags" :key="tag.id" variant="flat" size="sm">
              {{ tag.name }}
            </KunChip>
          </div>

          <div class="detail-actions">
            <KunButton
              v-if="has('novel:update')"
              color="default"
              variant="bordered"
              :href="`/novels/${novelId}/edit`"
            >
              <KunIcon name="lucide:pencil" />
              编辑
            </KunButton>

            <a-popconfirm
              v-if="has('novel:delete')"
              :title="`确定删除「${novel?.title ?? '该小说'}」吗？`"
              ok-text="删除"
              cancel-text="取消"
              @confirm="removeNovel"
            >
              <KunButton
                color="danger"
                variant="bordered"
                :disabled="deletingNovel"
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
        <PostContent
          v-if="novel?.summary"
          class="detail-description-markdown"
          :content="novel.summary"
          mode="markdown"
        />
        <p v-else class="detail-description">暂无简介</p>
        <p v-if="novel?.official_website" class="detail-website">
          <KunIcon name="lucide:globe" />
          <a
            :href="novel.official_website"
            target="_blank"
            rel="noopener noreferrer nofollow"
          >
            {{ novel.official_website }}
          </a>
        </p>
      </div>
    </KunCard>

    <KunCard padding="lg" class-name="volumes-card">
      <div class="section-head-row">
        <KunHeader
          name="卷册"
          :description="`共 ${novel?.statistics?.volume_count ?? 0} 卷`"
          scale="h3"
          class="section-heading"
        />
      </div>

      <KunNull
        v-if="!novel?.volumes?.length"
        message="还没有卷册资料"
        :description="has('novel:update') ? '可在编辑页添加卷册' : undefined"
      />

      <div v-else class="volume-grid">
        <div
          v-for="volume in novel.volumes"
          :key="volume.id"
          class="volume-item"
        >
          <div class="volume-cover">
            <SensitiveImage
              :src="volume.cover_url || novel?.cover_url"
              :alt="volume.title || `Vol.${volume.volume_number}`"
              :sensitive="novel?.is_cover_sensitive"
            />
          </div>
          <div class="volume-info">
            <span v-if="volume.volume_number !== undefined && volume.volume_number !== null" class="volume-number">
              Vol.{{ volume.volume_number }}
            </span>
            <h4 class="volume-title">{{ volume.title || '未命名卷' }}</h4>
            <p v-if="volume.original_title" class="volume-original-title">
              {{ volume.original_title }}
            </p>
            <p v-if="volume.release_date" class="volume-meta">
              <KunIcon name="lucide:calendar" />
              {{ volume.release_date.slice(0, 10) }}
            </p>
            <p v-if="volume.isbn" class="volume-meta">
              ISBN {{ volume.isbn }}
            </p>
          </div>
        </div>
      </div>
    </KunCard>

    <KunCard v-if="novel?.related_galgames?.length" padding="lg" class-name="relations-card">
      <div class="section-head-row">
        <KunHeader name="关联视觉小说" scale="h3" class="section-heading" />
      </div>

      <div class="relation-grid">
        <NuxtLink
          v-for="work in novel.related_galgames"
          :key="work.relation_id"
          :to="`/galgames/${work.work_id}`"
          class="relation-item"
        >
          <div class="relation-cover">
            <SensitiveImage
              :src="work.cover_url"
              :alt="work.title"
              :sensitive="work.cover_sensitive"
            />
            <span v-if="work.age_rating === 3" class="age-badge">R18</span>
          </div>
          <div class="relation-info">
            <h4 class="relation-title">{{ work.title }}</h4>
            <KunChip size="sm" variant="flat">
              {{ relationTypeLabel(work.relation_type) }}
            </KunChip>
          </div>
        </NuxtLink>
      </div>
    </KunCard>

    <KunCard padding="lg" class-name="contributors-card">
      <div class="section-head-row">
        <KunHeader
          name="贡献者"
          :description="`共 ${novel?.contributor_count ?? 0} 位贡献者`"
          scale="h3"
          class="section-heading"
        />
      </div>
      <NovelContributors
        :contributors="novel?.contributors"
        :contributor-count="novel?.contributor_count"
      />
    </KunCard>

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
              <a-button size="small" @click="openReport(resource.id ?? 0)">
                <template #icon><KunIcon name="lucide:flag" /></template>
                举报
              </a-button>
            </div>
          </div>
        </div>

        <div v-if="resourceTotalPage > 1" class="pagination-row">
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
      :target-type="'novel'"
      :target-id="novelId"
      @created="loadResources"
    />

    <ResourceEditModal
      v-model:open="editResourceOpen"
      :resource="editingResource"
      @updated="loadResources"
    />

    <ResourceReportModal
      v-model:open="reportOpen"
      :resource-id="reportResourceId ?? 0"
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
  font-size: 13px;
}

.breadcrumb a {
  color: var(--color-default-600);
}

.detail-layout {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 24px;
  padding: 24px;
}

.detail-cover {
  width: 240px;
}

.detail-cover :deep(img) {
  width: 240px;
  height: 340px;
  border-radius: var(--radius-kun-md);
  object-fit: cover;
  box-shadow: var(--shadow-kun-md);
}

.detail-title-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.detail-title-row h1 {
  margin: 0;
  font-size: 22px;
}

.detail-subtitle {
  margin: 6px 0 0;
  color: var(--color-default-500);
  font-size: 14px;
}

.detail-meta {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 14px 0 0;
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
  gap: 4px;
  min-width: 90px;
  color: var(--color-default-500);
}

.meta-row dd {
  margin: 0;
}

.detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 12px;
}

.detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 18px;
}

.detail-body {
  padding: 0 24px 24px;
}

.detail-description-markdown {
  margin-top: 4px;
  line-height: 1.8;
}

.detail-description {
  color: var(--color-default-500);
}

.detail-website {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 10px 0 0;
  color: var(--color-default-500);
  font-size: 13px;
  overflow-wrap: anywhere;
}

.detail-website a {
  color: var(--color-primary);
}

.section-heading {
  margin-bottom: 12px;
}

.volumes-card,
.relations-card,
.contributors-card,
.resource-card {
  margin-top: 16px;
}

.section-head-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.volume-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 14px;
}

.volume-item {
  display: flex;
  gap: 10px;
  padding: 10px;
  border: 1px solid var(--color-content3);
  border-radius: var(--radius-kun-md);
}

.volume-cover {
  width: 70px;
  flex-shrink: 0;
}

.volume-cover :deep(img) {
  width: 70px;
  height: 100px;
  border-radius: var(--radius-kun-sm);
  object-fit: cover;
}

.volume-info {
  min-width: 0;
}

.volume-number {
  color: var(--color-primary);
  font-size: 12px;
  font-weight: 700;
}

.volume-title {
  margin: 2px 0 0;
  font-size: 14px;
  line-height: 1.4;
}

.volume-original-title {
  margin: 2px 0 0;
  color: var(--color-default-500);
  font-size: 12px;
}

.volume-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  margin: 4px 0 0;
  color: var(--color-default-500);
  font-size: 12px;
}

.relation-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 14px;
}

.relation-item {
  overflow: hidden;
  border: 1px solid var(--color-content3);
  border-radius: var(--radius-kun-md);
  color: var(--color-foreground);
  transition: box-shadow var(--kun-dur-fast) var(--ease-kun-standard);
}

.relation-item:hover {
  box-shadow: var(--shadow-kun-md);
}

.relation-cover {
  position: relative;
  aspect-ratio: 3 / 4;
}

.relation-cover :deep(img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.age-badge {
  position: absolute;
  top: 6px;
  right: 6px;
  padding: 1px 6px;
  border-radius: var(--radius-kun-sm);
  background: color-mix(in srgb, var(--color-danger) 88%, transparent);
  color: #fff;
  font-size: 11px;
  font-weight: 700;
}

.relation-info {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px;
}

.relation-title {
  display: -webkit-box;
  overflow: hidden;
  margin: 0;
  font-size: 13px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.resource-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.resource-item {
  padding: 12px;
  border: 1px solid var(--color-content3);
  border-radius: var(--radius-kun-md);
}

.resource-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.resource-title {
  margin: 0;
  font-size: 15px;
}

.resource-description {
  margin: 6px 0 0;
  color: var(--color-default-600);
  font-size: 13px;
  white-space: pre-wrap;
}

.resource-uploader {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  color: var(--color-default-500);
  font-size: 13px;
}

.resource-links {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 8px 0 0;
  padding: 0;
  list-style: none;
}

.resource-links li {
  display: flex;
  align-items: center;
  gap: 8px;
}

.resource-link-url {
  overflow-wrap: anywhere;
  font-size: 13px;
  color: var(--color-primary);
}

.resource-link-open {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  flex-shrink: 0;
  font-size: 13px;
}

.resource-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 8px;
}

.pagination-row {
  display: flex;
  justify-content: center;
  margin-top: 16px;
}

@media (max-width: 640px) {
  .detail-layout {
    grid-template-columns: 1fr;
  }

  .detail-cover {
    width: 100%;
  }

  .detail-cover :deep(img) {
    width: 100%;
    height: auto;
    aspect-ratio: 3 / 4;
  }
}
</style>
