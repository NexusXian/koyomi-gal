<script setup lang="ts">
import { listGalgames } from '~/api/generated/galgames/galgames'
import { listDevelopers } from '~/api/generated/developers/developers'
import { listTags } from '~/api/generated/tags/tags'
import type {
  DtoDeveloperSummary,
  DtoGalgameListData,
  DtoTagSummary
} from '~/api/generated/models'
import {
  AGE_RATINGS,
  GALGAME_SORTS,
  domainSortSlug
} from '~/constants/domain'

useSeoMeta({
  title: 'Galgame - Koyomi',
  description: '浏览和发现 Galgame'
})

const route = useRoute()
const router = useRouter()

const keywordInput = ref(
  typeof route.query.q === 'string' ? route.query.q : ''
)
const keyword = ref(keywordInput.value)
let keywordTimer: ReturnType<typeof setTimeout> | undefined
let applyingRoute = false

watch(keywordInput, (value) => {
  if (applyingRoute) return
  if (keywordTimer) {
    clearTimeout(keywordTimer)
  }

  keywordTimer = setTimeout(() => {
    keyword.value = value
  }, 350)
}, { flush: 'sync' })
const developerId = ref<number | undefined>(
  route.query.developer ? Number(route.query.developer) : undefined
)
const companyId = ref<number | undefined>(
  positiveID(route.query.company_id)
)
const tagIds = ref<number[]>(
  typeof route.query.tags === 'string' && route.query.tags
    ? route.query.tags.split(',').map(Number).filter(Boolean)
    : []
)
const ageRating = ref<number | undefined>(
  route.query.age !== undefined ? Number(route.query.age) : undefined
)
const sortIndex = ref(Number(route.query.sort ?? 0) || 0)
const page = ref(Math.max(1, Number(route.query.page ?? 1) || 1))

function positiveID(value: unknown): number | undefined {
  const id = typeof value === 'string' ? Number(value) : NaN
  return Number.isSafeInteger(id) && id > 0 ? id : undefined
}

watch(() => route.query, (query) => {
  applyingRoute = true
  clearTimeout(keywordTimer)
  keywordInput.value = typeof query.q === 'string' ? query.q : ''
  keyword.value = keywordInput.value
  developerId.value = positiveID(query.developer)
  companyId.value = positiveID(query.company_id)
  tagIds.value = typeof query.tags === 'string' && query.tags
    ? query.tags.split(',').map(positiveID).filter((id): id is number => id !== undefined)
    : []
  ageRating.value = query.age !== undefined ? Number(query.age) : undefined
  sortIndex.value = Number(query.sort ?? 0) || 0
  page.value = Math.max(1, Math.floor(Number(query.page ?? 1)) || 1)
  applyingRoute = false
}, { flush: 'sync' })

const limit = 20
const developers = ref<DtoDeveloperSummary[]>([])
const tags = ref<DtoTagSummary[]>([])

const { data: listData, pending, error, refresh } = await useAsyncData<
  DtoGalgameListData,
  Error
>(
  computed(() => `galgames-${JSON.stringify([keyword.value, developerId.value, companyId.value, tagIds.value, ageRating.value, sortIndex.value, page.value])}`),
  async () => {
    const response = await listGalgames({
      keyword: keyword.value.trim() || undefined,
      developer_id: developerId.value,
      company_id: companyId.value,
      tag_ids: tagIds.value.length
        ? tagIds.value.join(',')
        : undefined,
      age_rating: ageRating.value,
      sort: domainSortSlug(GALGAME_SORTS, sortIndex.value),
      page: page.value,
      limit
    })
    return unwrapApiData(response, '查询 Galgame 失败')
  }
)

const items = computed(() => listData.value?.items ?? [])
const total = computed(() => listData.value?.total ?? 0)
const totalPage = computed(() =>
  Math.max(1, Math.ceil(total.value / limit))
)

const { has } = usePermissions()

function syncQuery(): void {
  void router.replace({
    query: {
      ...route.query,
      q: keyword.value.trim() || undefined,
      developer: developerId.value ?? undefined,
      company_id: companyId.value,
      tags: tagIds.value.length ? tagIds.value.join(',') : undefined,
      age: ageRating.value ?? undefined,
      sort: sortIndex.value || undefined,
      page: page.value > 1 ? page.value : undefined
    }
  })
}

