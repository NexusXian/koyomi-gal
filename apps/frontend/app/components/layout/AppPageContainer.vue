<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    title?: string
    description?: string
    width?: 'default' | 'wide'
  }>(),
  {
    title: '',
    description: '',
    width: 'default'
  }
)

const slots = useSlots()
const hasAside = computed(() => Boolean(slots.aside))
const hasHeader = computed(() => Boolean(props.title || props.description || slots.actions))
</script>

<template>
  <section class="page-container" :class="`page-container-${width}`">
    <header v-if="hasHeader" class="page-header">
      <div class="page-heading">
        <h1 v-if="title" class="page-title">{{ title }}</h1>
        <p v-if="description" class="page-description">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="page-actions">
        <slot name="actions" />
      </div>
    </header>

    <div class="page-grid" :class="{ 'page-grid-with-aside': hasAside }">
      <div class="page-content">
        <slot />
      </div>
      <aside v-if="hasAside" class="page-aside">
        <slot name="aside" />
      </aside>
    </div>
  </section>
</template>

<style scoped>
.page-container {
  width: 100%;
  margin: 0 auto;
}

.page-container-default {
  max-width: 1040px;
}

.page-container-wide {
  max-width: 1232px;
}

.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
}

.page-heading {
  min-width: 0;
}

.page-title {
  margin: 0;
  color: var(--color-foreground);
  font-size: clamp(24px, 3vw, 32px);
  font-weight: 700;
  line-height: 1.25;
  letter-spacing: -0.025em;
}

.page-description {
  max-width: 680px;
  margin: 7px 0 0;
  color: var(--color-default-500);
  font-size: 14px;
  line-height: 1.65;
}

.page-actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
}

.page-grid,
.page-content {
  min-width: 0;
}

.page-aside {
  min-width: 0;
}

@media (min-width: 1024px) {
  .page-grid-with-aside {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 280px;
    align-items: start;
    gap: 20px;
  }

  .page-aside {
    position: sticky;
    top: 88px;
  }
}

@media (max-width: 639px) {
  .page-header {
    align-items: flex-start;
    flex-direction: column;
    gap: 12px;
  }
}
</style>
