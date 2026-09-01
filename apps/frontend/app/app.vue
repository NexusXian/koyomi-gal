<script setup lang="ts">
import { theme } from 'ant-design-vue'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'

dayjs.locale('zh-cn')

const colorMode = useCookie<'light' | 'dark'>('koyomi-color-mode', {
  default: () => 'light',
  maxAge: 60 * 60 * 24 * 365
})

const antdTheme = computed(() => ({
  algorithm:
    colorMode.value === 'dark'
      ? theme.darkAlgorithm
      : theme.defaultAlgorithm
}))

const backgroundStore = useBackgroundStore()

onMounted(() => {
  void backgroundStore.initialize()
})

onBeforeUnmount(() => {
  backgroundStore.dispose()
})
</script>

<template>
  <a-config-provider :locale="zhCN" :theme="antdTheme">
    <div class="koyomi-app">
      <AppBackground />

      <div class="koyomi-app-content">
        <NuxtRouteAnnouncer />
        <NuxtLayout>
          <NuxtPage />
        </NuxtLayout>
      </div>
    </div>
  </a-config-provider>
</template>

<style scoped>
.koyomi-app,
.koyomi-app-content {
  min-height: 100vh;
  min-height: 100dvh;
}

.koyomi-app {
  position: relative;
}

.koyomi-app-content {
  position: relative;
  z-index: 1;
}
</style>
