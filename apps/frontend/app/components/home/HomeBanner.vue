<script setup lang="ts">
import type { Component } from 'vue'
import type { HomeBanner } from '~/types/home'

const props = defineProps<{
  banners: HomeBanner[]
}>()

const activeIndex = ref(0)
const paused = ref(false)
const reducedMotion = ref(false)
let timer: ReturnType<typeof setInterval> | undefined
let touchStartX = 0
let touchStartY = 0
let motionQuery: MediaQueryList | undefined
let motionChangeHandler: (() => void) | undefined

const current = computed(() => props.banners[activeIndex.value])

const currentLink = computed(() => {
  const banner = current.value
  if (!banner?.link_type || !banner.link_value) {
    return null
  }

  const value = String(banner.link_value).trim()
  const internalTypes: Record<string, string> = {
    galgame: '/galgames/',
    post: '/posts/',
    news: '/articles/',
    article: '/articles/'
  }
  const prefix = internalTypes[banner.link_type]
  if (prefix && /^\d+$/.test(value)) {
    return { external: false, href: `${prefix}${value}` }
  }

  if (['external', 'url'].includes(banner.link_type)) {
    try {
      const url = new URL(value)
      if (url.protocol === 'http:' || url.protocol === 'https:') {
        return { external: true, href: url.toString() }
      }
    } catch {
      return null
    }
  }

  return null
})

const linkComponent = computed<Component | string>(() =>
  currentLink.value?.external ? 'a' : resolveComponent('NuxtLink')
)

function show(index: number): void {
  const count = props.banners.length
  if (!count) return
  activeIndex.value = (index + count) % count
}

function next(): void {
  show(activeIndex.value + 1)
}

function showManually(index: number): void {
  show(index)
  startAutoplay()
}

function nextManually(): void {
  showManually(activeIndex.value + 1)
}

function previousManually(): void {
  showManually(activeIndex.value - 1)
}

function startAutoplay(): void {
  stopAutoplay()
  if (props.banners.length > 1 && !reducedMotion.value) {
    timer = setInterval(() => {
      if (!paused.value) next()
    }, 6000)
  }
}

function stopAutoplay(): void {
  if (timer) clearInterval(timer)
  timer = undefined
}

function onTouchStart(event: TouchEvent): void {
  touchStartX = event.changedTouches[0]?.clientX ?? 0
  touchStartY = event.changedTouches[0]?.clientY ?? 0
}

function onTouchEnd(event: TouchEvent): void {
  const touch = event.changedTouches[0]
  const distanceX = (touch?.clientX ?? touchStartX) - touchStartX
  const distanceY = (touch?.clientY ?? touchStartY) - touchStartY
  if (Math.abs(distanceX) < 45 || Math.abs(distanceX) <= Math.abs(distanceY)) return
  distanceX < 0 ? nextManually() : previousManually()
}

watch(
  () => props.banners.length,
  () => {
    if (activeIndex.value >= props.banners.length) activeIndex.value = 0
    if (import.meta.client) startAutoplay()
  }
)

onMounted(() => {
  motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  const updateMotion = (): void => {
    reducedMotion.value = motionQuery?.matches ?? false
    startAutoplay()
  }
  motionChangeHandler = updateMotion
  motionQuery.addEventListener('change', updateMotion)
  updateMotion()
})
onBeforeUnmount(() => {
  stopAutoplay()
  if (motionChangeHandler) motionQuery?.removeEventListener('change', motionChangeHandler)
})
</script>

<template>
  <section
    class="banner"
    aria-label="精选内容"
    @mouseenter="paused = true"
    @mouseleave="paused = false"
    @focusin="paused = true"
    @focusout="paused = false"
    @touchstart.passive="onTouchStart"
    @touchend.passive="onTouchEnd"
  >
    <template v-if="current">
      <Transition name="banner-fade" mode="out-in">
        <component
          :is="currentLink ? linkComponent : 'div'"
          :key="current.id"
          class="banner-slide"
          :to="currentLink && !currentLink.external ? currentLink.href : undefined"
          :href="currentLink?.external ? currentLink.href : undefined"
          :target="currentLink?.external ? '_blank' : undefined"
          :rel="currentLink?.external ? 'noopener noreferrer nofollow' : undefined"
        >
          <picture v-if="current.image_url">
            <img :src="current.image_url" :alt="current.title" fetchpriority="high" />
          </picture>
          <div class="banner-shade" />
          <div class="banner-copy">
            <span class="banner-kicker">KOYOMI GAL SELECTION</span>
            <h1>{{ current.title }}</h1>
            <p v-if="current.subtitle">{{ current.subtitle }}</p>
          </div>
        </component>
      </Transition>

      <template v-if="banners.length > 1">
        <button class="banner-arrow banner-arrow-left" type="button" aria-label="上一张" @click="previousManually">
          <KunIcon name="lucide:chevron-left" />
        </button>
        <button class="banner-arrow banner-arrow-right" type="button" aria-label="下一张" @click="nextManually">
          <KunIcon name="lucide:chevron-right" />
        </button>
        <div class="banner-indicators" aria-label="选择轮播图">
          <button
            v-for="(banner, index) in banners"
            :key="banner.id"
            type="button"
            :class="{ active: index === activeIndex }"
            :aria-label="`显示第 ${index + 1} 张轮播图`"
            :aria-current="index === activeIndex ? 'true' : undefined"
            @click="showManually(index)"
          />
        </div>
      </template>
    </template>

    <div v-else class="banner-empty">
      <div>
        <span>KOYOMI GAL</span>
        <h1>发现故事，分享感动</h1>
        <p>和同好一起记录每一部值得珍藏的 Galgame。</p>
      </div>
    </div>
  </section>
