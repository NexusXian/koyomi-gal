<script setup lang="ts">
const highlights = [
  {
    icon: 'lucide:layout-dashboard',
    title: '专注内容',
    description: '清晰的页面层级，让 Galgame 内容始终位于视觉中心。',
    color: 'primary' as const
  },
  {
    icon: 'lucide:panels-top-left',
    title: '灵活布局',
    description: '顶部导航、侧栏与移动端抽屉会跟随屏幕尺寸自然切换。',
    color: 'secondary' as const
  },
  {
    icon: 'lucide:moon-star',
    title: '舒适阅读',
    description: '完整支持明暗主题，在不同环境下都保持舒适观感。',
    color: 'success' as const
  }
]

function openSandbox() {
  navigateTo('/test')
}
</script>

<template>
  <AppPageContainer width="wide">
    <KunCard color="primary" padding="lg" class-name="hero-card">
      <div class="hero-content">
        <div class="hero-copy">
          <KunChip color="primary" variant="flat" size="sm">
            <KunIcon name="lucide:sparkles" />
            Koyomi Gal
          </KunChip>

          <div>
            <h1 class="hero-title">欢迎来到 Koyomi</h1>
            <p class="hero-description">
              一个清爽、安静，专注于 Galgame 内容与交流的空间。
            </p>
          </div>

          <div class="hero-actions">
            <KunButton color="primary" @click="openSandbox">
              打开沙盒
              <KunIcon name="lucide:arrow-right" />
            </KunButton>
            <KunButton color="default" variant="bordered" href="#features">
              了解布局
            </KunButton>
          </div>
        </div>

        <div class="hero-visual" aria-hidden="true">
          <span class="orbit orbit-large" />
          <span class="orbit orbit-small" />
          <span class="hero-mark">
            <KunIcon name="lucide:gamepad-2" />
          </span>
          <span class="spark spark-one"><KunIcon name="lucide:sparkles" /></span>
          <span class="spark spark-two"><KunIcon name="lucide:star" /></span>
        </div>
      </div>
    </KunCard>

    <section id="features" class="features-section" aria-labelledby="features-title">
      <div class="section-heading">
        <KunHeader
          name="基础布局已经就绪"
          description="页面结构由 KunUI 组件构建，可在后续页面中直接复用。"
          scale="h2"
        />
        <KunChip color="success" variant="flat" size="sm">
          <KunIcon name="lucide:circle-check" />
          Ready
        </KunChip>
      </div>

      <div class="feature-grid">
        <KunCard
          v-for="item in highlights"
          :key="item.title"
          :color="item.color"
          padding="md"
          :is-hoverable="true"
        >
          <div class="feature-card">
            <span class="feature-icon">
              <KunIcon :name="item.icon" />
            </span>
            <div>
              <h2>{{ item.title }}</h2>
              <p>{{ item.description }}</p>
            </div>
          </div>
        </KunCard>
      </div>
    </section>
  </AppPageContainer>
</template>

<style scoped>
.hero-card {
  overflow: hidden;
}

.hero-content {
  display: grid;
  min-height: 360px;
  align-items: center;
  gap: 32px;
}

.hero-copy {
  position: relative;
  z-index: 1;
  display: flex;
  max-width: 680px;
  flex-direction: column;
  align-items: flex-start;
  gap: 24px;
}

.hero-title {
  margin: 0;
  font-size: clamp(36px, 6vw, 64px);
  font-weight: 800;
  line-height: 1.08;
  letter-spacing: -0.045em;
}

.hero-description {
  max-width: 560px;
  margin: 16px 0 0;
  color: var(--color-default-600);
  font-size: clamp(16px, 2vw, 19px);
  line-height: 1.75;
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.hero-visual {
  position: relative;
  display: none;
  width: 300px;
  height: 300px;
  place-items: center;
}

.orbit {
  position: absolute;
  border: 1px solid color-mix(in srgb, var(--color-primary) 22%, transparent);
  border-radius: 50%;
}

.orbit-large {
  width: 280px;
  height: 280px;
}

.orbit-small {
  width: 210px;
  height: 210px;
  border-style: dashed;
  transform: rotate(-12deg);
}

.hero-mark {
  display: grid;
  width: 132px;
  height: 132px;
  place-items: center;
  border: 1px solid color-mix(in srgb, var(--color-primary) 25%, transparent);
  border-radius: 38px;
  background: color-mix(in srgb, var(--color-content1) 72%, transparent);
  box-shadow: var(--shadow-kun-lg);
  color: var(--color-primary-600);
  font-size: 58px;
  transform: rotate(5deg);
  backdrop-filter: blur(10px);
}

.spark {
  position: absolute;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: var(--color-content1);
  box-shadow: var(--shadow-kun-md);
  color: var(--color-primary-500);
}

.spark-one {
  top: 42px;
  right: 24px;
  width: 46px;
  height: 46px;
}

.spark-two {
  bottom: 28px;
  left: 34px;
  width: 38px;
  height: 38px;
  color: var(--color-warning-500);
}

.features-section {
  padding-top: 36px;
  scroll-margin-top: 88px;
}

.section-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
}

.feature-grid {
  display: grid;
  gap: 14px;
}

.feature-card {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.feature-icon {
  display: grid;
  width: 44px;
  height: 44px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: var(--radius-kun-md);
  background: color-mix(in srgb, var(--color-primary) 10%, transparent);
  color: var(--color-primary-600);
  font-size: 21px;
}

.feature-card h2 {
  margin: 1px 0 0;
  font-size: 16px;
  font-weight: 700;
}

.feature-card p {
  margin: 7px 0 0;
  color: var(--color-default-500);
  font-size: 14px;
  line-height: 1.7;
}

@media (min-width: 720px) {
  .feature-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (min-width: 900px) {
  .hero-content {
    grid-template-columns: minmax(0, 1fr) auto;
    padding: 12px 24px;
  }

  .hero-visual {
    display: grid;
  }
}

@media (max-width: 479px) {
  .hero-content {
    min-height: 410px;
  }

  .hero-actions,
  .hero-actions :deep(a),
  .hero-actions :deep(button) {
    width: 100%;
  }

  .section-heading {
    flex-direction: column;
    gap: 12px;
  }
}
</style>
