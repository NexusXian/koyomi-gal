<script setup lang="ts">
import type { ChatMessage } from '~/stores/message'

const props = defineProps<{
  conversationId: number
  messages: ChatMessage[]
  currentUserId?: number
}>()

const emit = defineEmits<{
  loadOlder: []
  delete: [messageId: number]
  retry: [clientTempId: string]
}>()

const scrollContainer = ref<HTMLElement>()
const newCount = ref(0)
let pinnedToBottom = true
let untrackedHeight = 0

const loadingOlder = computed(
  () => useMessageStore().olderLoading[props.conversationId]
)

watch(
  () => props.messages.length,
  (count, previous) => {
    if (count <= (previous ?? 0)) return
    const latest = props.messages[count - 1]
    if (pinnedToBottom) {
      scrollToBottom()
      newCount.value = 0
    } else if (latest && latest.sender_id !== props.currentUserId) {
      newCount.value += count - (previous ?? 0)
    }
  }
)

function isMine(message: ChatMessage): boolean {
  return message.sender_id === props.currentUserId
}

function showAvatar(index: number): boolean {
  const next = props.messages[index + 1]
  return !next || next.sender_id !== props.messages[index]?.sender_id
}

function showTime(index: number): boolean {
  const previous = props.messages[index - 1]
  if (!previous) return true
  if (!previous.created_at || !props.messages[index]?.created_at) return true
  const gap =
    new Date(props.messages[index]!.created_at!).getTime() -
    new Date(previous.created_at).getTime()
  return gap > 5 * 60 * 1000
}

function scrollToBottom(): void {
  requestAnimationFrame(() => {
    const element = scrollContainer.value
    if (element) {
      element.scrollTop = element.scrollHeight
    }
  })
}

function handleScroll(): void {
  const element = scrollContainer.value
  if (!element) return
  const distanceToBottom =
    element.scrollHeight - element.scrollTop - element.clientHeight
  const wasPinned = pinnedToBottom
  pinnedToBottom = distanceToBottom < 80
  if (pinnedToBottom) {
    newCount.value = 0
  } else if (wasPinned && !pinnedToBottom) {
    untrackedHeight = element.scrollHeight
  }
  if (element.scrollTop <= 60 && !loadingOlder.value) {
    emit('loadOlder')
  }
}

function restorePosition(): void {
  const element = scrollContainer.value
  if (element && untrackedHeight) {
    element.scrollTop += element.scrollHeight - untrackedHeight
    untrackedHeight = 0
  }
}

watch(loadingOlder, (loading, was) => {
  if (was && !loading) {
    restorePosition()
  }
})

onMounted(() => {
  scrollToBottom()
})

defineExpose({ scrollToBottom })
</script>

<template>
  <div class="message-list-wrap">
    <div
      ref="scrollContainer"
      class="message-list"
      @scroll="handleScroll"
    >
      <div v-if="loadingOlder" class="older-loading">
        <KunIcon name="lucide:loader-circle" class="spin" />
      </div>
      <MessageBubble
        v-for="(item, index) in messages"
        :key="item.id ?? item.clientTempId"
        :message="item"
        :mine="isMine(item)"
        :show-avatar="showAvatar(index)"
        :show-time="showTime(index)"
        @delete="emit('delete', $event)"
        @retry="emit('retry', $event)"
      />
    </div>
    <button
      v-if="newCount > 0"
      type="button"
      class="new-messages"
      @click="scrollToBottom(); newCount = 0"
    >
      ↓ {{ newCount }} 条新消息
    </button>
  </div>
</template>

<style scoped>
.message-list-wrap {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 14px 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.older-loading {
  display: flex;
  justify-content: center;
  padding: 6px;
  color: var(--color-default-400);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.new-messages {
  position: absolute;
  bottom: 14px;
  left: 50%;
  transform: translateX(-50%);
  padding: 6px 14px;
  border: 1px solid var(--app-glass-border);
  border-radius: 999px;
  background: var(--color-content1);
  color: var(--color-primary);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 4px 14px rgb(0 0 0 / 12%);
}
</style>
