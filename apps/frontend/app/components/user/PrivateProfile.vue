<script setup lang="ts">
defineProps<{ isPrivate?: boolean }>()
</script>

<template>
  <KunCard padding="lg">
    <div class="private-profile">
      <span><KunIcon :name="isPrivate ? 'lucide:lock-keyhole' : 'lucide:log-in'" /></span>
      <h2>{{ isPrivate ? '此用户的个人空间已设为私密' : '登录后可查看此用户的个人空间' }}</h2>
      <p>{{ isPrivate ? '该用户未公开个人资料和社区活动。' : '该用户仅向已登录成员展示个人资料。' }}</p>
      <KunButton v-if="!isPrivate" href="/login" color="primary">前往登录</KunButton>
    </div>
  </KunCard>
</template>

<style scoped>
.private-profile { display: grid; min-height: 240px; place-items: center; align-content: center; gap: 10px; text-align: center; }
.private-profile > span { display: grid; width: 58px; height: 58px; place-items: center; border-radius: 50%; background: color-mix(in srgb, var(--color-primary) 12%, transparent); color: var(--color-primary); font-size: 26px; }
.private-profile h2, .private-profile p { margin: 0; }
.private-profile h2 { font-size: 18px; }
.private-profile p { color: var(--color-default-500); font-size: 14px; }
</style>
