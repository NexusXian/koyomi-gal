import {
  getNotificationUnreadCount,
  markAllNotificationsRead,
  markNotificationRead
} from '~/api/generated/notifications/notifications'

export function useNotifications() {
  const userStore = useUserStore()
  const notificationStore = useNotificationStore()
  const fetchingCount = useState('notification-count-loading', () => false)

  async function fetchUnreadCount(): Promise<void> {
    if (!userStore.isAuthenticated || fetchingCount.value) {
      if (!userStore.isAuthenticated) {
        notificationStore.clearUnread()
      }
      return
    }

    fetchingCount.value = true
    try {
      const data = unwrapApiData(
        await getNotificationUnreadCount(),
        '查询未读通知失败'
      )
      notificationStore.setUnreadCount(data.count ?? 0)
    } catch {
    } finally {
      fetchingCount.value = false
    }
  }

  async function markRead(id: number): Promise<void> {
    await markNotificationRead(id)
  }

  async function markAllRead(): Promise<void> {
    await markAllNotificationsRead()
    notificationStore.clearUnread()
  }

  return {
    unreadCount: computed(() => notificationStore.unreadCount),
    fetchingCount,
    fetchUnreadCount,
    markRead,
    markAllRead,
    decrementUnread: notificationStore.decrementUnread,
    clearUnread: notificationStore.clearUnread
  }
}
