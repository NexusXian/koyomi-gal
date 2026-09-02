<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  listAdminComments,
  listAdminPosts
} from '~/api/generated/admin/admin'
import { deleteComment } from '~/api/generated/comments/comments'
import { deletePost } from '~/api/generated/posts/posts'
import type {
  DtoAdminCommentData,
  DtoAdminPostData
} from '~/api/generated/models'
import { formatDate } from '~/constants/domain'
import { normalizeEditorMode } from '~/types/post'
import { stripMarkdownForExcerpt } from '~/utils/markdown'

useSeoMeta({ title: '社区管理 - Koyomi' })

const { has, load: loadPermissions } = usePermissions()
const canModeratePosts = computed(() => has('post:moderate'))
const canModerateComments = computed(() => has('comment:moderate'))
const activeTab = ref<'posts' | 'comments'>('posts')
const limit = 20

const postQuery = reactive({ keyword: '', page: 1 })
const posts = ref<DtoAdminPostData[]>([])
const postTotal = ref(0)
const postsLoading = ref(false)
const deletingPostId = ref<number | null>(null)

const commentQuery = reactive({ keyword: '', page: 1 })
const comments = ref<DtoAdminCommentData[]>([])
const commentTotal = ref(0)
const commentsLoading = ref(false)
const deletingCommentId = ref<number | null>(null)

const postColumns: TableColumnsType = [
  { title: 'ID', dataIndex: 'id', width: 70 },
  { title: '帖子', key: 'post', ellipsis: true },
  { title: '内容摘要', key: 'excerpt', ellipsis: true },
  { title: '作者', key: 'author', width: 140, ellipsis: true },
  { title: '评论', dataIndex: 'comment_count', width: 75 },
  { title: '发布时间', dataIndex: 'created_at', width: 170 },
  { title: '操作', key: 'actions', width: 190 }
]

const commentColumns: TableColumnsType = [
  { title: 'ID', dataIndex: 'id', width: 70 },
  { title: '评论内容', key: 'content', ellipsis: true },
  { title: '作者', key: 'author', width: 140, ellipsis: true },
  { title: '所属帖子', key: 'post', ellipsis: true },
  { title: '发布时间', dataIndex: 'created_at', width: 170 },
  { title: '操作', key: 'actions', width: 120 }
]

function plainExcerpt(content: string, length = 120): string {
  const normalized = content.replace(/\s+/g, ' ').trim()
  return normalized.length > length
    ? `${normalized.slice(0, length)}…`
    : normalized
}

function postExcerpt(post: DtoAdminPostData): string {
  const content = post.content ?? ''
  return normalizeEditorMode(post.editor_mode) === 'markdown'
    ? stripMarkdownForExcerpt(content, 120)
    : plainExcerpt(content)
}

async function loadPosts(): Promise<void> {
  if (!canModeratePosts.value) {
    return
  }

  postsLoading.value = true
  try {
    const data = unwrapApiData(
      await listAdminPosts({
        keyword: postQuery.keyword.trim() || undefined,
        page: postQuery.page,
        limit
      })
    )
    posts.value = data.items ?? []
    postTotal.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '帖子列表加载失败'))
  } finally {
    postsLoading.value = false
  }
}

async function loadComments(): Promise<void> {
  if (!canModerateComments.value) {
    return
  }

  commentsLoading.value = true
  try {
    const data = unwrapApiData(
      await listAdminComments({
        keyword: commentQuery.keyword.trim() || undefined,
        page: commentQuery.page,
        limit
      })
    )
    comments.value = data.items ?? []
    commentTotal.value = data.total ?? 0
  } catch (error) {
    message.error(getApiErrorMessage(error, '评论列表加载失败'))
  } finally {
    commentsLoading.value = false
  }
}

onMounted(async () => {
  await loadPermissions()
  if (canModeratePosts.value) {
    activeTab.value = 'posts'
    await loadPosts()
  } else if (canModerateComments.value) {
    activeTab.value = 'comments'
    await loadComments()
  }
})

function changeTab(tab: string | number): void {
  if (tab === 'posts' && canModeratePosts.value) {
    void loadPosts()
  } else if (tab === 'comments' && canModerateComments.value) {
    void loadComments()
  }
}

function searchPosts(): void {
  postQuery.page = 1
  void loadPosts()
}

function searchComments(): void {
  commentQuery.page = 1
  void loadComments()
}

function changePostPage(next: number): void {
  postQuery.page = next
  void loadPosts()
}

function changeCommentPage(next: number): void {
  commentQuery.page = next
  void loadComments()
}

async function removePost(post: DtoAdminPostData): Promise<void> {
  if (!post.id || !canModeratePosts.value) {
    return
  }

  deletingPostId.value = post.id
  try {
    await deletePost(post.id)
    message.success('帖子已删除')
    if (posts.value.length === 1 && postQuery.page > 1) {
      postQuery.page -= 1
    }
    await loadPosts()
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除帖子失败'))
  } finally {
    deletingPostId.value = null
  }
}

