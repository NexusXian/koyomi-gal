export default defineNuxtPlugin(() => {
  const userStore = useUserStore()
  const messageStore = useMessageStore()
  let timer: ReturnType<typeof setInterval> | undefined

  const stop = (): void => {
    if (timer) {
      clearInterval(timer)
      timer = undefined
    }
  }

  const start = (): void => {
    stop()
    void messageStore.fetchUnreadCount()
    messageStore.connectRealtime()
    timer = setInterval(() => void messageStore.fetchUnreadCount(), 60_000)
  }

  const handleFocus = (): void => {
    if (userStore.isAuthenticated) {
      void messageStore.fetchUnreadCount()
      if (!messageStore.wsConnected) {
        messageStore.connectRealtime()
      }
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
        messageStore.resetState()
      }
    },
    { immediate: true }
  )

  window.addEventListener('focus', handleFocus)
})
