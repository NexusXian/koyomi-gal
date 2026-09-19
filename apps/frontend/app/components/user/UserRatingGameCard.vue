<script setup lang="ts">
import type { UserdtoProfileGalgameData } from '~/api/generated/models'
import { formatDate } from '~/constants/domain'
import {
  RATING_DIMENSIONS,
  recommendationColor,
  recommendationLabel,
  spoilerColor,
  spoilerLabel
} from '~/constants/rating'

const props = defineProps<{
  rating: UserdtoProfileGalgameData
}>()

const revealed = ref(false)

watch(
  () => props.rating.id,
  () => {
    revealed.value = false
  }
)

const ratedDimensions = computed(() =>
  RATING_DIMENSIONS.map((dimension) => ({
    key: dimension.key,
    label: dimension.label,
    value: props.rating[dimension.key] ?? null
  })).filter((dimension) => dimension.value !== null)
)

const spoilerLevel = computed(() => props.rating.spoiler_level ?? 0)

const isSevereSpoiler = computed(() => spoilerLevel.value === 2)

const reviewText = computed(() => props.rating.review_text?.trim() ?? '')
</script>

<template>
  <article class="rating-game-card">
    <NuxtLink :to="`/galgames/${rating.id}`" class="rating-game-main">
      <div class="game-cover">
        <SensitiveImage
          v-if="rating.cover_url"
          :src="rating.cover_url"
          :alt="rating.title || 'Galgame 封面'"
          :sensitive="rating.cover_sensitive"
        />
        <KunIcon v-else name="lucide:image" />
      </div>
      <div class="game-info">
        <h2>{{ rating.title || `Galgame #${rating.id}` }}</h2>
        <span v-if="rating.score != null" class="game-score">
          <KunIcon name="lucide:star" />{{ rating.score }} 分
        </span>
        <a-tag
          v-if="rating.recommendation !== undefined && rating.recommendation !== null"
          :color="recommendationColor(rating.recommendation)"
          class="game-recommendation"
        >
          {{ recommendationLabel(rating.recommendation) }}
        </a-tag>
        <ul v-if="ratedDimensions.length" class="game-dimensions">
          <li v-for="dimension in ratedDimensions" :key="dimension.key">
            <span>{{ dimension.label }}</span>
            <strong>{{ dimension.value }}</strong>
          </li>
        </ul>
      </div>
    </NuxtLink>

    <div v-if="reviewText" class="rating-game-review">
      <a-alert
        v-if="spoilerLevel === 1"
        class="review-spoiler-tip"
        type="warning"
        show-icon
        message="本评价包含部分剧透"
      />
      <template v-if="!isSevereSpoiler">
        <p class="review-excerpt">{{ reviewText }}</p>
      </template>
      <div v-else class="review-spoiler">
        <span class="review-spoiler-label">
          该评价包含严重剧透（{{ spoilerLabel(spoilerLevel) }}）
        </span>
        <KunButton
          v-if="!revealed"
          size="sm"
          color="primary"
          variant="flat"
          @click="revealed = true"
        >
          点击查看
        </KunButton>
        <p v-else class="review-excerpt">{{ reviewText }}</p>
      </div>
    </div>

    <div class="rating-game-meta">
      <a-tag :color="spoilerColor(spoilerLevel)" class="spoiler-tag">
        {{ spoilerLabel(spoilerLevel) }}
      </a-tag>
      <span class="game-date">{{ formatDate(rating.updated_at || rating.created_at) }}</span>
    </div>
  </article>
</template>

<style scoped>
.rating-game-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  padding: 14px;
  border: 1px solid var(--app-glass-border);
  border-radius: var(--radius-kun-lg);
  background: var(--app-glass-background);
  transition: border-color var(--kun-dur-fast);
}

.rating-game-card:hover {
  border-color: var(--color-primary);
}

.rating-game-main {
  display: flex;
  gap: 14px;
}

.game-cover {
  display: grid;
  overflow: hidden;
  width: 72px;
  height: 96px;
  flex: 0 0 72px;
  place-items: center;
  border-radius: var(--radius-kun-md);
  background: var(--color-default-100);
  color: var(--color-default-400);
}

.game-info {
  min-width: 0;
}

.game-info h2 {
  margin: 0;
  font-size: 16px;
}

.game-score {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 8px;
  color: var(--color-warning-600, #d97706);
  font-size: 13px;
  font-weight: 600;
}

.game-recommendation {
  margin-top: 6px;
  margin-inline-start: 0;
}

.game-dimensions {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 12px;
  margin: 8px 0 0;
  padding: 0;
  list-style: none;
}

.game-dimensions li {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
  color: var(--color-default-500);
  font-size: 12px;
}

.game-dimensions strong {
  color: var(--color-foreground);
  font-size: 13px;
}

.rating-game-review {
  min-width: 0;
}

.review-spoiler-tip {
  margin-bottom: 8px;
}

.review-excerpt {
  display: -webkit-box;
  overflow: hidden;
  margin: 0;
  color: var(--color-default-500);
  font-size: 13px;
  line-height: 1.65;
  white-space: pre-wrap;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.review-spoiler {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
}

.review-spoiler-label {
  color: var(--color-default-500);
  font-size: 13px;
}

.rating-game-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.spoiler-tag {
  margin-inline-start: 0;
}

.game-date {
  color: var(--color-default-400);
  font-size: 12px;
}
</style>
