<script setup lang="ts">
import { message } from 'ant-design-vue'
import {
  deletePost,
  favoritePost,
  getPost,
  likePost,
  unfavoritePost,
  unlikePost
} from '~/api/generated/posts/posts'
import {
  createComment,
  listPostComments
} from '~/api/generated/comments/comments'
import type { DtoCommentData, DtoCommentListData, DtoPostData } from '~/api/generated/models'
import { formatDate } from '~/constants/domain'

const route = useRoute()
const router = useRouter()

const postId = computed(() => Number(route.params.id))

const { data: post, error } = await useAsyncData<DtoPostData, Error>(
  `post-${postId.value}`,
  async () => unwrapApiData(await getPost(postId.value), '加载帖子失败')
)

if (error.value || !post.value) {
  throw createError({
    statusCode: 404,
    statusMessage: '帖子不存在',
    fatal: true
  })
}

useSeoMeta({
  title: () => `${post.value?.title ?? '帖子'} - Koyomi`,
  description: () => post.value?.content?.slice(0, 120) ?? '帖子详情'
})

const userStore = useUserStore()
const { isAuthenticated } = storeToRefs(userStore)

const comments = ref<DtoCommentData[]>([])
const commentTotal = ref(0)
const commentPage = ref(1)
const commentLimit = 20
const commentsLoading = ref(false)

const likeState = ref(false)
const likeCount = ref(0)
const favoriteState = ref(false)
const favoriteCount = ref(0)
const actionPending = ref(false)

const commentContent = ref('')
const replyTarget = ref<DtoCommentData | null>(null)
const commentSubmitting = ref(false)

const commentTotalPage = computed(() =>
  Math.max(1, Math.ceil(commentTotal.value / commentLimit))
)

async function loadPostActions(): Promise<void> {
  likeCount.value = post.value?.like_count ?? 0
  favoriteCount.value = post.value?.favorite_count ?? 0
}

async function loadComments(): Promise<void> {
  commentsLoading.value = true
  try {
    const data = unwrapApiData<DtoCommentListData>(
      await listPostComments(postId.value, {
        page: commentPage.value,
        limit: commentLimit
      })
    )
    comments.value = data.items ?? []
    commentTotal.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '加载评论失败'))
  } finally {
    commentsLoading.value = false
  }
}

function updateCommentPage(next: number): void {
  commentPage.value = next
  void loadComments()
}

async function toggleLike(): Promise<void> {
  if (!isAuthenticated.value) {
    message.warning('登录后才能点赞')
    return
  }

  actionPending.value = true
  try {
    const response = likeState.value
      ? await unlikePost(postId.value)
      : await likePost(postId.value)
    const data = unwrapApiData(response)
    likeState.value = Boolean(data.liked)
    likeCount.value = data.like_count ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '操作失败'))
  } finally {
    actionPending.value = false
  }
}

async function toggleFavorite(): Promise<void> {
  if (!isAuthenticated.value) {
    message.warning('登录后才能收藏')
    return
  }

  actionPending.value = true
  try {
    const response = favoriteState.value
      ? await unfavoritePost(postId.value)
      : await favoritePost(postId.value)
    const data = unwrapApiData(response)
    favoriteState.value = Boolean(data.favorited)
    favoriteCount.value = data.favorite_count ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '操作失败'))
  } finally {
    actionPending.value = false
  }
}

async function submitComment(): Promise<void> {
  const content = commentContent.value.trim()
  if (!content) {
    return
  }

  commentSubmitting.value = true
  try {
    await createComment(postId.value, {
      content,
      parent_id: replyTarget.value?.id
    })
    message.success('评论已发布')
    commentContent.value = ''
    replyTarget.value = null
    await loadComments()
  } catch (error) {
    message.error(getApiErrorMessage(error, '发布评论失败'))
  } finally {
    commentSubmitting.value = false
  }
}

function setReply(target: DtoCommentData): void {
  replyTarget.value = target
}

async function removePost(): Promise<void> {
  try {
    await deletePost(postId.value)
    message.success('帖子已删除')
    void router.push('/posts')
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除失败'))
  }
}

onMounted(() => {
  void loadPostActions()
  void loadComments()
})
</script>

