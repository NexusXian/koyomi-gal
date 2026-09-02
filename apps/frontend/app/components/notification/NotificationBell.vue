<script setup lang="ts">
import { listNotifications } from '~/api/generated/notifications/notifications'
import type { DtoNotificationData } from '~/api/generated/models'

const router = useRouter()
const open = ref(false)
const loading = ref(false)
const items = ref<DtoNotificationData[]>([])
const { unreadCount, decrementUnread, markRead } = useNotifications()

async function loadRecent(): Promise<void> {
  loading.value = true
  try {
    const data = unwrapApiData(
      await listNotifications({ page: 1, limit: 8 }),
      '查询通知失败'
    )
    items.value = data.items ?? []
  } catch {
    items.value = []
  } finally {
    loading.value = false
  }
}

function safeTarget(target: string | undefined): string {
  return target?.startsWith('/') && !target.startsWith('//')
    ? target
    : '/notifications'
}

function selectNotification(notification: DtoNotificationData): void {
  open.value = false
  if (!notification.is_read && notification.id) {
    notification.is_read = true
    decrementUnread()
    void markRead(notification.id).catch(() => undefined)
  }
  void router.push(safeTarget(notification.target_url))
}

watch(open, (visible) => {
  if (visible) void loadRecent()
})
</script>

<template>
  <a-popover v-model:open="open" trigger="click" placement="bottomRight">
    <template #content>
      <NotificationDropdown
        :items="items"
        :loading="loading"
        @select="selectNotification"
      />
    </template>
    <KunTooltip text="通知" position="bottom">
      <KunBadge :count="unreadCount" :max="99" :show="unreadCount > 0">
        <KunButton
          color="default"
          variant="light"
          size="sm"
          rounded="full"
          :is-icon-only="true"
          :aria-label="unreadCount ? `通知，${unreadCount} 条未读` : '通知'"
        >
          <KunIcon name="lucide:bell" />
        </KunButton>
      </KunBadge>
    </KunTooltip>
  </a-popover>
</template>
