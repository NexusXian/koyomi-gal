<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { message } from 'ant-design-vue'
import { listNotifications } from '~/api/generated/notifications/notifications'
import type {
  DtoNotificationData,
  ListNotificationsCategory
} from '~/api/generated/models'

type NotificationTab = 'all' | ListNotificationsCategory

useSeoMeta({
  title: '通知 - Koyomi',
  description: '查看互动、审核和系统通知'
})

const router = useRouter()
const userStore = useUserStore()
const { isAuthenticated } = storeToRefs(userStore)
const { decrementUnread, markAllRead, markRead } = useNotifications()
const activeTab = ref<NotificationTab>('all')
const items = ref<DtoNotificationData[]>([])
const page = ref(1)
const limit = 20
const total = ref(0)
const loading = ref(false)
const loadingMore = ref(false)
const errorMessage = ref('')
let requestSequence = 0

const hasMore = computed(() => items.value.length < total.value)

async function load(reset = true): Promise<void> {
  const sequence = ++requestSequence
  if (reset) {
    page.value = 1
    loading.value = true
    items.value = []
  } else {
    loadingMore.value = true
  }
  errorMessage.value = ''

  try {
    const data = unwrapApiData(
      await listNotifications({
        page: page.value,
        limit,
        category: activeTab.value === 'all' ? undefined : activeTab.value
      }),
      '查询通知失败'
    )
    if (sequence !== requestSequence) return
    const nextItems = data.items ?? []
    items.value = reset ? nextItems : [...items.value, ...nextItems]
    total.value = data.total ?? 0
  } catch (error) {
    if (sequence === requestSequence) {
      errorMessage.value = getApiErrorMessage(error, '查询通知失败')
    }
  } finally {
    if (sequence === requestSequence) {
      loading.value = false
      loadingMore.value = false
    }
  }
}

function changeTab(key: string | number): void {
  activeTab.value = String(key) as NotificationTab
  void load()
}

function loadMore(): void {
  page.value += 1
  void load(false)
}

function safeTarget(target: string | undefined): string {
  return target?.startsWith('/') && !target.startsWith('//')
    ? target
    : '/notifications'
}

function selectNotification(notification: DtoNotificationData): void {
  if (!notification.is_read && notification.id) {
    notification.is_read = true
    decrementUnread()
    void markRead(notification.id).catch(() => undefined)
  }
  void router.push(safeTarget(notification.target_url))
}

async function readAll(): Promise<void> {
  const unread = items.value.filter((item) => !item.is_read)
  unread.forEach((item) => {
    item.is_read = true
  })
  try {
    await markAllRead()
  } catch (error) {
    unread.forEach((item) => {
      item.is_read = false
    })
    message.error(getApiErrorMessage(error, '全部标记已读失败'))
  }
}

watch(
  () => [userStore.getInitialized, isAuthenticated.value] as const,
  ([initialized, authenticated]) => {
    if (!initialized) return
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
  <AppPageContainer title="通知" description="查看与你相关的站内消息。">
    <template #actions>
      <KunButton
        color="primary"
        variant="light"
        :disabled="!items.some((item) => !item.is_read)"
        @click="readAll"
      >
        <KunIcon name="lucide:check-check" />
        全部已读
      </KunButton>
    </template>

    <KunCard padding="none" class-name="notification-card">
      <a-tabs :active-key="activeTab" class="notification-tabs" @change="changeTab">
        <a-tab-pane key="all" tab="全部" />
        <a-tab-pane key="interaction" tab="互动" />
        <a-tab-pane key="review" tab="审核" />
        <a-tab-pane key="system" tab="系统" />
      </a-tabs>

      <a-alert
        v-if="errorMessage"
        type="error"
        show-icon
        :message="errorMessage"
      >
        <template #action>
          <KunButton size="sm" color="danger" variant="light" @click="load()">
            重试
          </KunButton>
        </template>
      </a-alert>

      <div v-else-if="loading" class="notification-skeletons">
        <KunSkeleton v-for="index in 6" :key="index" class="notification-skeleton" />
      </div>

      <KunNull v-else-if="items.length === 0" text="暂无通知" />

      <template v-else>
        <NotificationItem
          v-for="item in items"
          :key="item.id"
          :notification="item"
          @select="selectNotification"
        />
        <div v-if="hasMore" class="load-more">
          <KunButton
            color="primary"
            variant="light"
            :loading="loadingMore"
            @click="loadMore"
          >
            加载更多
          </KunButton>
        </div>
      </template>
    </KunCard>
  </AppPageContainer>
</template>

<style scoped>
:deep(.notification-card) {
  overflow: hidden;
}

:deep(.notification-tabs .ant-tabs-nav) {
  margin: 0;
  padding: 0 16px;
}

.notification-skeletons {
  display: grid;
  gap: 12px;
  padding: 16px;
}

.notification-skeleton {
  height: 72px;
}

.load-more {
  display: flex;
  justify-content: center;
  padding: 20px;
}
</style>
