<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string
    description?: string
    icon?: string
    to?: string
  }>(),
  {
    description: '',
    icon: 'lucide:sparkles',
    to: ''
  }
)
</script>

<template>
  <header class="section-header">
    <div class="section-copy">
      <span class="section-icon" aria-hidden="true">
        <KunIcon :name="icon" />
      </span>
      <div>
        <h2>{{ title }}</h2>
        <p v-if="description">{{ description }}</p>
      </div>
    </div>
    <NuxtLink v-if="to" :to="to" class="section-more">
      查看全部
      <KunIcon name="lucide:arrow-right" />
    </NuxtLink>
  </header>
</template>

<style scoped>
.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.section-copy {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 11px;
}

.section-icon {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 12px;
  background: color-mix(in srgb, var(--color-primary) 12%, transparent);
  color: var(--color-primary-600);
  font-size: 18px;
}

h2 {
  margin: 0;
  font-size: clamp(18px, 2vw, 22px);
  font-weight: 750;
  letter-spacing: -0.02em;
}

p {
  margin: 3px 0 0;
  color: var(--color-default-500);
  font-size: 13px;
}

.section-more {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 5px;
  color: var(--color-default-500);
  font-size: 13px;
  transition: color var(--kun-dur-fast) var(--ease-kun-standard);
}

.section-more:hover {
  color: var(--color-primary);
}
</style>
