<script setup lang="ts">
import type { DtoPublicUserProfile } from '~/api/generated/models'
import { formatDate } from '~/constants/domain'

defineProps<{ profile: DtoPublicUserProfile }>()

const genderLabels: Record<string, string> = {
  male: '男', female: '女', non_binary: '非二元', undisclosed: '未公开'
}
</script>

<template>
  <KunCard padding="none" class-name="profile-header-card">
    <div class="profile-banner" :style="!profile.is_restricted && profile.banner_url ? { backgroundImage: `url(${profile.banner_url})` } : undefined" />
    <div class="profile-identity">
      <UserAvatar
        class="profile-avatar"
        :avatar-url="profile.avatar_url"
        :display-name="profile.display_name"
        :username="profile.username"
        size="xl"
      />
      <div class="identity-main">
        <h1>{{ profile.display_name || profile.username }}</h1>
        <div class="identity-line">
          <span>@{{ profile.username }}</span>
          <span v-if="profile.id">UID {{ profile.id }}</span>
        </div>
      </div>
      <div v-if="profile.is_self" class="owner-actions">
        <KunButton href="/settings/profile" color="primary" variant="light" size="sm">
          <KunIcon name="lucide:pencil" />编辑资料
        </KunButton>
        <KunButton href="/settings/privacy" color="default" variant="bordered" size="sm">
          <KunIcon name="lucide:lock-keyhole" />隐私设置
        </KunButton>
      </div>
    </div>

    <div v-if="!profile.is_restricted && profile.access?.can_view_profile" class="profile-details">
      <p v-if="profile.bio" class="profile-bio">{{ profile.bio }}</p>
      <div class="detail-list">
        <span v-if="profile.registered_at"><KunIcon name="lucide:calendar-plus" />{{ formatDate(profile.registered_at) }} 注册</span>
        <span v-if="profile.gender && profile.gender !== 'undisclosed'"><KunIcon name="lucide:user-round" />{{ genderLabels[profile.gender] || profile.gender }}</span>
        <span v-if="profile.access.can_view_location && profile.location"><KunIcon name="lucide:map-pin" />{{ profile.location }}</span>
        <span v-if="profile.access.can_view_birthday && profile.birthday"><KunIcon name="lucide:cake" />{{ profile.birthday }}</span>
        <a v-if="profile.website_url" :href="profile.website_url" target="_blank" rel="noopener noreferrer"><KunIcon name="lucide:link" />个人网站</a>
      </div>
    </div>
  </KunCard>
</template>

<style scoped>
:deep(.profile-header-card) { overflow: hidden; }
.profile-banner { height: clamp(130px, 24vw, 240px); background: linear-gradient(135deg, color-mix(in srgb, var(--color-primary) 45%, var(--color-content2)), color-mix(in srgb, var(--color-secondary) 35%, var(--color-content1))); background-position: center; background-size: cover; }
.profile-identity { display: flex; align-items: flex-end; gap: 18px; padding: 0 24px 18px; }
.profile-avatar { margin-top: -46px; border: 4px solid var(--color-content1); }
.identity-main { min-width: 0; flex: 1; padding-top: 14px; }
.identity-main h1 { overflow: hidden; margin: 0; font-size: clamp(22px, 4vw, 30px); text-overflow: ellipsis; white-space: nowrap; }
.identity-line { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 3px; color: var(--color-default-500); font-size: 13px; }
.owner-actions { display: flex; flex-wrap: wrap; gap: 8px; }
.profile-details { padding: 0 24px 22px; }
.profile-bio { margin: 0 0 12px; color: var(--color-default-600); line-height: 1.75; white-space: pre-wrap; }
.detail-list { display: flex; flex-wrap: wrap; gap: 12px 18px; color: var(--color-default-500); font-size: 13px; }
.detail-list span, .detail-list a { display: inline-flex; align-items: center; gap: 5px; }
.detail-list a { color: var(--color-primary); }
@media (max-width: 639px) {
  .profile-identity { align-items: flex-start; flex-wrap: wrap; padding: 0 16px 16px; }
  .profile-avatar { width: 82px; height: 82px; margin-top: -36px; }
  .identity-main { flex-basis: calc(100% - 104px); }
  .owner-actions { width: 100%; }
  .profile-details { padding: 0 16px 18px; }
}
</style>
