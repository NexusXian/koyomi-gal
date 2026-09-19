<script setup lang="ts">
import type { DtoRatingRecordData } from '~/api/generated/models'
import { USER_STATES, domainLabel, formatDate } from '~/constants/domain'
import {
  RATING_DIMENSIONS,
  recommendationColor,
  recommendationLabel,
  spoilerColor,
  spoilerLabel
} from '~/constants/rating'

const props = defineProps<{
  rating: DtoRatingRecordData
  likePending?: boolean
}>()

const emit = defineEmits<{
  like: [rating: DtoRatingRecordData]
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
    value: props.rating.dimensions?.[dimension.key] ?? null
  })).filter((dimension) => dimension.value !== null)
)

const showRadar = computed(() => ratedDimensions.value.length >= 3)

const reviewText = computed(() => props.rating.review_text?.trim() ?? '')

const spoilerLevel = computed(() => props.rating.spoiler_level ?? 0)

const isSevereSpoiler = computed(() => spoilerLevel.value === 2)

const createdAt = computed(() => formatDate(props.rating.created_at))
</script>

<template>
  <article class="rating-card">
    <div class="rating-card-head">
      <UserLink
        v-if="rating.user?.username"
        :username="rating.user.username"
        :display-name="rating.user.display_name"
        :user-id="rating.user.id"
        class="rating-card-author"
      >
        <UserAvatar
          :avatar-url="rating.user.avatar_url"
          :display-name="rating.user.display_name"
          :username="rating.user.username"
          size="sm"
        />
        <span class="rating-card-username">
          {{ rating.user.display_name || rating.user.username }}
        </span>
      </UserLink>
      <span v-else class="rating-card-username">匿名用户</span>

      <span class="rating-card-overall">
        {{ rating.overall ?? '-' }} <small>/ 10</small>
      </span>
    </div>

    <div class="rating-card-tags">
      <a-tag
        v-if="rating.recommendation !== undefined && rating.recommendation !== null"
        :color="recommendationColor(rating.recommendation)"
      >
        {{ recommendationLabel(rating.recommendation) }}
      </a-tag>
      <a-tag v-if="rating.play_status !== undefined && rating.play_status !== null">
        {{ domainLabel(USER_STATES, rating.play_status) }}
      </a-tag>
      <a-tag :color="spoilerColor(spoilerLevel)">
        {{ spoilerLabel(spoilerLevel) }}
      </a-tag>
    </div>

    <div class="rating-card-body" :class="{ 'with-radar': showRadar }">
      <div v-if="showRadar" class="rating-card-radar">
        <GalgameRatingRadar
          :points="
            RATING_DIMENSIONS.map((dimension) => ({
              label: dimension.label,
              value: rating.dimensions?.[dimension.key] ?? null
            }))
          "
        />
      </div>

      <div class="rating-card-content">
        <ul v-if="ratedDimensions.length" class="rating-card-dimensions">
          <li v-for="dimension in ratedDimensions" :key="dimension.key">
            <span>{{ dimension.label }}</span>
            <strong>{{ dimension.value }}</strong>
          </li>
        </ul>

        <template v-if="reviewText">
          <a-alert
            v-if="spoilerLevel === 1"
            class="rating-card-spoiler-tip"
            type="warning"
            show-icon
            message="本评价包含部分剧透"
          />
          <div v-if="!isSevereSpoiler" class="rating-card-review">
            <PostContent :content="reviewText" mode="markdown" />
          </div>
          <div v-else class="rating-card-spoiler">
            <p class="rating-card-spoiler-text">该评价包含严重剧透</p>
            <KunButton
              v-if="!revealed"
              size="sm"
              color="primary"
              variant="flat"
              @click="revealed = true"
            >
              点击查看
            </KunButton>
            <div v-else class="rating-card-review">
              <PostContent :content="reviewText" mode="markdown" />
            </div>
          </div>
        </template>
        <p v-else class="rating-card-no-review">该用户仅打了分，未写评价</p>
      </div>
    </div>

    <footer class="rating-card-foot">
      <KunButton
        size="sm"
        :color="rating.liked ? 'primary' : 'default'"
        :variant="rating.liked ? 'solid' : 'bordered'"
        :disabled="likePending"
        @click="emit('like', rating)"
      >
        <KunIcon name="lucide:heart" />
        {{ rating.like_count ?? 0 }}
      </KunButton>
      <time class="rating-card-date">{{ createdAt }}</time>
    </footer>
  </article>
</template>

<style scoped>
.rating-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--color-default-200);
  border-radius: var(--radius-kun-lg);
}

.rating-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.rating-card-author {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.rating-card-username {
  overflow: hidden;
  color: var(--color-foreground);
  font-size: 14px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rating-card-overall {
  flex: 0 0 auto;
  font-size: 18px;
  font-weight: 800;
  color: var(--color-primary);
}

.rating-card-overall small {
  font-size: 12px;
  font-weight: 400;
  color: var(--color-default-400);
}

.rating-card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.rating-card-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.rating-card-radar {
  width: 100%;
  max-width: 220px;
  margin-inline: auto;
}

.rating-card-content {
  min-width: 0;
}

.rating-card-dimensions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 14px;
  margin: 0 0 10px;
  padding: 0;
  list-style: none;
}

.rating-card-dimensions li {
  display: inline-flex;
  align-items: baseline;
  gap: 5px;
  font-size: 12px;
  color: var(--color-default-500);
}

.rating-card-dimensions strong {
  color: var(--color-foreground);
  font-size: 13px;
}

.rating-card-spoiler-tip {
  margin-bottom: 10px;
}

.rating-card-spoiler {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
}

.rating-card-spoiler-text {
  margin: 0;
  color: var(--color-default-500);
  font-size: 13px;
}

.rating-card-no-review {
  margin: 0;
  color: var(--color-default-400);
  font-size: 13px;
}

.rating-card-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.rating-card-date {
  color: var(--color-default-400);
  font-size: 12px;
}

@media (min-width: 1024px) {
  .rating-card-body.with-radar {
    flex-direction: row;
  }

  .rating-card-radar {
    flex: 0 0 220px;
    margin-inline: 0;
  }
}
</style>