<template>
  <AppPageContainer>
    <nav class="breadcrumb" aria-label="面包屑">
      <NuxtLink to="/posts">帖子</NuxtLink>
      <KunIcon name="lucide:chevron-right" />
      <span>{{ post?.title }}</span>
    </nav>

    <KunCard padding="lg" class-name="post-card">
      <div class="post-head">
        <h1>{{ post?.title }}</h1>
        <div class="post-meta">
          <span>
            <KunIcon name="lucide:user-round" />
            用户 #{{ post?.author_id ?? '未知' }}
          </span>
          <span>
            <KunIcon name="lucide:calendar" />
            {{ formatDate(post?.created_at) }}
          </span>
          <span v-if="post?.galgame_id">
            <KunIcon name="lucide:gamepad-2" />
            <NuxtLink :to="`/galgames/${post.galgame_id}`">
              Galgame #{{ post.galgame_id }}
            </NuxtLink>
          </span>
        </div>
      </div>

      <p class="post-content">{{ post?.content }}</p>

      <div class="post-actions">
        <button
          type="button"
          class="pill-button"
          :class="{ 'pill-active': likeState }"
          :disabled="actionPending"
          @click="toggleLike"
        >
          <KunIcon name="lucide:thumbs-up" />
          {{ likeCount }}
        </button>

        <button
          type="button"
          class="pill-button"
          :class="{ 'pill-active': favoriteState }"
          :disabled="actionPending"
          @click="toggleFavorite"
        >
          <KunIcon name="lucide:heart" />
          {{ favoriteCount }}
        </button>

        <NuxtLink
          v-if="isAuthenticated"
          class="pill-button"
          :to="`/posts/${postId}/edit`"
        >
          <KunIcon name="lucide:pencil" />
          编辑
        </NuxtLink>

        <a-popconfirm
          v-if="isAuthenticated"
          title="确定删除这篇帖子吗？"
          ok-text="删除"
          cancel-text="取消"
          @confirm="removePost"
        >
          <button type="button" class="pill-button pill-danger">
            <KunIcon name="lucide:trash-2" />
            删除
          </button>
        </a-popconfirm>
      </div>
    </KunCard>

    <KunCard padding="lg" class-name="comment-card">
      <KunHeader
        name="评论"
        :description="`共 ${commentTotal} 条`"
        scale="h3"
      />

      <div v-if="isAuthenticated" class="comment-editor">
        <p v-if="replyTarget" class="reply-hint">
          回复 #{{ replyTarget.id }}：
          <span class="reply-text">
            {{ (replyTarget.content ?? '').slice(0, 40) }}
          </span>
          <a-button type="link" size="small" @click="replyTarget = null">
            取消回复
          </a-button>
        </p>
        <a-textarea
          v-model:value="commentContent"
          :rows="3"
          :maxlength="10000"
          placeholder="友善发言，理性讨论"
        />
        <div class="editor-actions">
          <a-button
            type="primary"
            :loading="commentSubmitting"
            :disabled="!commentContent.trim()"
            @click="submitComment"
          >
            {{ replyTarget ? '发布回复' : '发布评论' }}
          </a-button>
        </div>
      </div>
      <p v-else class="login-hint">
        <NuxtLink to="/login">登录</NuxtLink>
        后参与评论
      </p>

      <a-spin :spinning="commentsLoading">
        <KunNull
          v-if="comments.length === 0"
          message="还没有评论，来说两句吧"
        />

        <div v-else class="comment-list">
          <CommentItem
            v-for="comment in comments"
            :key="comment.id"
            :comment="comment"
            @refresh="loadComments"
            @reply="setReply"
          />
        </div>
      </a-spin>

      <div v-if="commentTotalPage > 1" class="pagination-row">
        <KunPagination
          :current-page="commentPage"
          :total-page="commentTotalPage"
          :is-loading="commentsLoading"
          @update:current-page="updateCommentPage"
        />
      </div>
    </KunCard>
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

.post-card,
.comment-card {
  margin-bottom: 18px;
}

.post-head h1 {
  margin: 0;
  font-size: clamp(22px, 3vw, 28px);
  font-weight: 800;
  letter-spacing: -0.02em;
}

.post-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px;
  margin-top: 10px;
  color: var(--color-default-400);
  font-size: 13px;
}

.post-meta span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.post-meta a {
  color: var(--color-primary);
}

.post-content {
  margin: 20px 0 0;
  color: var(--color-default-600);
  font-size: 15px;
  line-height: 1.9;
  white-space: pre-wrap;
  word-break: break-word;
}

.post-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid var(--color-default-200);
}

.pill-button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border: 1px solid var(--color-default-200);
  border-radius: 999px;
  background: transparent;
  color: var(--color-default-600);
  font-size: 14px;
  cursor: pointer;
  transition:
    background var(--kun-dur-fast) var(--ease-kun-standard),
    color var(--kun-dur-fast) var(--ease-kun-standard),
    border-color var(--kun-dur-fast) var(--ease-kun-standard);
}

.pill-button:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.pill-active {
  border-color: var(--color-primary);
  background: color-mix(in srgb, var(--color-primary) 12%, transparent);
  color: var(--color-primary-600);
}

.pill-danger:hover {
  border-color: var(--color-danger);
  color: var(--color-danger);
}

.pill-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.comment-editor {
  margin: 14px 0 20px;
}

.reply-hint {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin: 0 0 8px;
  color: var(--color-default-500);
  font-size: 13px;
}

.reply-text {
  color: var(--color-primary-600);
}

.editor-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

.login-hint {
  margin: 14px 0 0;
  color: var(--color-default-500);
}

.login-hint a {
  color: var(--color-primary);
}

.comment-list {
  display: flex;
  flex-direction: column;
  gap: 18px;
  margin-top: 8px;
}

.pagination-row {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}
</style>
