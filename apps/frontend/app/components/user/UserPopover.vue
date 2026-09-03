<script setup lang="ts">
import { getPublicUserProfile } from '~/api/generated/users/users'
import type { DtoPublicUserProfile } from '~/api/generated/models'
import { formatDate } from '~/constants/domain'

const props = defineProps<{
  username: string
  active: boolean
}>()

const profile = ref<DtoPublicUserProfile | null>(null)
const loading = ref(false)
const failed = ref(false)

async function load(): Promise<void> {
  if (!props.active || profile.value || loading.value) return
  loading.value = true
  failed.value = false
  try {
    profile.value = unwrapApiData(
      await getPublicUserProfile(props.username),
      '用户资料加载失败'
    )
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}

watch(() => props.active, () => void load(), { immediate: true })
</script>

<template>
  <div class="user-popover" @click.stop>
    <a-skeleton v-if="loading" active avatar :paragraph="{ rows: 2 }" />
    <span v-else-if="failed" class="popover-muted">资料加载失败</span>
    <template v-else-if="profile">
      <div class="popover-head">
        <UserAvatar
          :avatar-url="profile.avatar_url"
          :display-name="profile.display_name"
          :username="profile.username"
          size="lg"
        />
        <div>
          <strong>{{ profile.display_name || profile.username }}</strong>
          <span>@{{ profile.username }}</span>
        </div>
      </div>
      <p v-if="profile.is_restricted" class="popover-muted">
        {{ profile.is_private ? '此用户的个人空间已设为私密' : '登录后可查看此用户的个人空间' }}
      </p>
      <template v-else-if="profile.access?.can_view_profile">
        <p v-if="profile.bio" class="popover-bio">{{ profile.bio }}</p>
        <div class="popover-stats">
          <span v-if="profile.access.can_view_posts"><strong>{{ profile.post_count ?? 0 }}</strong> 帖子</span>
          <span v-if="profile.access.can_view_comments"><strong>{{ profile.comment_count ?? 0 }}</strong> 评论</span>
        </div>
        <div class="popover-meta">
          <span v-if="profile.registered_at">{{ formatDate(profile.registered_at) }} 加入</span>
          <span v-if="profile.access.can_view_location && profile.location">
            <KunIcon name="lucide:map-pin" />{{ profile.location }}
          </span>
        </div>
      </template>
      <NuxtLink class="profile-link" :to="`/user/${encodeURIComponent(username)}`">查看主页</NuxtLink>
    </template>
  </div>
</template>

<style scoped>
.user-popover { width: 280px; padding: 4px; }
.popover-head { display: flex; align-items: center; gap: 12px; }
.popover-head div { display: grid; min-width: 0; gap: 2px; }
.popover-head strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.popover-head span, .popover-muted, .popover-meta { color: var(--color-default-500); font-size: 12px; }
.popover-bio { display: -webkit-box; overflow: hidden; margin: 12px 0 8px; color: var(--color-default-600); font-size: 13px; line-height: 1.55; -webkit-box-orient: vertical; -webkit-line-clamp: 3; }
.popover-muted { display: block; margin: 12px 0 2px; }
.popover-meta { display: flex; flex-wrap: wrap; gap: 10px; }
.popover-meta span { display: inline-flex; align-items: center; gap: 4px; }
.popover-stats { display: flex; gap: 14px; margin: 8px 0; color: var(--color-default-500); font-size: 12px; }
.popover-stats strong { color: var(--color-foreground); }
.profile-link { display: inline-flex; margin-top: 12px; color: var(--color-primary); font-size: 13px; font-weight: 600; }
</style>