watch([keyword, developerId, companyId, tagIds, ageRating, sortIndex], () => {
  if (applyingRoute) return
  page.value = 1
  syncQuery()
}, { flush: 'sync' })

watch(page, () => { if (!applyingRoute) syncQuery() }, { flush: 'sync' })

onBeforeUnmount(() => clearTimeout(keywordTimer))

onMounted(() => {
  try {
    void listDevelopers().then((response) => {
      developers.value = unwrapApiData(response) ?? []
    })
    void listTags().then((response) => {
      tags.value = unwrapApiData(response) ?? []
    })
  } catch {
    /* 过滤选项加载失败时列表仍可用 */
  }
})

function updatePage(next: number): void {
  page.value = next
}
</script>

<template>
  <AppPageContainer
    title="Galgame"
    description="浏览、搜索并发现你感兴趣的 Galgame。"
    width="wide"
  >
    <template #actions>
      <KunButton
        v-if="has('galgame:create')"
        color="primary"
        href="/galgames/new"
      >
        <KunIcon name="lucide:plus" />
        新建 Galgame
      </KunButton>
    </template>

    <KunCard padding="md" class-name="filter-card">
      <div class="filter-bar">
        <a-input
          v-model:value="keywordInput"
          class="filter-keyword"
          placeholder="搜索标题或别名"
          allow-clear
        >
          <template #prefix>
            <KunIcon name="lucide:search" class="filter-search-icon" />
          </template>
        </a-input>

        <a-select
          v-model:value="developerId"
          class="filter-item"
          :options="
            developers.map((item) => ({ value: item.id, label: item.name }))
          "
          placeholder="开发商"
          allow-clear
          show-search
          option-filter-prop="label"
        />

        <CompanySelector v-model="companyId" placeholder="搜索关联公司" />

        <a-select
          v-model:value="tagIds"
          class="filter-item filter-tags"
          :options="
            tags.map((item) => ({ value: item.id, label: item.name }))
          "
          mode="multiple"
          placeholder="Tag"
          allow-clear
          :max-tag-count="2"
          option-filter-prop="label"
        />

        <a-select
          v-model:value="ageRating"
          class="filter-item filter-narrow"
          :options="
            AGE_RATINGS.map((item) => ({
              value: item.value,
              label: item.label
            }))
          "
          placeholder="年龄等级"
          allow-clear
        />

        <a-select
          v-model:value="sortIndex"
          class="filter-item filter-narrow"
          :options="
            GALGAME_SORTS.map((item, index) => ({
              value: index,
              label: item.label
            }))
          "
        />
      </div>
    </KunCard>

    <a-alert v-if="error" type="error" show-icon :message="getApiErrorMessage(error, '查询 Galgame 失败')">
      <template #action><a-button @click="refresh()">重试</a-button></template>
    </a-alert>
    <a-alert
      v-else-if="!pending && items.length === 0"
      class="empty-alert"
      type="info"
      show-icon
      message="没有找到符合条件的 Galgame"
      description="尝试更换关键词或清空筛选条件。"
    />

    <div v-else class="galgame-grid">
      <template v-if="pending">
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
        />
      </template>
    </div>

    <div v-if="totalPage > 1" class="pagination-row">
      <KunPagination
        :current-page="page"
        :total-page="totalPage"
        :is-loading="pending"
        @update:current-page="updatePage"
      />
    </div>
  </AppPageContainer>
</template>

<style scoped>
.filter-card {
  margin-bottom: 18px;
}

.filter-bar {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 10px;
}

.filter-search-icon {
  color: var(--color-default-400);
}

.filter-item {
  width: 100%;
}

.galgame-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.galgame-skeleton {
  min-height: 320px;
}

.empty-alert {
  margin: 8px 0;
}

.pagination-row {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

@media (min-width: 640px) {
  .filter-bar {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .galgame-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (min-width: 1024px) {
  .filter-bar {
    grid-template-columns: minmax(200px, 1.4fr) repeat(3, minmax(0, 1fr)) repeat(2, minmax(0, 0.7fr));
  }

  .galgame-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
</style>
