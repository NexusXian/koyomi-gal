<script setup lang="ts">
const router = useRouter()
const messageStore = useMessageStore()
const unreadCount = computed(() => messageStore.unreadCount)
</script>

<template>
  <KunTooltip text="私信" position="bottom">
    <KunBadge :count="unreadCount" :max="99" :show="unreadCount > 0">
      <KunButton
        color="default"
        variant="light"
        size="sm"
        rounded="full"
        :is-icon-only="true"
        :aria-label="unreadCount ? `私信，${unreadCount} 条未读` : '私信'"
        @click="router.push('/messages')"
      >
        <KunIcon name="lucide:mail" />
      </KunButton>
    </KunBadge>
  </KunTooltip>
</template>
