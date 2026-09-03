<script setup lang="ts">
import type { DtoUserActivityData } from '~/api/generated/models'
import { formatDate } from '~/constants/domain'

const props = defineProps<{ activity: DtoUserActivityData }>()

const labels: Record<string, string> = {
  post_created: '发布了帖子', comment_created: '发表了评论', rating_created: '评价了 Galgame',
  favorite_created: '收藏了 Galgame', resource_submitted: '提交了资源', review_approved: '提交的 Galgame 已通过审核'
}

const label = computed(() => labels[props.activity.type ?? ''] || '更新了动态')
const title = computed(() => {
  const metadata = props.activity.metadata
  const value = metadata?.title ?? metadata?.post_title ?? metadata?.target_title ?? metadata?.name
  return typeof value === 'string' ? value : ''
})
const target = computed(() => {
  const metadata = props.activity.metadata
  const postId = typeof metadata?.post_id === 'number' ? metadata.post_id : undefined
  if (props.activity.target_type === 'post') return `/posts/${props.activity.target_id}`
  if (props.activity.target_type === 'comment' && postId) return `/posts/${postId}`
  if (props.activity.target_type === 'resource' && typeof metadata?.galgame_id === 'number') return `/galgames/${metadata.galgame_id}`
  if (['galgame', 'rating', 'favorite'].includes(props.activity.target_type ?? '')) return `/galgames/${props.activity.target_id}`
  return ''
})
</script>

<template>
  <article class="activity-item">
    <span class="activity-icon"><KunIcon name="lucide:activity" /></span>
    <div>
      <p>{{ label }}<NuxtLink v-if="target" :to="target">{{ title || `#${activity.target_id}` }}</NuxtLink><strong v-else-if="title">{{ title }}</strong></p>
      <time>{{ formatDate(activity.created_at) }}</time>
    </div>
  </article>
</template>

<style scoped>
.activity-item { display: flex; gap: 12px; padding: 14px 0; border-bottom: 1px solid var(--color-default-200); }
.activity-item:last-child { border-bottom: 0; }
.activity-icon { display: grid; width: 34px; height: 34px; flex: 0 0 34px; place-items: center; border-radius: 50%; background: color-mix(in srgb, var(--color-primary) 12%, transparent); color: var(--color-primary); }
.activity-item p { margin: 0; color: var(--color-default-600); }
.activity-item a, .activity-item strong { margin-left: 6px; color: var(--color-primary); font-weight: 600; }
.activity-item time { display: block; margin-top: 4px; color: var(--color-default-400); font-size: 12px; }
</style>
