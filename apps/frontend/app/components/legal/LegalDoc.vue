<script setup lang="ts">
defineProps<{
  title: string
  description: string
  updatedAt: string
}>()
</script>

<template>
  <AppPageContainer :title="title" :description="description">
    <KunCard class-name="legal-doc">
      <p class="legal-updated">最近更新：{{ updatedAt }}</p>
      <div class="legal-body">
        <slot />
      </div>
    </KunCard>
  </AppPageContainer>
</template>

<style scoped>
.legal-doc {
  max-width: 820px;
}

.legal-updated {
  margin: 0 0 16px;
  color: var(--color-default-500, inherit);
  font-size: 13px;
}

.legal-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
  color: var(--color-foreground);
  font-size: 14px;
  line-height: 1.9;
}

.legal-body :deep(h2) {
  margin: 18px 0 4px;
  font-size: 16px;
  font-weight: 700;
}

.legal-body :deep(ul),
.legal-body :deep(ol) {
  margin: 0;
  padding-left: 20px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.legal-body :deep(a) {
  color: var(--color-primary);
  text-decoration: underline;
}
</style>
