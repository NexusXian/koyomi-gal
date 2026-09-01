<script setup lang="ts">
import { storeToRefs } from 'pinia'

const backgroundStore = useBackgroundStore()
const { settings, backgroundUrl, hasBackground } = storeToRefs(backgroundStore)

const backgroundStyle = computed(() => {
  const blurOverflow = settings.value.blur > 0 ? settings.value.blur * 2 : 0

  return {
    backgroundImage: backgroundUrl.value
      ? `url(${JSON.stringify(backgroundUrl.value)})`
      : undefined,
    opacity: settings.value.opacity,
    filter: settings.value.blur > 0 ? `blur(${settings.value.blur}px)` : undefined,
    backgroundPosition: settings.value.position,
    backgroundSize: settings.value.size,
    inset: blurOverflow > 0 ? `-${blurOverflow}px` : '0'
  }
})
</script>

<template>
  <div
    v-if="hasBackground"
    class="app-background"
    :style="backgroundStyle"
    aria-hidden="true"
  />
</template>

<style scoped>
.app-background {
  position: fixed;
  z-index: 0;
  width: auto;
  height: auto;
  min-height: 100vh;
  min-height: 100dvh;
  background-repeat: no-repeat;
  pointer-events: none;
  transform: translateZ(0);
  transition:
    opacity 200ms ease,
    filter 200ms ease;
  will-change: opacity;
}
</style>
