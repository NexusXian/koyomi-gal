<script setup lang="ts">
import { listPosts } from '~/api/generated/posts/posts'
import { getGalgame } from '~/api/generated/galgames/galgames'
import type { DtoPostData, DtoPostListData } from '~/api/generated/models'
import { normalizeEditorMode } from '~/types/post'
import { stripMarkdownForExcerpt } from '~/utils/markdown'
import { formatDate } from '~/constants/domain'

useSeoMeta({
  title: '帖子 - Koyomi',
  description: '浏览社区帖子'
})

const route = useRoute()
const router = useRouter()

const galgameId = ref<number | undefined>(
  route.query.galgame_id ? Number(route.query.galgame_id) : undefined
)
const page = ref(Math.max(1, Number(route.query.page ?? 1) || 1))

const limit = 20

const { data: listData, pending } = await useAsyncData<
  DtoPostListData,
  Error
>(
  'posts-list',
  async () =>
    unwrapApiData(
      await listPosts({
        galgame_id: galgameId.value,
        page: page.value,
        limit
      }),
      '查询帖子失败'
    ),
  { watch: [galgameId, page] }
)

const items = computed(() => listData.value?.items ?? [])
const total = computed(() => listData.value?.total ?? 0)
const totalPage = computed(() => Math.max(1, Math.ceil(total.value / limit)))

// List cards show a plain-text excerpt only; full markdown rendering happens
// on the detail page.
function postExcerpt(post: DtoPostData): string {
  const content = post.content ?? ''
  return normalizeEditorMode(post.editor_mode) === 'markdown'
    ? stripMarkdownForExcerpt(content)
    : content
}

const galgameTitle = ref('')

async function loadGalgameTitle(): Promise<void> {
  if (!galgameId.value) {
    galgameTitle.value = ''
    return
  }
  try {
    const data = unwrapApiData(await getGalgame(galgameId.value))
    galgameTitle.value = data.title ?? ''
  } catch {
    galgameTitle.value = ''
  }
}

watch(galgameId, () => void loadGalgameTitle(), { immediate: true })

const { isAuthenticated } = storeToRefs(useUserStore())

watch([galgameId, page], () => {
  void router.replace({
    query: {
      ...route.query,
      galgame_id: galgameId.value ?? undefined,
      page: page.value > 1 ? page.value : undefined
    }
  })
})

function updatePage(next: number): void {
  page.value = next
}
</script>

<template>
  <AppPageContainer
    title="帖子"
    :description="
      galgameId
        ? '该 Galgame 下的社区讨论。'
        : '浏览社区帖子，参与讨论。'
    "
  >
    <template #actions>
      <KunButton
        v-if="isAuthenticated"
        color="primary"
        :href="
          galgameId ? `/posts/new?galgame_id=${galgameId}` : '/posts/new'
        "
      >
        <KunIcon name="lucide:plus" />
        发帖
      </KunButton>
    </template>

    <div v-if="galgameId" class="filter-row">
      <a-tag closable @close.prevent="galgameId = undefined">
        仅显示 {{ galgameTitle || `Galgame #${galgameId}` }} 的帖子
      </a-tag>
    </div>

    <a-alert
      v-if="!pending && items.length === 0"
      type="info"
      show-icon
      message="还没有帖子"
      description="成为第一个发帖的人吧。"
    />

    <div v-else class="post-list">
      <KunSkeleton v-if="pending" class="post-skeleton" />

      <KunCard
        v-for="post in items"
        v-else
        :key="post.id"
        padding="md"
        :is-hoverable="true"
      >
        <NuxtLink :to="`/posts/${post.id}`" class="post-link">
          <h3 class="post-title">{{ post.title }}</h3>
          <p class="post-excerpt">{{ postExcerpt(post) }}</p>
          <div class="post-meta">
            <span class="post-author">
              <KunIcon name="lucide:user-round" />
              {{ post.author_name || (post.author_id ? `用户 #${post.author_id}` : '未知') }}
            </span>
            <span class="post-item">
              <KunIcon name="lucide:calendar" />
              {{ formatDate(post.created_at) }}
            </span>
            <span class="post-item">
              <KunIcon name="lucide:message-circle" />
              {{ post.comment_count ?? 0 }}
            </span>
            <span class="post-item">
              <KunIcon name="lucide:thumbs-up" />
              {{ post.like_count ?? 0 }}
            </span>
            <span class="post-item">
              <KunIcon name="lucide:heart" />
              {{ post.favorite_count ?? 0 }}
            </span>
          </div>
        </NuxtLink>
      </KunCard>
    </div>

    <div v-if="totalPage > 1" class="pagination-row">
      <KunPagination
        :current-page="page"
        :total-page="totalPage"
        :is-loading="pending"
        @update:current-page="updatePage"
      />
    </div>
  </AppPageContainer>
</template>

<style scoped>
.filter-row {
  margin-bottom: 14px;
}

.post-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.post-skeleton {
  min-height: 120px;
}

.post-link {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.post-title {
  margin: 0;
  color: var(--color-foreground);
  font-size: 17px;
  font-weight: 700;
}

.post-title:hover {
  color: var(--color-primary);
}

.post-excerpt {
  display: -webkit-box;
  overflow: hidden;
  margin: 0;
  color: var(--color-default-500);
  font-size: 14px;
  line-height: 1.7;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.post-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px;
  color: var(--color-default-400);
  font-size: 13px;
}

.post-author,
.post-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.pagination-row {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}
</style>
