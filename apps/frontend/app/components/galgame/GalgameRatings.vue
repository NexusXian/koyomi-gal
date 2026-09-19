<script setup lang="ts">
import { message } from 'ant-design-vue'
import { storeToRefs } from 'pinia'
import type { DtoRatingRecordData } from '~/api/generated/models'
import { RATING_SORT_OPTIONS } from '~/constants/rating'
import type {
  GalgameRatingDraft,
  GalgameRatingSort
} from '~/composables/useGalgameRatings'

const props = defineProps<{
  galgameId: number
}>()

const router = useRouter()
const userStore = useUserStore()
const { isAuthenticated } = storeToRefs(userStore)

const ratings = useGalgameRatings(computed(() => props.galgameId))

const editorOpen = ref(false)

onMounted(() => {
  void ratings.loadSummary()
  void ratings.loadList(true)
})

function openEditor(): void {
  if (!isAuthenticated.value) {
    message.warning('登录后才能评价')
    void router.push('/login')
    return
  }
  editorOpen.value = true
}

async function handleSubmit(draft: GalgameRatingDraft): Promise<void> {
  try {
    await ratings.saveRating(draft)
    editorOpen.value = false
    message.success('评价已保存')
  } catch (error) {
    message.error(getApiErrorMessage(error, '保存评价失败'))
  }
}

async function handleDelete(): Promise<void> {
  try {
    await ratings.removeRating()
    editorOpen.value = false
    message.success('评价已删除')
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除评价失败'))
  }
}

async function handleLike(rating: DtoRatingRecordData): Promise<void> {
  if (!isAuthenticated.value) {
    message.warning('登录后才能点赞')
    return
  }
  try {
    await ratings.toggleLike(rating)
  } catch (error) {
    message.error(getApiErrorMessage(error, '点赞失败，请稍后重试'))
  }
}
</script>

<template>
  <KunCard padding="lg" class-name="community-ratings-card">
    <div class="section-head-row">
      <KunHeader
        name="社区评分"
        :description="
          ratings.total.value
            ? `${ratings.total.value.toLocaleString('zh-CN')} 条评价`
            : undefined
        "
        scale="h3"
        class="section-heading"
      />
      <KunButton color="primary" variant="flat" size="sm" @click="openEditor">
        <KunIcon name="lucide:pencil-line" />
        {{ ratings.myRating.value ? '修改我的评价' : '发表评价' }}
      </KunButton>
    </div>

    <GalgameRatingSummary
      :summary="ratings.summary.value"
      :loading="ratings.summaryLoading.value"
      :error="ratings.summaryError.value"
      @retry="ratings.loadSummary()"
    />

    <KunDivider />

    <div class="section-head-row list-head">
      <span class="list-title">评价列表</span>
      <a-select
        :value="ratings.sort.value"
        size="small"
        class="sort-select"
        :options="
          RATING_SORT_OPTIONS.map((option) => ({
            value: option.value,
            label: option.label
          }))
        "
        @change="(value: unknown) => ratings.changeSort(value as GalgameRatingSort)"
      />
    </div>

    <div
      v-if="ratings.listError.value && ratings.items.value.length === 0"
      class="list-error"
    >
      <span>{{ ratings.listError.value }}</span>
      <KunButton
        size="sm"
        color="primary"
        variant="flat"
        @click="ratings.loadList(true)"
      >
        重试
      </KunButton>
    </div>

    <a-spin v-else :spinning="ratings.listLoading.value && ratings.items.value.length === 0">
      <KunNull
        v-if="ratings.items.value.length === 0 && !ratings.listLoading.value"
        message="暂无评价，来发表第一条评价吧"
      />
      <TransitionGroup v-else name="rating-list" tag="div" class="rating-list">
        <GalgameRatingCard
          v-for="rating in ratings.items.value"
          :key="rating.id"
          :rating="rating"
          :like-pending="ratings.isLikePending(rating.id)"
          @like="handleLike"
        />
      </TransitionGroup>
    </a-spin>

    <div v-if="ratings.totalPage.value > 1" class="rating-pagination">
      <KunPagination
        :current-page="ratings.page.value"
        :total-page="ratings.totalPage.value"
        :is-loading="ratings.listLoading.value"
        @update:current-page="ratings.changePage"
      />
    </div>

    <GalgameRatingEditor
      v-model:open="editorOpen"
      :initial="ratings.myRating.value"
      :saving="ratings.saving.value"
      :deleting="ratings.deleting.value"
      @submit="handleSubmit"
      @delete="handleDelete"
    />
  </KunCard>
</template>

<style scoped>
.community-ratings-card {
  margin-bottom: 18px;
}

.section-heading {
  margin-bottom: 4px;
}

.section-head-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.list-head {
  margin-bottom: 10px;
}

.list-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-foreground);
}

.sort-select {
  width: 110px;
}

.list-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 0;
  color: var(--color-default-500);
  font-size: 14px;
}

.rating-list {
  display: grid;
  gap: 12px;
}

.rating-pagination {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

.rating-list-enter-active,
.rating-list-leave-active {
  transition: opacity 0.2s ease;
}

.rating-list-enter-from,
.rating-list-leave-to {
  opacity: 0;
}
</style>
