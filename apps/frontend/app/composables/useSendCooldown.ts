export function useSendCooldown(seconds = 60) {
  const cooldown = ref(0)
  let timer: ReturnType<typeof setInterval> | undefined

  function stop(): void {
    if (timer) {
      clearInterval(timer)
      timer = undefined
    }
    cooldown.value = 0
  }

  function start(): void {
    stop()
    cooldown.value = seconds

    timer = setInterval(() => {
      if (cooldown.value <= 1) {
        stop()
        return
      }
      cooldown.value -= 1
    }, 1000)
  }

  onScopeDispose(stop)

  return { cooldown, start, stop }
}
