export const RATING_DIMENSIONS = [
  { key: 'visual', label: '画面' },
  { key: 'story', label: '剧情' },
  { key: 'music', label: '音乐' },
  { key: 'character', label: '角色' },
  { key: 'branch', label: '分支' },
  { key: 'system', label: '系统' },
  { key: 'voice', label: '配音' },
  { key: 'replay', label: '重玩' }
] as const

export type RatingDimensionKey = (typeof RATING_DIMENSIONS)[number]['key']

export type RatingDimensionValues = Partial<
  Record<RatingDimensionKey, number | null>
>

export const RECOMMENDATION_OPTIONS = [
  { value: 2, label: '强烈推荐', color: 'success' },
  { value: 1, label: '推荐', color: 'processing' },
  { value: 0, label: '中立', color: 'default' },
  { value: -1, label: '不推荐', color: 'error' }
] as const

export const SPOILER_OPTIONS = [
  { value: 0, label: '无剧透', color: 'success' },
  { value: 1, label: '部分剧透', color: 'warning' },
  { value: 2, label: '严重剧透', color: 'error' }
] as const

export const RATING_SORT_OPTIONS = [
  { value: 'newest', label: '最新' },
  { value: 'highest', label: '最高分' },
  { value: 'lowest', label: '最低分' },
  { value: 'popular', label: '最热门' }
] as const

export function recommendationLabel(
  value: number | null | undefined
): string | undefined {
  return RECOMMENDATION_OPTIONS.find((item) => item.value === value)?.label
}

export function recommendationColor(
  value: number | null | undefined
): string | undefined {
  return RECOMMENDATION_OPTIONS.find((item) => item.value === value)?.color
}

export function spoilerLabel(
  value: number | null | undefined
): string | undefined {
  return SPOILER_OPTIONS.find((item) => item.value === value)?.label
}

export function spoilerColor(
  value: number | null | undefined
): string | undefined {
  return SPOILER_OPTIONS.find((item) => item.value === value)?.color
}

export function formatRatingAverage(value: number | null | undefined): string {
  if (value === null || value === undefined) {
    return '-'
  }
  return value.toFixed(1)
}
