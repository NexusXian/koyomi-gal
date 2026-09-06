<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { getMyExperience } from '~/api/generated/me/me'
import type { LeveldtoUserLevelData } from '~/api/generated/models'

useSeoMeta({
  title: '等级与经验 - Koyomi',
  description: '查看你的等级成长、每日签到与经验记录'
})

const router = useRouter()
const userStore = useUserStore()
const { initialized, isAuthenticated } = storeToRefs(userStore)

const level = ref<LeveldtoUserLevelData | null>(null)
const loading = ref(true)
const errorMessage = ref('')
const historyKey = ref(0)
let requestSequence = 0

async function load(): Promise<void> {
  const sequence = ++requestSequence
  loading.value = true
  errorMessage.value = ''
  try {
    const data = unwrapApiData(await getMyExperience(), '等级信息加载失败')
    if (sequence === requestSequence) {
      level.value = data
    }
  } catch (error) {
    if (sequence === requestSequence) {
      level.value = null
      errorMessage.value = getApiErrorMessage(error, '等级信息加载失败')
    }
  } finally {
    if (sequence === requestSequence) {
      loading.value = false
    }
  }
}

function onCheckedIn(): void {
  historyKey.value++
  void load()
}

watch(
  [initialized, isAuthenticated] as const,
  ([ready, authenticated]) => {
    if (!ready) {
      return
    }
    if (!authenticated) {
      loading.value = false
      void router.replace('/login')
      return
    }
    void load()
  },
  { immediate: true }
)
</script>

<template>
  <AppPageContainer
    title="等级与经验"
    description="通过签到、评论、点赞与内容贡献获取经验，提升你的社区等级。"
  >
    <a-alert
      v-if="errorMessage"
      class="state-alert"
      type="error"
      show-icon
      :message="errorMessage"
    >
      <template #action>
        <KunButton size="sm" color="danger" variant="light" @click="load">
          重试
        </KunButton>
      </template>
    </a-alert>

    <KunSkeleton v-else-if="loading" class="page-skeleton" />

    <div v-else class="experience-layout">
      <div class="experience-main">
        <KunCard padding="md" class-name="experience-card">
          <h2 class="experience-section-title">
            <KunIcon name="lucide:trending-up" />
            等级成长
          </h2>
          <UserLevelProgress :level="level" />
          <dl v-if="level" class="experience-meta">
            <div class="experience-meta-item">
              <dt>当前等级</dt>
              <dd>LV{{ level.level }} {{ level.level_name }}</dd>
            </div>
            <div class="experience-meta-item">
              <dt>总经验</dt>
              <dd>{{ level.total_exp }} EXP</dd>
            </div>
            <div class="experience-meta-item">
              <dt>连续签到</dt>
              <dd>{{ level.consecutive_days }} 天</dd>
            </div>
          </dl>
        </KunCard>

        <KunCard padding="md" class-name="experience-card">
          <h2 class="experience-section-title">
            <KunIcon name="lucide:scroll-text" />
            经验记录
          </h2>
          <UserExperienceHistory :refresh-key="historyKey" />
        </KunCard>
      </div>

      <div class="experience-aside">
        <UserCheckinCard @checked-in="onCheckedIn" />
        <KunCard padding="md" class-name="experience-card">
          <h2 class="experience-section-title">
            <KunIcon name="lucide:sparkle" />
            如何获取经验
          </h2>
          <ul class="experience-tips">
            <li>每日签到可获得经验</li>
            <li>发布评论、点赞内容可获得经验</li>
            <li>评论被他人点赞可获得经验</li>
            <li>贡献游戏资料、资源、CG 通过审核后可获得大量经验</li>
          </ul>
        </KunCard>
      </div>
    </div>
  </AppPageContainer>
</template>

<style scoped>
.state-alert {
  margin-bottom: 14px;
}

.page-skeleton {
  min-height: 320px;
}

.experience-layout {
  display: grid;
  grid-template-columns: 1fr;
  gap: 16px;
}

.experience-main,
.experience-aside {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.experience-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.experience-section-title {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  margin: 0;
  font-size: 15px;
  font-weight: 700;
}

.experience-meta {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin: 0;
}

.experience-meta-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px;
  border: 1px solid var(--app-glass-border);
  border-radius: 10px;
}

.experience-meta-item dt {
  color: var(--color-foreground-500);
  font-size: 12px;
}

.experience-meta-item dd {
  margin: 0;
  font-size: 14px;
  font-weight: 700;
}

.experience-tips {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding-left: 18px;
  color: var(--color-foreground-500);
  font-size: 13px;
}

@media (min-width: 900px) {
  .experience-layout {
    grid-template-columns: minmax(0, 1fr) 300px;
  }
}
</style>
