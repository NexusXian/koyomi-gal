<script setup lang="ts">
import { formatDate } from '~/constants/domain'
import type { HomePost } from '~/types/home'

const props = defineProps<{
  post: HomePost
}>()

const router = useRouter()

function authorName(post: HomePost): string {
  return post.author?.display_name || post.author_name || post.author?.username || `用户 #${post.author_id || post.author?.id || '-'}`
}

function galgameTitle(post: HomePost): string {
  return post.galgame_title || post.galgame?.title || ''
}

function openPost(): void {
  void router.push(`/posts/${props.post.id}`)
}
</script>

<template>
  <article class="post-item" role="link" tabindex="0" @click="openPost" @keydown.enter="openPost">
    <div class="post-main">
      <h3>{{ post.title || '未命名帖子' }}</h3>
      <div class="post-context">
        <UserLink
          :username="post.author?.username"
          :display-name="post.author?.display_name || post.author_name"
          :user-id="post.author?.id || post.author_id"
        >
          <UserAvatar
            :avatar-url="post.author?.avatar_url || post.author?.avatar"
            :display-name="post.author?.display_name || post.author_name"
            :username="post.author?.username"
            size="sm"
          />
          {{ authorName(post) }}
        </UserLink>
        <span v-if="galgameTitle(post)" class="galgame-name">
          <KunIcon name="lucide:gamepad-2" />
          {{ galgameTitle(post) }}
        </span>
      </div>
    </div>
    <div class="post-meta">
      <span :title="formatDate(post.created_at)">{{ formatDate(post.created_at) }}</span>
      <span><KunIcon name="lucide:thumbs-up" /> {{ post.like_count ?? 0 }}</span>
      <span><KunIcon name="lucide:message-circle" /> {{ post.comment_count ?? 0 }}</span>
      <span><KunIcon name="lucide:heart" /> {{ post.favorite_count ?? 0 }}</span>
    </div>
  </article>
</template>

<style scoped>
.post-item {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 13px 15px;
  border: 1px solid var(--app-glass-border);
  border-radius: 13px;
  background: var(--app-glass-background);
  -webkit-backdrop-filter: var(--app-glass-filter);
  backdrop-filter: var(--app-glass-filter);
  cursor: pointer;
  transition:
    border-color var(--kun-dur-fast) var(--ease-kun-standard),
    transform var(--kun-dur-fast) var(--ease-kun-standard);
}

.post-item:hover {
  border-color: color-mix(in srgb, var(--color-primary) 35%, transparent);
  transform: translateX(2px);
}

.post-main {
  min-width: 0;
}

h3 {
  overflow: hidden;
  margin: 0;
  font-size: 14px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.post-context,
.post-meta {
  display: flex;
  align-items: center;
  color: var(--color-default-400);
  font-size: 11px;
}

.post-context {
  gap: 12px;
  margin-top: 6px;
}

.post-meta {
  flex: 0 0 auto;
  gap: 10px;
}

.post-context span,
.post-meta span {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 4px;
}

.galgame-name {
  overflow: hidden;
  max-width: 180px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 639px) {
  .post-item {
    align-items: flex-start;
    flex-direction: column;
    gap: 9px;
  }

  .post-meta {
    width: 100%;
    justify-content: space-between;
  }

  .post-meta span:first-child {
    display: none;
  }
}
</style>
