<script setup lang="ts">
import type { GameDescriptions } from '~/types/galgame'
import {
  LOCALE_LABELS,
  SITE_DESCRIPTION_LOCALE,
  getBestDescription,
  normalizeDescriptionLocale
} from '~/utils/gameDescriptions'

const props = withDefaults(
  defineProps<{
    descriptions?: GameDescriptions | null
    locale?: string
  }>(),
  {
    descriptions: null,
    locale: SITE_DESCRIPTION_LOCALE
  }
)

const requestedLocale = computed(() =>
  normalizeDescriptionLocale(props.locale)
)

const best = computed(() =>
  getBestDescription(props.descriptions, requestedLocale.value)
)

const headingLabels: Record<string, string> = {
  'zh-CN': '游戏简介',
  'en-US': 'Description',
  'ja-JP': 'あらすじ'
}

const sourcePrefixes: Record<string, string> = {
  'zh-CN': '来源',
  'en-US': 'Source',
  'ja-JP': '出典'
}

const heading = computed(
  () => headingLabels[requestedLocale.value] ?? headingLabels['zh-CN']
)

const sourcePrefix = computed(
  () => sourcePrefixes[requestedLocale.value] ?? sourcePrefixes['zh-CN']
)

const fallbackNotice = computed(() => {
  if (!best.value?.isFallback) {
    return ''
  }
  const current = LOCALE_LABELS[requestedLocale.value] ?? requestedLocale.value
  const shown = LOCALE_LABELS[best.value.locale] ?? best.value.locale
  return `暂无${current}简介，当前显示${shown}简介`
})

const sourceLinkAttrs = {
  target: '_blank',
  rel: 'noopener noreferrer nofollow'
} as const
</script>

<template>
  <div class="game-description">
    <KunHeader :name="heading" scale="h3" class="section-heading" />
    <PostContent
      v-if="best?.description.content"
      class="game-description-markdown"
      :content="best.description.content"
      mode="markdown"
    />
    <p v-else class="game-description-empty">暂无简介</p>
    <p v-if="fallbackNotice" class="game-description-fallback">
      {{ fallbackNotice }}
    </p>
    <p v-if="best?.description.content" class="game-description-source">
      <span>{{ sourcePrefix }}：</span>
      <a
        v-if="best.description.source.url"
        :href="best.description.source.url"
        v-bind="sourceLinkAttrs"
        class="game-description-source-link"
      >
        {{ best.description.source.name }}
      </a>
      <span v-else>{{ best.description.source.name }}</span>
      <span
        v-if="best.description.source.official"
        class="game-description-official"
      >
        · 官方
      </span>
    </p>
  </div>
</template>

<style scoped>
.game-description-markdown {
  margin-top: 8px;
}

.game-description-empty {
  color: var(--color-default-500);
  font-size: 14px;
}

.game-description-fallback {
  margin-top: 4px;
  color: var(--color-default-400);
  font-size: 12px;
}

.game-description-source {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 2px;
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px solid var(--color-default-100);
  color: var(--color-default-500);
  font-size: 12px;
}

.game-description-source-link {
  color: var(--color-default-500);
  text-decoration: underline;
  text-underline-offset: 2px;
}

.game-description-source-link:hover {
  color: var(--color-default-700);
}

.game-description-official {
  margin-left: 2px;
}
</style>
