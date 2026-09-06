<script setup lang="ts">
import type { LeveldtoUserLevelData } from '~/api/generated/models'

const props = withDefaults(
  defineProps<{
    level: LeveldtoUserLevelData | null | undefined
    loading?: boolean
  }>(),
  { loading: false }
)

const percent = computed(() => {
  if (!props.level) {
    return 0
  }
  return Math.min(100, Math.round((props.level.progress ?? 0) * 1000) / 10)
})

const expLabel = computed(() => {
  if (!props.level) {
    return ''
  }
  if (props.level.is_max_level) {
    return `${props.level.total_exp ?? 0} EXP · 已达最高等级`
  }
  return `${props.level.total_exp ?? 0} / ${props.level.next_level_exp ?? 0} EXP`
})

const remainingLabel = computed(() => {
  if (!props.level || props.level.is_max_level) {
    return ''
  }
  return `距离 LV${props.level.next_level ?? '?'} ${props.level.next_level_name ?? ''} 还需要 ${props.level.remaining_exp ?? 0} EXP`
})
</script>

<template>
  <div class="user-level-progress">
    <div class="user-level-progress-head">
      <UserLevelBadge
        :level="level?.level ?? 1"
        :name="level?.level_name ?? ''"
        :color="level?.color"
        :icon-url="level?.icon_url"
        size="md"
      />
      <span class="user-level-progress-exp">{{ expLabel }}</span>
    </div>
    <a-tooltip :title="`${percent}%`" placement="top">
      <div class="user-level-progress-track">
        <div class="user-level-progress-bar" :style="{ width: `${percent}%` }" />
      </div>
    </a-tooltip>
    <p v-if="remainingLabel" class="user-level-progress-remaining">{{ remainingLabel }}</p>
  </div>
</template>

<style scoped>
.user-level-progress {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.user-level-progress-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.user-level-progress-exp {
  color: var(--color-foreground-500);
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.user-level-progress-track {
  overflow: hidden;
  width: 100%;
  height: 10px;
  border: 1px solid var(--app-glass-border);
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-primary) 10%, transparent);
}

.user-level-progress-bar {
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, var(--color-primary-400), var(--color-primary-600));
  transition: width 0.3s ease;
}

.user-level-progress-remaining {
  margin: 0;
  color: var(--color-foreground-500);
  font-size: 12px;
}
</style>
