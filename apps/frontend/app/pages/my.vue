<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { listMyGalgames } from '~/api/generated/me/me'
import type { DtoGalgameListData } from '~/api/generated/models'

type CollectionType = 'uploaded' | 'favorite'

useSeoMeta({
  title: '我的 - Koyomi',
  description: '查看你上传和收藏的 Galgame'
})

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { isAuthenticated } = storeToRefs(userStore)
const activeCollection = ref<CollectionType>(
  route.query.type === 'favorite' ? 'favorite' : 'uploaded'
)
const page = ref(Math.max(1, Number(route.query.page ?? 1) || 1))
const limit = 20
const listData = ref<DtoGalgameListData>()
const loading = ref(true)
const errorMessage = ref('')
let requestSequence = 0

const items = computed(() => listData.value?.items ?? [])
const total = computed(() => listData.value?.total ?? 0)
const totalPage = computed(() => Math.max(1, Math.ceil(total.value / limit)))

async function load(): Promise<void> {
  const sequence = ++requestSequence
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await listMyGalgames({
      type: activeCollection.value,
      page: page.value,
      limit
    })
    if (sequence === requestSequence) {
      listData.value = unwrapApiData(response, '查询我的 Galgame 失败')
    }
  } catch (error) {
    if (sequence === requestSequence) {
      listData.value = undefined
      errorMessage.value = getApiErrorMessage(error, '查询我的 Galgame 失败')
    }
  } finally {
    if (sequence === requestSequence) {
      loading.value = false
    }
  }
}

function changeCollection(key: string | number): void {
  activeCollection.value = key === 'favorite' ? 'favorite' : 'uploaded'
  page.value = 1
  void router.replace({
    query: {
      ...route.query,
      type: activeCollection.value === 'favorite' ? 'favorite' : undefined,
      page: undefined
    }
  })
}

function updatePage(next: number): void {
  page.value = next
  void router.replace({
    query: {
      ...route.query,
      type: activeCollection.value === 'favorite' ? 'favorite' : undefined,
      page: next > 1 ? next : undefined
    }
  })
}

function cardTarget(id: number | undefined, status: number | undefined): string {
  if (activeCollection.value === 'uploaded' && status !== 1) {
    return `/galgames/${id}/edit`
  }
  return `/galgames/${id}`
}

watch(
  () => [
    userStore.getInitialized,
    isAuthenticated.value,
    activeCollection.value,
    page.value
  ] as const,
  ([initialized, authenticated]) => {
    if (!initialized) {
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
    title="我的"
    description="查看你上传和收藏的 Galgame。"
    width="wide"
  >
    <template #actions>
      <KunButton color="primary" variant="bordered" href="/galgames">
        <KunIcon name="lucide:compass" />
        发现 Galgame
      </KunButton>
    </template>

    <KunCard padding="md" class-name="collection-tabs">
      <a-tabs :active-key="activeCollection" @change="changeCollection">
        <a-tab-pane key="uploaded" tab="我的上传" />
        <a-tab-pane key="favorite" tab="我的收藏" />
      </a-tabs>
    </KunCard>

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

    <a-alert
      v-else-if="!loading && items.length === 0"
      class="state-alert"
      type="info"
      show-icon
      :message="activeCollection === 'uploaded' ? '还没有上传 Galgame' : '还没有收藏 Galgame'"
      :description="activeCollection === 'uploaded' ? '你上传的条目会显示在这里。' : '收藏感兴趣的 Galgame 后，可以在这里快速找到它们。'"
    />

    <div v-else class="galgame-grid">
      <template v-if="loading">
        <KunSkeleton
          v-for="index in 8"
          :key="index"
          class="galgame-skeleton"
        />
      </template>

      <template v-else>
        <GalgameCard
          v-for="galgame in items"
          :key="galgame.id"
          :galgame="galgame"
          :show-status="activeCollection === 'uploaded'"
          :to="cardTarget(galgame.id, galgame.status)"
        />
      </template>
    </div>

    <div v-if="!errorMessage && totalPage > 1" class="pagination-row">
      <KunPagination
        :current-page="page"
        :total-page="totalPage"
        :is-loading="loading"
        @update:current-page="updatePage"
      />
    </div>
  </AppPageContainer>
</template>

<style scoped>
.collection-tabs {
  margin-bottom: 18px;
}

.collection-tabs :deep(.ant-tabs-nav) {
  margin-bottom: 0;
}

.state-alert {
  margin: 8px 0;
}

.galgame-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.galgame-skeleton {
  min-height: 320px;
}

.pagination-row {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

@media (min-width: 640px) {
  .galgame-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (min-width: 1024px) {
  .galgame-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
</style>
