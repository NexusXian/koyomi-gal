<script setup lang="ts">
import type { DtoRatingSummaryData } from '~/api/generated/models'
import {
  RATING_DIMENSIONS,
  formatRatingAverage
} from '~/constants/rating'

const props = defineProps<{
  summary: DtoRatingSummaryData | null
  loading?: boolean
  error?: string
}>()

const emit = defineEmits<{
  retry: []
}>()

const hasSummary = computed(
  () =>
    props.summary !== null &&
    (props.summary.count ?? 0) > 0 &&
    props.summary.overall !== null &&
    props.summary.overall !== undefined
)

const radarPoints = computed(() =>
  RATING_DIMENSIONS.map((dimension) => ({
    label: dimension.label,
    value: props.summary?.dimensions?.[dimension.key]?.average ?? null
  }))
)

const ratedDimensionCount = computed(
  () => radarPoints.value.filter((point) => point.value !== null).length
)

const dimensionEntries = computed(() =>
  RATING_DIMENSIONS.map((dimension) => ({
    key: dimension.key,
    label: dimension.label,
    average: props.summary?.dimensions?.[dimension.key]?.average ?? null,
    count: props.summary?.dimensions?.[dimension.key]?.count ?? 0
  }))
)
</script>

<template>
  <div class="rating-summary">
    <div v-if="error && !summary" class="rating-summary-error">
      <span>{{ error }}</span>
      <KunButton size="sm" color="primary" variant="flat" @click="emit('retry')">
        重试
      </KunButton>
    </div>

    <KunNull v-else-if="!loading && !hasSummary" message="暂无评分" />

    <template v-else>
      <div class="rating-summary-main">
        <div class="rating-summary-score">
          <span class="score-number">
            {{ formatRatingAverage(summary?.overall ?? null) }}
          </span>
          <span class="score-max">/ 10</span>
          <span class="score-count">
            {{ (summary?.count ?? 0).toLocaleString('zh-CN') }} 人评分
          </span>
        </div>
        <GalgameRatingRadar
          v-if="ratedDimensionCount >= 3"
          :points="radarPoints"
          class="rating-summary-radar"
        />
      </div>

      <ul class="rating-summary-dimensions">
        <li v-for="dimension in dimensionEntries" :key="dimension.key">
          <span class="dimension-label">{{ dimension.label }}</span>
          <span
            class="dimension-value"
            :class="{ 'dimension-empty': dimension.average === null }"
          >
            {{
              dimension.average === null
                ? '暂无'
                : formatRatingAverage(dimension.average)
            }}
          </span>
          <span class="dimension-count">{{ dimension.count }} 人</span>
        </li>
      </ul>
    </template>
  </div>
</template>

<style scoped>
.rating-summary-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 0;
  color: var(--color-default-500);
  font-size: 14px;
}

.rating-summary-main {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.rating-summary-score {
  display: flex;
  align-items: baseline;
  justify-content: center;
  gap: 6px;
}

.score-number {
  font-size: 40px;
  font-weight: 800;
  line-height: 1;
  color: var(--color-primary);
}

.score-max {
  color: var(--color-default-400);
  font-size: 15px;
}

.score-count {
  margin-left: 8px;
  color: var(--color-default-500);
  font-size: 13px;
}

.rating-summary-radar {
  width: 240px;
}

.rating-summary-dimensions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px 18px;
  margin: 16px 0 0;
  padding: 0;
  list-style: none;
}

.rating-summary-dimensions li {
  display: flex;
  align-items: baseline;
  gap: 8px;
  font-size: 13px;
}

.dimension-label {
  flex: 0 0 auto;
  color: var(--color-default-500);
}

.dimension-value {
  font-weight: 700;
  color: var(--color-foreground);
}

.dimension-value.dimension-empty {
  font-weight: 400;
  color: var(--color-default-400);
}

.dimension-count {
  margin-left: auto;
  color: var(--color-default-400);
  font-size: 12px;
}

@media (min-width: 768px) {
  .rating-summary-main {
    flex-direction: row;
    justify-content: center;
    gap: 32px;
  }

  .rating-summary-score {
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }

  .score-count {
    margin-left: 0;
  }

  .rating-summary-dimensions {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
</style>
