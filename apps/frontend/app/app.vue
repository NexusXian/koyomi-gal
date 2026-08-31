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
</script>

<template>
  <a-config-provider :locale="zhCN" :theme="antdTheme">
    <NuxtRouteAnnouncer />
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </a-config-provider>
</template>
