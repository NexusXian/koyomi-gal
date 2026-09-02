<script setup lang="ts">
import { SITE_GITHUB_URL } from '~/constants/site'

const year = new Date().getFullYear()

const siteLinks = [
  { label: 'Galgame', to: '/galgames' },
  { label: '资讯', to: '/articles' },
  { label: '社区', to: '/posts' }
]

const legalLinks = [
  { label: '用户协议', to: '/agreement' },
  { label: '隐私政策', to: '/privacy' },
  { label: '内容规范', to: '/guidelines' },
  { label: '版权投诉', to: '/copyright' }
]
</script>

<template>
  <footer class="app-footer">
    <div class="footer-inner">
      <NuxtLink class="footer-brand" to="/">
        <KunIcon name="lucide:flower-2" />
        Koyomi Gal
      </NuxtLink>

      <nav class="footer-links" aria-label="站点导航">
        <NuxtLink v-for="link in siteLinks" :key="link.to" :to="link.to">
          {{ link.label }}
        </NuxtLink>
      </nav>

      <nav class="footer-links" aria-label="项目相关">
        <NuxtLink to="/changelog">更新日志</NuxtLink>
        <NuxtLink to="/feedback">意见反馈</NuxtLink>
        <a :href="SITE_GITHUB_URL" target="_blank" rel="noopener noreferrer">
          <KunIcon name="lucide:github" />
          GitHub
        </a>
      </nav>

      <nav class="footer-links" aria-label="条款与规范">
        <NuxtLink v-for="link in legalLinks" :key="link.to" :to="link.to">
          {{ link.label }}
        </NuxtLink>
      </nav>

      <p class="footer-copyright">© {{ year }} Koyomi Gal</p>
    </div>
  </footer>
</template>

<style scoped>
.app-footer {
  margin-top: 16px;
  border-top: 1px solid var(--app-glass-border);
  background: var(--color-content1, transparent);
  padding: 18px 14px 22px;
}

.footer-inner {
  max-width: 1360px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  text-align: center;
}

.footer-brand {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--color-foreground);
  font-size: 15px;
  font-weight: 700;
}

.footer-brand:hover {
  color: var(--color-primary);
}

.footer-links {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 6px 18px;
}

.footer-links a {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--color-default-500, inherit);
  font-size: 13px;
}

.footer-links a:hover {
  color: var(--color-primary);
}

.footer-copyright {
  margin: 2px 0 0;
  color: var(--color-default-400, inherit);
  font-size: 12px;
}
</style>
