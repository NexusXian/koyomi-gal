<script setup lang="ts">
import dayjs from 'dayjs'
import type { InternalNovelDtoContributorData as ContributorData } from '~/api/generated/models'

withDefaults(
  defineProps<{
    contributors?: ContributorData[]
    contributorCount?: number
  }>(),
  { contributors: () => [], contributorCount: 0 }
)

function formatDate(value?: string): string {
  return value ? dayjs(value).format('YYYY-MM-DD') : '未知'
}
</script>

<template>
  <div class="novel-contributors">
    <KunNull
      v-if="contributors.length === 0"
      message="暂无贡献者"
      :is-show-sticker="false"
    />
    <div v-else class="contributor-list">
      <div
        v-for="contributor in contributors"
        :key="contributor.user_id"
        class="contributor-item"
      >
        <UserAvatar
          :avatar-url="contributor.avatar_url"
          :username="contributor.username"
          size="sm"
        />
        <div class="contributor-info">
          <UserLink
            :user-id="contributor.user_id"
            :username="contributor.username"
          />
          <span class="contributor-meta">
            {{ contributor.contribution_count }} 次贡献 ·
            {{ formatDate(contributor.last_contributed_at) }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.contributor-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
}

.contributor-item {
  display: flex;
  align-items: center;
  gap: 10px;
}

.contributor-info {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.contributor-meta {
  color: var(--color-default-500);
  font-size: 12px;
}
</style>
