<script setup lang="ts">
import type { DtoConversationData } from '~/api/generated/models'

defineProps<{
  conversation: DtoConversationData
  active: boolean
}>()

defineEmits<{ select: [conversationId: number] }>()

const userStore = useUserStore()

function preview(conversation: DtoConversationData): string {
  const last = conversation.last_message
  if (!last) return '还没有消息'
  if (last.is_deleted) return '消息已删除'
  const mine = last.sender_id === userStore.user?.id
  return `${mine ? '我：' : ''}${last.content ?? ''}`
}

function displayTime(value: string | undefined): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const now = new Date()
  const sameDay = date.toDateString() === now.toDateString()
  if (sameDay) {
    return date.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false
    })
  }
  const yesterday = new Date(now)
  yesterday.setDate(now.getDate() - 1)
  if (date.toDateString() === yesterday.toDateString()) {
    return '昨天'
  }
  return date.toLocaleDateString('zh-CN', { month: 'numeric', day: 'numeric' })
}
</script>

<template>
  <button
    type="button"
    class="conversation-item"
    :class="{ active }"
    @click="$emit('select', conversation.id ?? 0)"
  >
    <UserAvatar
      :avatar-url="conversation.user?.avatar_url"
      :display-name="conversation.user?.display_name"
      :username="conversation.user?.username"
      size="md"
    />
    <div class="item-main">
      <div class="item-top">
        <span class="item-name">
          {{ conversation.user?.display_name || conversation.user?.username || '用户' }}
        </span>
        <span class="item-time">{{ displayTime(conversation.last_message_at) }}</span>
      </div>
      <div class="item-bottom">
        <span class="item-preview">{{ preview(conversation) }}</span>
        <span
          v-if="(conversation.unread_count ?? 0) > 0"
          class="item-unread"
        >
          {{ (conversation.unread_count ?? 0) > 99 ? '99+' : conversation.unread_count }}
        </span>
      </div>
    </div>
  </button>
</template>

<style scoped>
.conversation-item {
  display: flex;
  width: 100%;
  gap: 10px;
  padding: 10px 12px;
  border: 0;
  border-radius: var(--radius-kun-md);
  background: transparent;
  text-align: left;
  cursor: pointer;
  transition: background var(--kun-dur-fast) var(--ease-kun-standard);
}

.conversation-item:hover {
  background: color-mix(in srgb, var(--color-primary) 8%, transparent);
}

.conversation-item.active {
  background: color-mix(in srgb, var(--color-primary) 14%, transparent);
}

.item-main {
  min-width: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.item-top,
.item-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.item-name {
  overflow: hidden;
  font-weight: 600;
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-default-800);
}

.item-time {
  flex: 0 0 auto;
  color: var(--color-default-400);
  font-size: 12px;
}

.item-preview {
  overflow: hidden;
  color: var(--color-default-500);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-unread {
  flex: 0 0 auto;
  min-width: 20px;
  padding: 1px 6px;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--color-primary-foreground);
  font-size: 11px;
  font-weight: 700;
  line-height: 18px;
  text-align: center;
}
</style>
