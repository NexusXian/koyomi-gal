export default defineNuxtPlugin(() => {
  const userStore = useUserStore()
  const { fetchUnreadCount, clearUnread } = useNotifications()
  let timer: ReturnType<typeof setInterval> | undefined

  const stop = (): void => {
    if (timer) {
      clearInterval(timer)
      timer = undefined
    }
  }

  const start = (): void => {
    stop()
    void fetchUnreadCount()
    timer = setInterval(() => void fetchUnreadCount(), 60_000)
  }

  const handleFocus = (): void => {
    if (userStore.isAuthenticated) {
      void fetchUnreadCount()
    }
  }

  watch(
    () => [userStore.getInitialized, userStore.isAuthenticated] as const,
    ([initialized, authenticated]) => {
      if (!initialized) return
      if (authenticated) {
        start()
      } else {
        stop()
        clearUnread()
      }
    },
    { immediate: true }
  )

  window.addEventListener('focus', handleFocus)
})
