<template>
  <AppPageContainer
    title="沙盒"
    description="用于验证通用页面布局和基础交互。"
  >
    <div class="counter-card">
      <span class="counter-label">当前计数</span>
      <strong class="counter-value">{{ count }}</strong>
      <KunButton color="primary" @click="increment">
        <KunIcon name="lucide:plus" />
        增加计数
      </KunButton>
    </div>
  </AppPageContainer>
</template>

<script setup lang="ts">
const count = ref(0)

function increment() {
  count.value++
}
</script>

<style scoped>
.counter-card {
  display: flex;
  min-height: 260px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  padding: 24px;
  border: 1px solid var(--color-kun-border);
  border-radius: var(--radius-kun-lg);
  background: var(--color-content1);
  box-shadow: var(--shadow-kun-sm);
}

.counter-label {
  color: var(--color-default-500);
  font-size: 14px;
}

.counter-value {
  color: var(--color-primary-600);
  font-size: 56px;
  line-height: 1;
}
</style>
