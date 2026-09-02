<script setup lang="ts">
import type { DtoNotificationData } from '~/api/generated/models'

defineProps<{
  items: DtoNotificationData[]
  loading: boolean
}>()

defineEmits<{
  select: [notification: DtoNotificationData]
}>()
</script>

<template>
  <div class="notification-dropdown">
    <div class="dropdown-heading">最近通知</div>
    <div v-if="loading" class="dropdown-loading">
      <KunLoading />
    </div>
    <div v-else-if="items.length === 0" class="dropdown-empty">暂无通知</div>
    <NotificationItem
      v-for="item in items"
      v-else
      :key="item.id"
      :notification="item"
      compact
      @select="$emit('select', $event)"
    />
    <KunButton class-name="view-all" variant="light" color="primary" href="/notifications">
      查看全部通知
    </KunButton>
  </div>
</template>

<style scoped>
.notification-dropdown {
  width: min(380px, calc(100vw - 32px));
  max-height: min(560px, 75vh);
  overflow: auto;
}

.dropdown-heading {
  padding: 12px 14px;
  font-weight: 700;
}

.dropdown-loading,
.dropdown-empty {
  display: grid;
  min-height: 100px;
  place-items: center;
  color: var(--color-default-600);
}

:deep(.view-all) {
  width: calc(100% - 24px);
  margin: 12px;
}
</style>
