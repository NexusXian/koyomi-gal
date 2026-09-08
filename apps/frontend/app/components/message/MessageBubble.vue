<script setup lang="ts">
import type { ChatMessage } from '~/stores/message'

const props = withDefaults(
  defineProps<{
    message: ChatMessage
    mine: boolean
    showAvatar: boolean
    showTime: boolean
  }>(),
  { showAvatar: true, showTime: false }
)

const emit = defineEmits<{
  delete: [messageId: number]
  retry: [clientTempId: string]
}>()

function formatTime(value: string | undefined): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  })
}

function remove(): void {
  if (props.message.id) {
    emit('delete', props.message.id)
  }
}

function retry(): void {
  if (props.message.clientTempId) {
    emit('retry', props.message.clientTempId)
  }
}
</script>

<template>
  <div class="message-row" :class="{ mine }">
    <UserAvatar
      v-if="!mine && showAvatar"
      class="bubble-avatar"
      :avatar-url="message.sender?.avatar_url"
      :display-name="message.sender?.display_name"
      :username="message.sender?.username"
      size="sm"
    />
    <span v-else-if="!mine" class="bubble-avatar-spacer" />
    <div class="bubble-main">
      <div class="bubble-body">
        <span v-if="message.is_deleted" class="bubble-deleted">该消息已删除</span>
        <template v-else>
          <span class="bubble-content">{{ message.content }}</span>
          <span
            v-if="message.status === 'sending'"
            class="bubble-status"
            aria-label="发送中"
          >
            <KunIcon name="lucide:clock" />
          </span>
          <span
            v-else-if="message.status === 'failed'"
            class="bubble-status failed"
            aria-label="发送失败"
          >
            <KunIcon name="lucide:circle-alert" />
          </span>
        </template>
      </div>
      <div v-if="showTime || message.status === 'failed'" class="bubble-meta">
        <span class="bubble-time">{{ formatTime(message.created_at) }}</span>
        <button
          v-if="message.status === 'failed'"
          type="button"
          class="bubble-retry"
          @click="retry"
        >
          点击重试
        </button>
        <a-dropdown v-if="mine && message.id && !message.is_deleted">
          <button type="button" class="bubble-more" aria-label="消息操作">
            <KunIcon name="lucide:ellipsis" />
          </button>
          <template #overlay>
            <a-menu @click="remove">
              <a-menu-item key="delete" danger>
                <span class="bubble-menu-item">
                  <KunIcon name="lucide:trash-2" />删除消息
                </span>
              </a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
      </div>
    </div>
  </div>
</template>

<style scoped>
.message-row {
  display: flex;
  gap: 8px;
  padding: 1px 12px;
}

.message-row.mine {
  flex-direction: row-reverse;
}

.bubble-avatar,
.bubble-avatar-spacer {
  flex: 0 0 auto;
}

.bubble-avatar-spacer {
  width: 32px;
}

.bubble-main {
  max-width: min(72%, 480px);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.message-row.mine .bubble-main {
  align-items: flex-end;
}

.bubble-body {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 8px 12px;
  border-radius: 14px;
  background: var(--color-content2);
  font-size: 14px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.message-row.mine .bubble-body {
  background: var(--color-primary);
  color: var(--color-primary-foreground);
  border-bottom-right-radius: 4px;
}

.message-row:not(.mine) .bubble-body {
  border-bottom-left-radius: 4px;
}

.bubble-content {
  white-space: pre-wrap;
}

.bubble-deleted {
  color: var(--color-default-400);
  font-style: italic;
}

.message-row.mine .bubble-deleted {
  color: color-mix(in srgb, var(--color-primary-foreground) 70%, transparent);
}

.bubble-status {
  display: inline-flex;
  opacity: 0.7;
  font-size: 12px;
}

.bubble-status.failed {
  color: var(--color-danger, #ef4444);
  opacity: 1;
}

.bubble-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 4px;
}

.bubble-time {
  color: var(--color-default-400);
  font-size: 11px;
}

.bubble-retry {
  border: 0;
  padding: 0;
  background: transparent;
  color: var(--color-danger, #ef4444);
  font-size: 11px;
  cursor: pointer;
}

.bubble-more {
  display: inline-flex;
  border: 0;
  padding: 0 2px;
  background: transparent;
  color: var(--color-default-400);
  cursor: pointer;
}

.bubble-menu-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
</style>
