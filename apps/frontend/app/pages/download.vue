<script setup lang="ts">
import { createAppReleaseService } from '~/services/appRelease'

useSeoMeta({
  title: '下载 App - Koyomi',
  description: '下载 Koyomi Gal 安卓客户端安装包，随时随地浏览 Galgame、小说与社区。'
})

const appReleaseService = createAppReleaseService(useNuxtApp().$api)

const { data: release, pending, error, refresh } = await useAsyncData(
  'app-download-latest',
  () => appReleaseService.latest()
)

const version = computed(() => release.value?.latestVersion?.versionName)

function formatDate(value?: string): string {
  return value ? value.slice(0, 10) : ''
}

function formatSize(bytes?: number | null): string {
  if (!bytes || bytes <= 0) return ''
  const mb = bytes / (1024 * 1024)
  return mb >= 1 ? `${mb.toFixed(1)} MB` : `${Math.round(bytes / 1024)} KB`
}
</script>

<template>
  <AppPageContainer
    title="下载 App"
    description="在手机上随时随地浏览 Koyomi Gal。"
  >
    <div v-if="pending" class="download-list">
      <KunSkeleton v-for="item in 2" :key="item" class="download-skeleton" />
    </div>

    <KunCard v-else-if="error" padding="lg">
      <div class="download-state">
        <p>{{ getApiErrorMessage(error, '下载信息加载失败') }}</p>
        <KunButton color="primary" @click="() => refresh()">重新加载</KunButton>
      </div>
    </KunCard>

    <KunCard v-else-if="!release?.hasUpdate || !release.downloadUrl" padding="lg">
      <div class="download-state">
        <p>安装包暂未发布，请稍后再来看看</p>
      </div>
    </KunCard>

    <div v-else class="download-card">
      <div class="download-main">
        <div class="download-icon">
          <KunIcon name="lucide:smartphone" />
        </div>
        <div class="download-info">
          <h2 class="download-title">
            {{ release.title || 'Koyomi Gal 安卓客户端' }}
          </h2>
          <p class="download-meta">
            <span class="platform-tag">Android</span>
            <span v-if="version">v{{ version }}</span>
            <span v-if="formatDate(release.publishedAt)">
              {{ formatDate(release.publishedAt) }}
            </span>
            <span v-if="formatSize(release.fileSize)">
              {{ formatSize(release.fileSize) }}
            </span>
          </p>
        </div>
        <KunButton
          class="download-button"
          color="primary"
          variant="solid"
          :href="release.downloadUrl"
          target="_blank"
        >
          <KunIcon name="lucide:download" />
          下载安装包
        </KunButton>
      </div>

      <div v-if="release.changelog" class="download-changelog">
        <h3>更新内容</h3>
        <PostContent :content="release.changelog" mode="markdown" />
      </div>

      <p v-if="release.sha256" class="download-sha" :title="release.sha256">
        SHA256：{{ release.sha256 }}
      </p>
    </div>
  </AppPageContainer>
</template>

<style scoped>
.download-list {
  max-width: 720px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.download-skeleton {
  height: 140px;
}

.download-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  color: var(--color-foreground);
}

.download-card {
  max-width: 720px;
  padding: 24px;
  border: 1px solid var(--app-glass-border);
  border-radius: var(--radius-kun-lg);
  background: var(--color-content1);
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.download-main {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.download-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 16px;
  background: color-mix(in srgb, var(--color-primary) 12%, transparent);
  color: var(--color-primary);
  font-size: 28px;
}

.download-info {
  flex: 1;
  min-width: 200px;
}

.download-title {
  margin: 0 0 6px;
  font-size: 17px;
  font-weight: 700;
  color: var(--color-foreground);
}

.download-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  margin: 0;
  color: var(--color-default-500);
  font-size: 13px;
}

.platform-tag {
  padding: 1px 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-primary) 14%, transparent);
  color: var(--color-primary);
  font-weight: 600;
}

.download-button {
  flex: 0 0 auto;
}

.download-changelog {
  padding: 16px 18px;
  border: 1px solid var(--app-glass-border);
  border-radius: var(--radius-kun-md, 12px);
  background: var(--color-content2, transparent);
}

.download-changelog h3 {
  margin: 0 0 10px;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-foreground);
}

.download-sha {
  margin: 0;
  overflow: hidden;
  color: var(--color-default-400);
  font-size: 12px;
  font-family: monospace;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
