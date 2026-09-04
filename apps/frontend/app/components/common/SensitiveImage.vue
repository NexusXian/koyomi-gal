<script setup lang="ts">
const props = defineProps<{
  src?: string | null
  alt?: string
  sensitive?: boolean
}>()

const { showSensitiveCovers } = useSensitiveCover()
const revealed = ref(false)

const shouldBlur = computed(
  () =>
    Boolean(props.sensitive) &&
    !showSensitiveCovers.value &&
    !revealed.value
)

function reveal(event: MouseEvent): void {
  event.preventDefault()
  event.stopPropagation()
  revealed.value = true
}
</script>

<template>
  <div class="sensitive-image" :class="{ 'is-blurred': shouldBlur }">
    <img :src="src || undefined" :alt="alt" loading="lazy" />
    <button
      v-if="shouldBlur"
      type="button"
      class="sensitive-overlay"
      aria-label="显示敏感封面"
      @click="reveal"
    >
      <span class="sensitive-label">敏感封面</span>
      <span class="sensitive-hint">点击查看</span>
    </button>
  </div>
</template>

<style scoped>
.sensitive-image {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.sensitive-image img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition:
    filter 0.25s ease,
    transform 0.25s ease;
}

.sensitive-image.is-blurred img {
  filter: blur(14px);
  transform: scale(1.05);
}

.sensitive-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 0;
  background: rgb(15 13 22 / 22%);
  color: #fff;
  cursor: pointer;
  font: inherit;
}

.sensitive-label {
  padding: 3px 10px;
  border-radius: 999px;
  background: rgb(15 13 22 / 66%);
  font-size: 12px;
  font-weight: 700;
  backdrop-filter: blur(6px);
}

.sensitive-hint {
  font-size: 12px;
  opacity: 0.85;
}

.sensitive-overlay:focus-visible {
  outline: 2px solid #fff;
  outline-offset: -2px;
}
</style>
