<script setup lang="ts">
import { SITE_GITHUB_URL } from '~/constants/site'

useSeoMeta({ title: '意见反馈 - Koyomi' })
</script>

<template>
  <AppPageContainer
    title="意见反馈"
    description="对站点有建议、发现了问题，或者有什么想吐槽的，都欢迎告诉我们。"
  >
    <KunCard class-name="feedback-card">
      <FeedbackForm type="feedback" />
      <p class="feedback-note">
        提交按 IP 每小时限 5 次；如果留下联系方式，我们可能会通过它回复你。
        也可以到 <a :href="SITE_GITHUB_URL" target="_blank" rel="noopener noreferrer">GitHub</a> 提 Issue。
      </p>
    </KunCard>
  </AppPageContainer>
</template>

<style scoped>
.feedback-card {
  max-width: 720px;
}

.feedback-note {
  margin: 18px 0 0;
  color: var(--color-default-500, inherit);
  font-size: 13px;
  line-height: 1.8;
}

.feedback-note a {
  color: var(--color-primary);
  text-decoration: underline;
}
</style>