async function removeComment(comment: DtoAdminCommentData): Promise<void> {
  if (!comment.id || !canModerateComments.value) {
    return
  }

  deletingCommentId.value = comment.id
  try {
    await deleteComment(comment.id)
    message.success('评论已删除')
    if (comments.value.length === 1 && commentQuery.page > 1) {
      commentQuery.page -= 1
    }
    await loadComments()
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除评论失败'))
  } finally {
    deletingCommentId.value = null
  }
}
</script>

<template>
  <div>
    <KunHeader
      name="社区管理"
      description="搜索并管理最新发布的帖子与评论。"
      scale="h3"
      class="page-heading"
    />

    <a-tabs v-if="canModeratePosts || canModerateComments" v-model:active-key="activeTab" @change="changeTab">
      <a-tab-pane v-if="canModeratePosts" key="posts" tab="帖子">
        <KunCard padding="md" class-name="filter-card">
          <a-input-search
            v-model:value="postQuery.keyword"
            placeholder="搜索标题、内容、作者或 Galgame"
            allow-clear
            enter-button="搜索"
            @search="searchPosts"
          />
        </KunCard>

        <a-table
          :columns="postColumns"
          :data-source="posts"
          :loading="postsLoading"
          :pagination="{
            current: postQuery.page,
            pageSize: limit,
            total: postTotal,
            showSizeChanger: false,
            showTotal: (count: number) => `共 ${count} 条`
          }"
          row-key="id"
          :scroll="{ x: 1050 }"
          @change="(pagination: { current?: number }) => changePostPage(pagination.current ?? 1)"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'post'">
              <NuxtLink :to="`/posts/${record.id}`" class="primary-link">
                {{ record.title || `帖子 #${record.id}` }}
              </NuxtLink>
              <div v-if="record.galgame_id" class="secondary-text">
                {{ record.galgame_title || `Galgame #${record.galgame_id}` }}
              </div>
            </template>
            <template v-else-if="column.key === 'excerpt'">
              {{ postExcerpt(record) || '-' }}
            </template>
            <template v-else-if="column.key === 'author'">
              {{ record.author_name || (record.author_id ? `用户 #${record.author_id}` : '-') }}
            </template>
            <template v-else-if="column.dataIndex === 'created_at'">
              {{ formatDate(record.created_at) }}
            </template>
            <template v-else-if="column.key === 'actions'">
              <div class="table-actions">
                <a-button size="small" :href="`/posts/${record.id}`">查看</a-button>
                <a-button size="small" :href="`/posts/${record.id}/edit`">编辑</a-button>
                <a-popconfirm
                  :title="`确定删除帖子「${record.title || record.id}」吗？其评论也会被删除。`"
                  ok-text="删除"
                  cancel-text="取消"
                  @confirm="removePost(record)"
                >
                  <a-button size="small" danger :loading="deletingPostId === record.id">删除</a-button>
                </a-popconfirm>
              </div>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane v-if="canModerateComments" key="comments" tab="评论">
        <KunCard padding="md" class-name="filter-card">
          <a-input-search
            v-model:value="commentQuery.keyword"
            placeholder="搜索评论内容、作者或帖子标题"
            allow-clear
            enter-button="搜索"
            @search="searchComments"
          />
        </KunCard>

        <a-table
          :columns="commentColumns"
          :data-source="comments"
          :loading="commentsLoading"
          :pagination="{
            current: commentQuery.page,
            pageSize: limit,
            total: commentTotal,
            showSizeChanger: false,
            showTotal: (count: number) => `共 ${count} 条`
          }"
          row-key="id"
          :scroll="{ x: 900 }"
          @change="(pagination: { current?: number }) => changeCommentPage(pagination.current ?? 1)"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'content'">
              {{ plainExcerpt(record.content ?? '') || '-' }}
            </template>
            <template v-else-if="column.key === 'author'">
              {{ record.author_name || (record.author_id ? `用户 #${record.author_id}` : '-') }}
            </template>
            <template v-else-if="column.key === 'post'">
              <NuxtLink :to="`/posts/${record.post_id}`" class="primary-link">
                {{ record.post_title || `帖子 #${record.post_id}` }}
              </NuxtLink>
            </template>
            <template v-else-if="column.dataIndex === 'created_at'">
              {{ formatDate(record.created_at) }}
            </template>
            <template v-else-if="column.key === 'actions'">
              <div class="table-actions">
                <a-button size="small" :href="`/posts/${record.post_id}`">查看</a-button>
                <a-popconfirm
                  title="确定删除该评论及其回复吗？"
                  ok-text="删除"
                  cancel-text="取消"
                  @confirm="removeComment(record)"
                >
                  <a-button size="small" danger :loading="deletingCommentId === record.id">删除</a-button>
                </a-popconfirm>
              </div>
            </template>
          </template>
        </a-table>
      </a-tab-pane>
    </a-tabs>

    <KunNull v-else message="当前账号没有社区管理权限。" />
  </div>
</template>

<style scoped>
.page-heading,
.filter-card {
  margin-bottom: 16px;
}

.primary-link {
  color: var(--color-primary);
  font-weight: 600;
}

.secondary-text {
  margin-top: 3px;
  color: var(--color-default-400);
  font-size: 12px;
}

.table-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