</template>

<style scoped>
.banner {
  position: relative;
  width: 100%;
  max-height: 420px;
  aspect-ratio: 21 / 9;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--color-primary) 16%, transparent);
  border-radius: 24px;
  background:
    radial-gradient(circle at 75% 20%, rgb(198 168 255 / 32%), transparent 35%),
    linear-gradient(135deg, #252039, #625484);
  box-shadow: 0 24px 64px rgb(42 28 68 / 18%);
}

.banner-slide,
.banner-empty {
  position: absolute;
  inset: 0;
  display: block;
}

.banner-slide picture,
.banner-slide img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.banner-shade {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(90deg, rgb(12 9 22 / 80%) 0%, rgb(17 12 29 / 45%) 50%, transparent 78%),
    linear-gradient(0deg, rgb(9 7 16 / 58%), transparent 55%);
}

.banner-copy {
  position: absolute;
  z-index: 1;
  bottom: clamp(34px, 7vw, 74px);
  left: clamp(24px, 6vw, 74px);
  width: min(630px, 68%);
  color: #fff;
  text-shadow: 0 2px 18px rgb(0 0 0 / 35%);
}

.banner-kicker,
.banner-empty span {
  color: #e8d8ff;
  font-size: 11px;
  font-weight: 750;
  letter-spacing: 0.16em;
}

.banner-copy h1,
.banner-empty h1 {
  margin: 9px 0 0;
  background: linear-gradient(100deg, #fff 5%, #f2dfff 54%, #d9ecff 100%);
  background-clip: text;
  color: transparent;
  font-size: clamp(25px, 4vw, 49px);
  font-weight: 800;
  line-height: 1.12;
  letter-spacing: -0.035em;
}

.banner-copy p,
.banner-empty p {
  display: -webkit-box;
  overflow: hidden;
  margin: 11px 0 0;
  color: rgb(255 255 255 / 82%);
  font-size: clamp(13px, 1.6vw, 17px);
  line-height: 1.6;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.banner-empty {
  display: grid;
  place-items: center start;
  padding: clamp(25px, 6vw, 74px);
  color: #fff;
}

.banner-arrow {
  position: absolute;
  z-index: 2;
  top: 50%;
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  border: 1px solid rgb(255 255 255 / 25%);
  border-radius: 50%;
  background: rgb(15 11 27 / 40%);
  color: #fff;
  cursor: pointer;
  opacity: 0;
  transform: translateY(-50%);
  transition:
    opacity var(--kun-dur-fast) var(--ease-kun-standard),
    background var(--kun-dur-fast) var(--ease-kun-standard);
  backdrop-filter: blur(8px);
}

.banner:hover .banner-arrow,
.banner:focus-within .banner-arrow {
  opacity: 1;
}

.banner-arrow:hover {
  background: rgb(15 11 27 / 68%);
}

.banner-arrow-left { left: 15px; }
.banner-arrow-right { right: 15px; }

.banner-indicators {
  position: absolute;
  z-index: 2;
  right: 22px;
  bottom: 18px;
  display: flex;
  gap: 6px;
}

.banner-indicators button {
  width: 7px;
  height: 7px;
  padding: 0;
  border: 0;
  border-radius: 999px;
  background: rgb(255 255 255 / 48%);
  cursor: pointer;
  transition: width var(--kun-dur-fast) var(--ease-kun-standard);
}

.banner-indicators button.active {
  width: 24px;
  background: #fff;
}

.banner-fade-enter-active,
.banner-fade-leave-active { transition: opacity 220ms ease; }
.banner-fade-enter-from,
.banner-fade-leave-to { opacity: 0; }

@media (max-width: 639px) {
  .banner {
    max-height: none;
    aspect-ratio: 16 / 9;
    border-radius: 18px;
  }

  .banner-copy {
    bottom: 28px;
    left: 20px;
    width: calc(100% - 40px);
  }

  .banner-kicker { display: none; }
  .banner-copy h1 { font-size: clamp(22px, 7vw, 30px); }
  .banner-copy p { margin-top: 7px; }
  .banner-arrow { display: none; }
  .banner-indicators { right: 15px; bottom: 12px; }
}

@media (prefers-reduced-motion: reduce) {
  .banner-fade-enter-active,
  .banner-fade-leave-active { transition: none; }
}
</style>
