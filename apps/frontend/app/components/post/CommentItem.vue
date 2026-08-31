<script setup lang="ts">
import { message } from 'ant-design-vue'
import {
  deleteComment,
  likeComment,
  unlikeComment,
  updateComment
} from '~/api/generated/comments/comments'
import type { DtoCommentData } from '~/api/generated/models'
import { formatDate } from '~/constants/domain'

const props = defineProps<{
  comment: DtoCommentData
}>()

const emit = defineEmits<{
  refresh: []
  reply: [comment: DtoCommentData]
}>()

const { isAuthenticated } = storeToRefs(useUserStore())
const likePending = ref(false)
const likeState = ref(false)
const editing = ref(false)
const editContent = ref('')

async function toggleLike(): Promise<void> {
  if (!isAuthenticated.value) {
    message.warning('登录后才能点赞')
    return
  }

  likePending.value = true
  try {
    const response = props.comment.id && likeState.value
      ? await unlikeComment(props.comment.id)
      : props.comment.id
        ? await likeComment(props.comment.id)
        : null
    if (response) {
      const data = unwrapApiData(response)
      props.comment.like_count = data.like_count
      likeState.value = Boolean(data.liked)
    }
  } catch (error) {
    message.error(getApiErrorMessage(error, '操作失败'))
  } finally {
    likePending.value = false
  }
}

async function startEdit(): Promise<void> {
  editing.value = true
  editContent.value = props.comment.content ?? ''
}

async function saveEdit(): Promise<void> {
  if (!props.comment.id || !editContent.value.trim()) {
    return
  }

  try {
    await updateComment(props.comment.id, {
      content: editContent.value.trim()
    })
    message.success('评论已更新')
    editing.value = false
    emit('refresh')
  } catch (error) {
    message.error(getApiErrorMessage(error, '更新失败'))
  }
}

async function removeComment(): Promise<void> {
  if (!props.comment.id) {
    return
  }

  try {
    await deleteComment(props.comment.id)
    message.success('评论已删除')
    emit('refresh')
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除失败'))
  }
}
</script>

<template>
  <div class="comment-item">
    <div class="comment-main">
      <span class="comment-avatar" aria-hidden="true">
        <KunIcon name="lucide:user-round" />
      </span>

      <div class="comment-body">
        <div class="comment-head">
          <span class="comment-author">
            用户 #{{ comment.author_id ?? '未知' }}
          </span>
          <span class="comment-time">{{ formatDate(comment.created_at) }}</span>
        </div>

        <template v-if="editing">
          <a-textarea v-model:value="editContent" :rows="3" />
          <div class="edit-actions">
            <a-button size="small" type="primary" @click="saveEdit">
              保存
            </a-button>
            <a-button size="small" @click="editing = false">取消</a-button>
          </div>
        </template>

        <p v-else class="comment-content">{{ comment.content }}</p>

        <div class="comment-actions">
          <button
            type="button"
            class="action-button"
            :class="{ 'action-active': likeState }"
            :disabled="likePending"
            @click="toggleLike"
          >
            <KunIcon name="lucide:thumbs-up" />
            {{ comment.like_count ?? 0 }}
          </button>

          <button type="button" class="action-button" @click="emit('reply', comment)">
            <KunIcon name="lucide:message-circle" />
            回复
          </button>

          <button type="button" class="action-button" @click="startEdit">
            <KunIcon name="lucide:pencil" />
            编辑
          </button>

          <a-popconfirm
            title="确定删除这条评论吗？"
            ok-text="删除"
            cancel-text="取消"
            @confirm="removeComment"
          >
            <button type="button" class="action-button action-danger">
              <KunIcon name="lucide:trash-2" />
              删除
            </button>
          </a-popconfirm>
        </div>
      </div>
    </div>

    <div v-if="comment.replies?.length" class="comment-replies">
      <CommentItem
        v-for="reply in comment.replies"
        :key="reply.id"
        :comment="reply"
        @refresh="emit('refresh')"
        @reply="emit('reply', $event)"
      />
    </div>
  </div>
</template>

<style scoped>
.comment-item {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.comment-main {
  display: flex;
  gap: 12px;
}

.comment-avatar {
  display: grid;
  width: 36px;
  height: 36px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 50%;
  background: color-mix(in srgb, var(--color-primary) 12%, transparent);
  color: var(--color-primary-600);
  font-size: 18px;
}

.comment-body {
  min-width: 0;
  flex: 1;
}

.comment-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.comment-author {
  color: var(--color-foreground);
  font-size: 14px;
  font-weight: 600;
}

.comment-time {
  color: var(--color-default-400);
  font-size: 12px;
}

.comment-content {
  margin: 6px 0 0;
  color: var(--color-default-600);
  font-size: 14px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
}

.edit-actions {
  display: flex;
  gap: 8px;
  margin-top: 8px;
}

.comment-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 8px;
}

.action-button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 8px;
  border: 0;
  border-radius: var(--radius-kun-md);
  background: transparent;
  color: var(--color-default-500);
  font-size: 13px;
  cursor: pointer;
  transition:
    background var(--kun-dur-fast) var(--ease-kun-standard),
    color var(--kun-dur-fast) var(--ease-kun-standard);
}

.action-button:hover {
  background: color-mix(in srgb, var(--color-primary) 10%, transparent);
  color: var(--color-primary-600);
}

.action-active {
  color: var(--color-primary-600);
}

.action-danger:hover {
  background: color-mix(in srgb, var(--color-danger) 10%, transparent);
  color: var(--color-danger);
}

.action-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.comment-replies {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-left: 24px;
  padding-left: 16px;
  border-left: 2px solid var(--color-default-200);
}
</style>
