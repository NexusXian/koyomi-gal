<script setup lang="ts">
import type { DtoNotificationData } from '~/api/generated/models'
import { formatDate } from '~/constants/domain'

const props = defineProps<{
  notification: DtoNotificationData
  compact?: boolean
}>()

const emit = defineEmits<{
  select: [notification: DtoNotificationData]
}>()

const actorName = computed(() =>
  props.notification.actor?.display_name || props.notification.actor?.username
)

function select(): void {
  emit('select', props.notification)
}

const icon = computed(() => {
  switch (props.notification.type) {
    case 'post_commented':
    case 'comment_replied':
      return 'lucide:message-circle'
    case 'post_liked':
    case 'comment_liked':
      return 'lucide:thumbs-up'
    case 'galgame_submitted':
    case 'resource_submitted':
      return 'lucide:clipboard-clock'
    case 'galgame_approved':
    case 'resource_approved':
    case 'report_resolved':
      return 'lucide:circle-check'
    case 'galgame_rejected':
    case 'resource_rejected':
    case 'report_rejected':
      return 'lucide:circle-x'
    case 'resource_reported':
    case 'resource_hidden':
    case 'post_moderated':
    case 'comment_moderated':
      return 'lucide:shield-alert'
    default:
      return 'lucide:megaphone'
  }
})

const timeLabel = computed(() => {
  if (!props.notification.created_at) return ''
  const timestamp = new Date(props.notification.created_at).getTime()
  if (Number.isNaN(timestamp)) return props.notification.created_at
  const seconds = Math.max(0, Math.floor((Date.now() - timestamp) / 1000))
  if (seconds < 60) return '刚刚'
  if (seconds < 3600) return `${Math.floor(seconds / 60)} 分钟前`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)} 小时前`
  if (seconds < 604800) return `${Math.floor(seconds / 86400)} 天前`
  return formatDate(props.notification.created_at)
})
</script>

<template>
  <div
    class="notification-item"
    :class="{ unread: !notification.is_read, compact }"
    role="button"
    tabindex="0"
    @click="select"
    @keydown.enter="select"
  >
    <UserLink
      v-if="notification.actor"
      :username="notification.actor.username"
      :display-name="notification.actor.display_name"
      :user-id="notification.actor.id"
    >
      <UserAvatar
        :avatar-url="notification.actor.avatar_url || notification.actor.avatar"
        :display-name="notification.actor.display_name"
        :username="notification.actor.username"
        size="sm"
      />
    </UserLink>
    <span v-else class="notification-icon">
      <KunIcon :name="icon" />
    </span>

    <span class="notification-body">
      <span class="notification-title">{{ notification.title }}</span>
      <span class="notification-content">
        <UserLink
          v-if="notification.actor?.username"
          :username="notification.actor.username"
          :display-name="notification.actor.display_name"
          :user-id="notification.actor.id"
        >
          <strong>{{ actorName }}</strong>
        </UserLink>
        {{ notification.content }}
      </span>
      <span class="notification-time">{{ timeLabel }}</span>
    </span>

    <span v-if="!notification.is_read" class="unread-dot" aria-label="未读" />
  </div>
</template>

<style scoped>
.notification-item {
  display: flex;
  width: 100%;
  align-items: flex-start;
  gap: 12px;
  border: 0;
  border-bottom: 1px solid var(--color-default-200);
  background: transparent;
  padding: 16px;
  color: var(--color-foreground);
  text-align: left;
  cursor: pointer;
  transition: background-color var(--kun-dur-fast);
}

.notification-item:hover,
.notification-item.unread {
  background: color-mix(in srgb, var(--color-primary) 8%, transparent);
}

.notification-item.compact {
  padding: 12px;
}

.notification-icon {
  display: grid;
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  place-items: center;
  border-radius: 50%;
  background: color-mix(in srgb, var(--color-primary) 12%, transparent);
  color: var(--color-primary);
}

.notification-body {
  display: grid;
  min-width: 0;
  flex: 1;
  gap: 3px;
}

.notification-title {
  font-weight: 600;
}

.notification-content,
.notification-time {
  color: var(--color-default-600);
}

.notification-content {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notification-time {
  font-size: 0.75rem;
}

.unread-dot {
  width: 8px;
  height: 8px;
  flex: 0 0 8px;
  margin-top: 6px;
  border-radius: 50%;
  background: var(--color-primary);
}
</style>
