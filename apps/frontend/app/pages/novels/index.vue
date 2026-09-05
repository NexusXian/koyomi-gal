<script setup lang="ts">
import type {
  DtoNovelListData,
  ListNovelsReleaseStatus,
  ListNovelsSort
} from '~/api/generated/models'
import {
  NOVEL_RELEASE_STATUS,
  NOVEL_SORTS,
  domainSortSlug
} from '~/constants/domain'
import { listNovels } from '~/api/generated/novels/novels'
import { listTags } from '~/api/generated/tags/tags'

useSeoMeta({
  title: '小说 - Koyomi',
  description: '轻小说资料库：查询作品、卷册、资源与关联视觉小说'
})

const route = useRoute()
const router = useRouter()

const keywordInput = ref(
  typeof route.query.q === 'string' ? route.query.q : ''
)
const keyword = ref(keywordInput.value)
let keywordTimer: ReturnType<typeof setTimeout> | undefined

watch(keywordInput, (value) => {
  if (keywordTimer) {
    clearTimeout(keywordTimer)
  }

  keywordTimer = setTimeout(() => {
    keyword.value = value
  }, 350)
})

const tagIds = ref<number[]>(
  typeof route.query.tags === 'string' && route.query.tags
    ? route.query.tags.split(',').map(Number).filter(Boolean)
    : []
)
const releaseStatus = ref(
  typeof route.query.status === 'string' ? route.query.status : ''
)
const publisher = ref(
  typeof route.query.publisher === 'string' ? route.query.publisher : ''
)
const label = ref(
  typeof route.query.label === 'string' ? route.query.label : ''
)
const sortIndex = ref(Number(route.query.sort ?? 0) || 0)
const page = ref(Math.max(1, Number(route.query.page ?? 1) || 1))

const limit = 12
const tags = ref<{ id: number; name: string }[]>([])

const { data: listData, pending } = await useAsyncData<DtoNovelListData, Error>(
  `novels-${JSON.stringify(route.query)}`,
  async () => {
    const response = await listNovels({
      keyword: keyword.value.trim() || undefined,
      tag_ids: tagIds.value.length ? tagIds.value.join(',') : undefined,
      release_status: releaseStatus.value as ListNovelsReleaseStatus | undefined,
      publisher: publisher.value.trim() || undefined,
      label: label.value.trim() || undefined,
      sort: domainSortSlug(NOVEL_SORTS, sortIndex.value) as ListNovelsSort,
      page: page.value,
      limit
    })
    return unwrapApiData(response, '查询小说失败')
  },
  { watch: [keyword, tagIds, releaseStatus, publisher, label, sortIndex, page] }
)

const items = computed(() => listData.value?.items ?? [])
const total = computed(() => listData.value?.total ?? 0)
const totalPage = computed(() => Math.max(1, Math.ceil(total.value / limit)))

const { has } = usePermissions()

function syncQuery(): void {
  void router.replace({
    query: {
      ...route.query,
      q: keyword.value.trim() || undefined,
      tags: tagIds.value.length ? tagIds.value.join(',') : undefined,
      status: releaseStatus.value || undefined,
      publisher: publisher.value.trim() || undefined,
      label: label.value.trim() || undefined,
      sort: sortIndex.value || undefined,
      page: page.value > 1 ? page.value : undefined
    }
  })
}

watch([keyword, tagIds, releaseStatus, publisher, label, sortIndex], () => {
  page.value = 1
  syncQuery()
})

watch(page, syncQuery)

onMounted(() => {
  try {
    void listTags().then((response) => {
      tags.value = (unwrapApiData(response) ?? [])
        .filter((item) => item.id !== undefined && item.name !== undefined)
        .map((item) => ({ id: item.id as number, name: item.name as string }))
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
    title="小说"
    description="浏览、搜索小说作品、卷册与关联视觉小说。"
    width="wide"
  >
    <template #actions>
      <KunButton v-if="has('novel:create')" color="primary" href="/novels/new">
        <KunIcon name="lucide:plus" />
        新建小说
      </KunButton>
    </template>

    <KunCard padding="md" class-name="filter-card">
      <div class="filter-bar">
        <a-input
          v-model:value="keywordInput"
          class="filter-keyword"
          placeholder="搜索标题 / 原文标题 / 作者"
          allow-clear
        >
          <template #prefix>
            <KunIcon name="lucide:search" class="filter-search-icon" />
          </template>
        </a-input>

        <a-select
          v-model:value="releaseStatus"
          class="filter-item"
          :options="
            NOVEL_RELEASE_STATUS.map((item) => ({
              value: item.slug,
              label: item.label
            }))
          "
          placeholder="连载状态"
          allow-clear
        />

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

        <a-input
          v-model:value="publisher"
          class="filter-item"
          placeholder="出版社"
          allow-clear
        />

        <a-input
          v-model:value="label"
          class="filter-item"
          placeholder="文库"
          allow-clear
        />

        <a-select
          v-model:value="sortIndex"
          class="filter-item filter-narrow"
          :options="
            NOVEL_SORTS.map((item, index) => ({
              value: index,
              label: item.label
            }))
          "
        />
      </div>
    </KunCard>

    <a-alert
      v-if="!pending && items.length === 0"
      class="empty-alert"
      type="info"
      show-icon
      message="没有找到符合条件的小说"
      description="尝试更换关键词或清空筛选条件。"
    />

    <div v-else class="novel-grid">
      <template v-if="pending">
        <KunSkeleton v-for="index in 8" :key="index" class="novel-skeleton" />
      </template>

      <template v-else>
        <NovelCard v-for="novel in items" :key="novel.id" :novel="novel" />
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

.novel-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.novel-skeleton {
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

  .novel-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (min-width: 1024px) {
  .filter-bar {
    grid-template-columns: minmax(220px, 1.4fr) minmax(0, 0.8fr) minmax(0, 1.1fr) minmax(0, 0.8fr) minmax(0, 0.8fr) minmax(0, 0.6fr);
  }

  .novel-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
</style>
