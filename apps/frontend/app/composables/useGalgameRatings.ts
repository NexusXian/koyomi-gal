import { storeToRefs } from 'pinia'
import {
  deleteMyGalgameRating,
  getGalgameRatingSummary,
  getMyGalgameRating,
  likeGalgameRating,
  listGalgameRatings,
  putMyGalgameRating,
  unlikeGalgameRating
} from '~/api/generated/galgames/galgames'
import type {
  DtoPutRatingRequest,
  DtoRatingRecordData,
  DtoRatingSummaryData
} from '~/api/generated/models'
import type { RatingDimensionValues } from '~/constants/rating'
import { RATING_DIMENSIONS } from '~/constants/rating'

export interface GalgameRatingDraft {
  overall: number | null
  dimensions: RatingDimensionValues
  recommendation: number | null
  reviewText: string
  spoilerLevel: number
}

export type GalgameRatingSort = 'newest' | 'highest' | 'lowest' | 'popular'

const PAGE_SIZE = 20

export function useGalgameRatings(galgameId: MaybeRefOrGetter<number>) {
  const userStore = useUserStore()
  const { initialized, isAuthenticated } = storeToRefs(userStore)

  const summary = ref<DtoRatingSummaryData | null>(null)
  const summaryLoading = ref(false)
  const summaryError = ref('')

  const items = ref<DtoRatingRecordData[]>([])
  const total = ref(0)
  const page = ref(1)
  const sort = ref<GalgameRatingSort>('newest')
  const listLoading = ref(false)
  const listError = ref('')

  const myRating = ref<DtoRatingRecordData | null>(null)
  const myRatingLoading = ref(false)
  const myRatingLoaded = ref(false)
  const saving = ref(false)
  const deleting = ref(false)
  const likePendingIds = ref<number[]>([])

  const totalPage = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))

  function isLikePending(ratingId: number | undefined): boolean {
    return ratingId !== undefined && likePendingIds.value.includes(ratingId)
  }

  async function loadSummary(): Promise<void> {
    const id = toValue(galgameId)
    if (!id) {
      return
    }
    summaryLoading.value = true
    summaryError.value = ''
    try {
      summary.value = unwrapApiData(
        await getGalgameRatingSummary(id),
        '加载评分失败'
      )
    } catch (error) {
      summaryError.value = getApiErrorMessage(error, '评分加载失败')
    } finally {
      summaryLoading.value = false
    }
  }

  async function loadList(reset = false): Promise<void> {
    if (listLoading.value) {
      return
    }
    const id = toValue(galgameId)
    if (!id) {
      return
    }
    if (reset) {
      page.value = 1
    }
    listLoading.value = true
    listError.value = ''
    try {
      const data = unwrapApiData(
        await listGalgameRatings(id, {
          page: page.value,
          page_size: PAGE_SIZE,
          sort: sort.value
        }),
        '加载评价失败'
      )
      items.value = data.items ?? []
      total.value = data.total ?? 0
    } catch (error) {
      listError.value = getApiErrorMessage(error, '评价加载失败')
    } finally {
      listLoading.value = false
    }
  }

  function changeSort(next: GalgameRatingSort): void {
    if (sort.value === next) {
      return
    }
    sort.value = next
    void loadList(true)
  }

  function changePage(next: number): void {
    if (page.value === next) {
      return
    }
    page.value = next
    void loadList()
  }

  async function loadMine(): Promise<void> {
    const id = toValue(galgameId)
    if (!id) {
      return
    }
    myRatingLoading.value = true
    try {
      const data = unwrapApiData(
        await getMyGalgameRating(id),
        '加载我的评价失败'
      )
      myRating.value = (data as DtoRatingRecordData | null) ?? null
    } catch {
      myRating.value = null
    } finally {
      myRatingLoading.value = false
      myRatingLoaded.value = true
    }
  }

  function buildRequest(draft: GalgameRatingDraft): DtoPutRatingRequest {
    const request: DtoPutRatingRequest = {
      overall: draft.overall as number,
      spoiler_level: draft.spoilerLevel as DtoPutRatingRequest['spoiler_level']
    }
    for (const { key } of RATING_DIMENSIONS) {
      const value = draft.dimensions[key]
      if (typeof value === 'number') {
        request[key] = value
      }
    }
    if (draft.recommendation !== null) {
      request.recommendation =
        draft.recommendation as DtoPutRatingRequest['recommendation']
    }
    const text = draft.reviewText.trim()
    if (text) {
      request.review_text = text
    }
    return request
  }

  async function saveRating(draft: GalgameRatingDraft): Promise<void> {
    const id = toValue(galgameId)
    if (!id) {
      return
    }
    saving.value = true
    try {
      const data = unwrapApiData(
        await putMyGalgameRating(id, buildRequest(draft)),
        '保存评价失败'
      )
      myRating.value = data
      await Promise.all([loadSummary(), loadList(true)])
      void refreshNuxtData(`galgame-${id}`)
    } finally {
      saving.value = false
    }
  }

  async function removeRating(): Promise<void> {
    const id = toValue(galgameId)
    if (!id) {
      return
    }
    deleting.value = true
    try {
      await deleteMyGalgameRating(id)
      const mineId = myRating.value?.id
      myRating.value = null
      if (mineId !== undefined) {
        items.value = items.value.filter((item) => item.id !== mineId)
        total.value = Math.max(0, total.value - 1)
      }
      await Promise.all([loadSummary(), loadList(true)])
      void refreshNuxtData(`galgame-${id}`)
    } finally {
      deleting.value = false
    }
  }

  function applyLike(ratingId: number, liked: boolean, likeCount: number): void {
    const patch = (item: DtoRatingRecordData): void => {
      item.liked = liked
      item.like_count = likeCount
    }
    const target = items.value.find((item) => item.id === ratingId)
    if (target) {
      patch(target)
    }
    if (myRating.value?.id === ratingId) {
      patch(myRating.value)
    }
  }

  async function toggleLike(rating: DtoRatingRecordData): Promise<boolean> {
    const ratingId = rating.id
    if (!ratingId || isLikePending(ratingId)) {
      return false
    }
    const wasLiked = Boolean(rating.liked)
    const previousCount = rating.like_count ?? 0
    const optimisticCount = wasLiked
      ? Math.max(0, previousCount - 1)
      : previousCount + 1

    likePendingIds.value = [...likePendingIds.value, ratingId]
    applyLike(ratingId, !wasLiked, optimisticCount)
    try {
      const data = unwrapApiData(
        wasLiked
          ? await unlikeGalgameRating(ratingId)
          : await likeGalgameRating(ratingId),
        '点赞失败'
      )
      applyLike(ratingId, Boolean(data.liked), data.like_count ?? optimisticCount)
      return true
    } catch (error) {
      applyLike(ratingId, wasLiked, previousCount)
      throw error
    } finally {
      likePendingIds.value = likePendingIds.value.filter((id) => id !== ratingId)
    }
  }

  watch(
    [initialized, isAuthenticated],
    ([ready, authed]) => {
      if (ready && authed && !myRatingLoaded.value) {
        void loadMine()
      } else if (ready && !authed) {
        myRating.value = null
        myRatingLoaded.value = false
      }
    },
    { immediate: true }
  )

  return {
    summary,
    summaryLoading,
    summaryError,
    items,
    total,
    page,
    sort,
    totalPage,
    listLoading,
    listError,
    myRating,
    myRatingLoading,
    saving,
    deleting,
    loadSummary,
    loadList,
    changeSort,
    changePage,
    loadMine,
    saveRating,
    removeRating,
    toggleLike,
    isLikePending
  }
}
