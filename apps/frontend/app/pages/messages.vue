<script setup lang="ts">
import { storeToRefs } from 'pinia'

useSeoMeta({
  title: '私信 - Koyomi',
  description: '与其他用户进行一对一私信交流'
})

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { isAuthenticated } = storeToRefs(userStore)
const messageStore = useMessageStore()

const hasThread = computed(() => route.path !== '/messages')

watch(
  () => [userStore.getInitialized, isAuthenticated.value] as const,
  ([initialized, authenticated]) => {
    if (!initialized) return
    if (!authenticated) {
      void router.replace('/login')
      return
    }
    if (messageStore.conversations.length === 0) {
      void messageStore.fetchConversations(true)
    }
  },
  { immediate: true }
)

onUnmounted(() => {
  messageStore.closeConversation()
})
</script>

<template>
  <AppPageContainer title="私信" description="一对一私信会话。">
    <div class="messages-layout" :class="{ 'has-thread': hasThread }">
      <aside class="conversations-pane">
        <ConversationList />
      </aside>
      <section class="thread-pane">
        <NuxtPage />
      </section>
    </div>
  </AppPageContainer>
</template>

<style scoped>
.messages-layout {
  display: grid;
  grid-template-columns: 1fr;
  grid-template-rows: minmax(0, 1fr);
  gap: 14px;
  height: calc(100dvh - 230px);
  min-height: 420px;
}

/* 移动端：会话列表与聊天页互斥显示 */
.conversations-pane {
  min-height: 0;
  border: 1px solid var(--app-glass-border);
  border-radius: var(--radius-kun-lg);
  background: var(--color-content1);
  overflow: hidden;
}

.thread-pane {
  display: none;
  min-height: 0;
}

.messages-layout.has-thread .conversations-pane {
  display: none;
}

.messages-layout.has-thread .thread-pane {
  display: flex;
  min-height: 0;
  border: 1px solid var(--app-glass-border);
  border-radius: var(--radius-kun-lg);
  background: var(--color-content1);
  overflow: hidden;
}

.thread-pane > :deep(*) {
  width: 100%;
}

@media (min-width: 1024px) {
  .messages-layout {
    grid-template-columns: 320px minmax(0, 1fr);
  }

  .conversations-pane,
  .messages-layout.has-thread .conversations-pane {
    display: block;
  }

  .thread-pane,
  .messages-layout.has-thread .thread-pane {
    display: flex;
  }
}
</style>
