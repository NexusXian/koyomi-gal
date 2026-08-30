<template>
  <AppPageContainer
    title="欢迎来到 Koyomi"
    description="一个清爽、专注于 Galgame 内容的空间。"
  >
    <div class="welcome-card">
      <div class="welcome-icon">
        <KunIcon name="lucide:sparkles" />
      </div>
      <div>
        <h2>基础布局已经就绪</h2>
        <p>顶部导航、响应式侧栏、移动端抽屉和页面容器可以在其他页面中直接复用。</p>
      </div>
      <KunButton color="primary" @click="testPage">
        打开沙盒
        <KunIcon name="lucide:arrow-right" />
      </KunButton>
    </div>
  </AppPageContainer>
</template>

<script setup lang="ts">
function testPage() {
  navigateTo('/test')
}
</script>

<style scoped>
.welcome-card {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 18px;
  padding: 24px;
  border: 1px solid var(--color-kun-border);
  border-radius: var(--radius-kun-lg);
  background: var(--color-content1);
  box-shadow: var(--shadow-kun-sm);
}

.welcome-icon {
  display: grid;
  width: 48px;
  height: 48px;
  place-items: center;
  border-radius: var(--radius-kun-md);
  background: var(--color-primary-50);
  color: var(--color-primary-600);
  font-size: 24px;
}

h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 650;
}

p {
  margin: 6px 0 0;
  color: var(--color-default-500);
  font-size: 14px;
  line-height: 1.65;
}

@media (max-width: 639px) {
  .welcome-card {
    grid-template-columns: auto 1fr;
    padding: 18px;
  }

  .welcome-card :deep(button) {
    grid-column: 1 / -1;
    width: 100%;
  }
}
</style>
