<script setup lang="ts">
import { createHomeService } from '~/services/home'

useSeoMeta({
  title: 'Koyomi Gal - Galgame 资料、评分与玩家社区',
  description: 'Koyomi Gal 是专注于 Galgame 发现、资料整理、作品评价与玩家交流的社区。浏览新作与热门作品，分享游玩感受。',
  ogTitle: 'Koyomi Gal - Galgame 玩家社区',
  ogDescription: '发现值得游玩的 Galgame，与同好分享故事、评价和收藏。',
  ogType: 'website'
})

const homeService = createHomeService(useNuxtApp().$api)
const { data: home, pending, error, refresh } = await useAsyncData(
  'home',
  () => homeService.getHome()
)
</script>

<template>
  <AppPageContainer width="wide">
    <HomeSkeleton v-if="pending" />

    <KunCard v-else-if="error" color="danger" padding="lg" class-name="home-error">
      <div class="error-content">
        <span class="error-icon"><KunIcon name="lucide:cloud-off" /></span>
        <div>
          <h1>首页暂时无法加载</h1>
          <p>{{ getApiErrorMessage(error, '请检查网络后重试。') }}</p>
        </div>
        <KunButton color="primary" @click="() => refresh()">
          <KunIcon name="lucide:refresh-cw" />
          重新加载
        </KunButton>
      </div>
    </KunCard>

    <div v-else-if="home" class="home-content">
      <HomeBanner :banners="home.banners ?? []" />

      <div class="home-primary-grid">
        <HomeGalgameSection
          title="最新 Galgame"
          description="刚刚收录与近期更新的作品"
          icon="lucide:sparkles"
          :items="home.latest_galgames ?? []"
          compact
        />
        <HomeAnnouncement :items="home.announcements ?? []" />
      </div>

      <HomeGalgameSection
        title="热门 Galgame"
        description="社区近期关注度较高的作品"
        icon="lucide:flame"
        :items="home.popular_galgames ?? []"
      />

      <div class="home-post-grid">
        <HomePostSection
          title="最新讨论"
          description="社区刚刚发布的话题"
          icon="lucide:message-square-more"
          :items="home.latest_posts ?? []"
        />
        <HomePostSection
          title="热门讨论"
          description="正在受到关注的话题"
          icon="lucide:trending-up"
          :items="home.popular_posts ?? []"
        />
      </div>
    </div>
  </AppPageContainer>
</template>

<style scoped>
.home-content {
  display: flex;
  flex-direction: column;
  gap: clamp(32px, 5vw, 48px);
}

.home-primary-grid,
.home-post-grid {
  display: grid;
  align-items: start;
  gap: 34px;
}

.error-content {
  display: flex;
  align-items: center;
  gap: 18px;
}

.error-icon {
  display: grid;
  width: 50px;
  height: 50px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 50%;
  background: color-mix(in srgb, var(--color-danger) 12%, transparent);
  color: var(--color-danger);
  font-size: 23px;
}

.error-content div { min-width: 0; flex: 1; }
.error-content h1 { margin: 0; font-size: 18px; }
.error-content p { margin: 5px 0 0; color: var(--color-default-500); font-size: 14px; }

@media (min-width: 900px) {
  .home-primary-grid { grid-template-columns: minmax(0, 1.65fr) minmax(300px, 1fr); }
  .home-post-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 639px) {
  .error-content { align-items: flex-start; flex-wrap: wrap; }
  .error-content :deep(button) { width: 100%; }
}
</style>
