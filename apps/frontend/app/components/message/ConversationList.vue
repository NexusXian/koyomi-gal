<script setup lang="ts">
const messageStore = useMessageStore()
const router = useRouter()

const conversations = computed(() => messageStore.conversations)
const loading = computed(() => messageStore.conversationsLoading)
const error = computed(() => messageStore.conversationsError)
const hasMore = computed(() => messageStore.hasMoreConversations)
const currentId = computed(() => messageStore.currentConversationId)

function select(conversationId: number): void {
  if (!conversationId) return
  void router.push(`/messages/${conversationId}`)
}

function loadMore(): void {
  void messageStore.loadMoreConversations()
}

function retry(): void {
  void messageStore.fetchConversations(true)
}
</script>

<template>
  <div class="conversation-list">
    <div v-if="loading && conversations.length === 0" class="list-skeleton">
      <KunSkeleton v-for="index in 6" :key="index" class="skeleton-row" />
    </div>

    <div v-else-if="error && conversations.length === 0" class="list-error">
      <a-alert type="error" :message="error" show-icon>
        <template #action>
          <KunButton size="sm" color="danger" variant="light" @click="retry">
            重试
          </KunButton>
        </template>
      </a-alert>
    </div>

    <KunNull
      v-else-if="conversations.length === 0"
      text="还没有私信，去用户主页打个招呼吧"
    />

    <template v-else>
      <ConversationItem
        v-for="conversation in conversations"
        :key="conversation.id"
        :conversation="conversation"
        :active="conversation.id === currentId"
        @select="select"
      />
      <div v-if="hasMore" class="list-more">
        <KunButton
          color="primary"
          variant="light"
          size="sm"
          :disabled="loading"
          @click="loadMore"
        >
          {{ loading ? '加载中...' : '加载更多' }}
        </KunButton>
      </div>
    </template>
  </div>
</template>

<style scoped>
.conversation-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
  height: 100%;
  padding: 8px;
}

.list-skeleton {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 8px;
}

:deep(.skeleton-row) {
  height: 48px;
  border-radius: var(--radius-kun-md);
}

.list-error {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
}

.list-more {
  display: flex;
  justify-content: center;
  padding: 8px 0 4px;
}
</style>
