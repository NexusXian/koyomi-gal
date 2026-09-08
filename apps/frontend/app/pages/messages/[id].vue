<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { message as antMessage } from 'ant-design-vue'

useSeoMeta({ title: '私信会话 - Koyomi' })

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { user, isAuthenticated } = storeToRefs(userStore)
const messageStore = useMessageStore()

const conversationId = computed(() => Number(route.params.id))
const conversation = computed(() =>
  messageStore.conversations.find((item) => item.id === conversationId.value)
)
const messages = computed(
  () => messageStore.messages[conversationId.value] ?? []
)
const loading = computed(
  () => messageStore.messagesLoading[conversationId.value]
)
const blocked = computed(() => Boolean(conversation.value?.is_blocked))
const canSend = computed(() => Boolean(conversation.value?.can_send))

watch(
  () => [userStore.getInitialized, isAuthenticated.value] as const,
  ([initialized, authenticated]) => {
    if (!initialized) return
    if (!authenticated) {
      void router.replace('/login')
    }
  },
  { immediate: true }
)

async function load(): Promise<void> {
  if (!Number.isFinite(conversationId.value) || conversationId.value <= 0) {
    return
  }
  await messageStore.openConversation(conversationId.value)
  if (!messageStore.conversations.some((item) => item.id === conversationId.value)) {
    await messageStore.fetchConversations(true)
  }
  if (!messageStore.conversations.some((item) => item.id === conversationId.value)) {
    const peerMessage = messages.value.find(
      (item) => item.id && item.sender_id && item.sender_id !== user.value?.id
    )
    const mineMessage = messages.value.find(
      (item) => item.id && item.sender_id === user.value?.id
    )
    const peerId =
      peerMessage?.sender_id ??
      (mineMessage?.receiver_id ? Number(mineMessage.receiver_id) : undefined)
    if (peerId) {
      try {
        await messageStore.startConversationWith(peerId)
      } catch {
        // 无权限时保持最小展示
      }
    }
  }
}

watch(
  conversationId,
  () => {
    void load()
  },
  { immediate: true }
)

onUnmounted(() => {
  messageStore.closeConversation()
})

function loadOlder(): void {
  const oldest = messages.value.find((item) => item.id)
  if (oldest?.id) {
    void messageStore.fetchMessages(conversationId.value, oldest.id)
  }
}

async function removeMessage(messageId: number): Promise<void> {
  try {
    await messageStore.deleteMessage(conversationId.value, messageId)
    antMessage.success('消息已删除')
  } catch (error) {
    antMessage.error(getApiErrorMessage(error, '删除消息失败'))
  }
}

async function retry(clientTempId: string): Promise<void> {
  try {
    await messageStore.retryMessage(conversationId.value, clientTempId)
  } catch (error) {
    antMessage.error(getApiErrorMessage(error, '重试失败'))
  }
}

function refreshConversation(): void {
  void messageStore.fetchConversations(true)
}

function back(): void {
  void router.replace('/messages')
}
</script>

<template>
  <div class="thread-page">
    <template v-if="conversation">
      <div class="thread-top">
        <KunButton
          class="thread-back"
          color="default"
          variant="light"
          size="sm"
          rounded="full"
          :is-icon-only="true"
          aria-label="返回会话列表"
          @click="back"
        >
          <KunIcon name="lucide:arrow-left" />
        </KunButton>
        <MessageHeader :conversation="conversation" @refresh="refreshConversation" />
      </div>
      <MessageList
        :conversation-id="conversationId"
        :messages="messages"
        :current-user-id="user?.id"
        @load-older="loadOlder"
        @delete="removeMessage"
        @retry="retry"
      />
      <MessageComposer
        :conversation-id="conversationId"
        :disabled="blocked || !canSend"
        :disabled-hint="blocked ? '你已拉黑该用户，取消拉黑后可继续发送' : '当前无法向该用户发送私信'"
      />
    </template>

    <div v-else-if="loading" class="thread-loading">
      <KunSkeleton class="thread-skeleton" />
      <KunSkeleton class="thread-skeleton" />
      <KunSkeleton class="thread-skeleton" />
    </div>

    <div v-else class="thread-missing">
      <p>该会话不存在或你已无法访问</p>
      <KunButton color="primary" variant="light" size="sm" @click="back">
        返回会话列表
      </KunButton>
    </div>
  </div>
</template>

<style scoped>
.thread-page {
  display: flex;
  flex-direction: column;
  width: 100%;
  min-height: 0;
  height: 100%;
}

.thread-top {
  display: flex;
  align-items: center;
  gap: 4px;
}

.thread-top :deep(.message-header) {
  flex: 1;
  min-width: 0;
}

.thread-back {
  display: none;
}

@media (max-width: 1023px) {
  .thread-back {
    display: inline-flex;
    margin-left: 8px;
  }
}

.thread-loading {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px;
}

:deep(.thread-skeleton) {
  height: 52px;
  border-radius: var(--radius-kun-md);
}

.thread-missing {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--color-default-500);
}

.thread-missing p {
  margin: 0;
  font-size: 14px;
}
</style>
