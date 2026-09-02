import { defineStore } from 'pinia'

interface NotificationState {
  unreadCount: number
}

export const useNotificationStore = defineStore('notification', {
  state: (): NotificationState => ({
    unreadCount: 0
  }),

  actions: {
    setUnreadCount(count: number): void {
      this.unreadCount = Math.max(0, count)
    },

    decrementUnread(): void {
      this.unreadCount = Math.max(0, this.unreadCount - 1)
    },

    clearUnread(): void {
      this.unreadCount = 0
    }
  }
})
