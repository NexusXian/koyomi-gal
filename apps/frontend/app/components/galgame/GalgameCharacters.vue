<script setup lang="ts">
import { listGalgameCharacters } from '~/api/generated/galgames/galgames'
import type { DtoGalgameCharacterResponse } from '~/api/generated/models'

const props = defineProps<{ galgameId: number }>()
const items = ref<DtoGalgameCharacterResponse[]>([])
const revealed = ref<DtoGalgameCharacterResponse[]>([])
const loading = ref(true)
const revealing = ref(false)
const error = ref('')
const confirmation = ref<'all' | 'appearance' | number | null>(null)
let requestVersion = 0
let controller: AbortController | undefined
let active = false

const visibleItems = computed(() => items.value.map((item) =>
  revealed.value.find((entry) => entry.id === item.id) ?? item
))
const ordinary = computed(() => visibleItems.value.filter((item) => !item.appearance_spoiler))
const appearances = computed(() => visibleItems.value.filter((item) => item.appearance_spoiler))
const hasSpoilers = computed(() => items.value.some((item) => item.has_spoiler))
const confirmLabel = computed(() => confirmation.value === 'appearance'
  ? '显示剧透角色'
  : confirmation.value === 'all' || !items.value.find((item) => item.id === confirmation.value)?.appearance_spoiler
    ? '显示剧透内容' : '显示角色')

async function loadSafe(): Promise<void> {
  const version = ++requestVersion
  const id = props.galgameId
  controller?.abort()
  controller = new AbortController()
  revealed.value = []
  items.value = []
  confirmation.value = null
  revealing.value = false
  loading.value = true
  error.value = ''
  try {
    const data = unwrapApiData(await listGalgameCharacters(id, { spoiler: false }, {
      signal: controller.signal,
      cache: 'no-store'
    }))
    if (version === requestVersion && id === props.galgameId) {
      items.value = data.items ?? []
    }
  } catch (cause) {
    if (version === requestVersion) error.value = getApiErrorMessage(cause, '加载登场角色失败')
  } finally {
    if (version === requestVersion) loading.value = false
  }
}

async function reveal(): Promise<void> {
  const scope = confirmation.value
  if (!import.meta.client || scope === null || revealing.value) return
  confirmation.value = null
  const version = ++requestVersion
  const id = props.galgameId
  controller?.abort()
  controller = new AbortController()
  revealing.value = true
  error.value = ''
  try {
    // Spoiler payloads stay out of Nuxt's SSR/hydration and async-data caches.
    const data = unwrapApiData(await listGalgameCharacters(id, { spoiler: true }, {
      signal: controller.signal,
      cache: 'no-store'
    }))
    if (version !== requestVersion || id !== props.galgameId) return
    const selected = new Set(revealed.value.map((item) => item.id))
    revealed.value = (data.items ?? []).filter((item) =>
      selected.has(item.id) || scope === 'all' ||
      (scope === 'appearance' ? item.appearance_spoiler : item.id === scope)
    )
  } catch (cause) {
    if (version === requestVersion) error.value = getApiErrorMessage(cause, '加载剧透内容失败，请重新确认后重试')
  } finally {
    if (version === requestVersion) revealing.value = false
  }
}

watch(() => props.galgameId, () => {
  if (active) void loadSafe()
}, { flush: 'sync' })
onMounted(() => {
  active = true
  void loadSafe()
})
onBeforeUnmount(() => {
  active = false
  ++requestVersion
  controller?.abort()
  revealed.value = []
  confirmation.value = null
})
</script>

<template>
  <KunCard padding="lg" class-name="characters-section">
    <div class="section-head">
      <KunHeader name="登场角色" scale="h3" />
      <div class="actions">
        <KunButton
          v-if="hasSpoilers"
          size="sm"
          variant="bordered"
          :disabled="loading || revealing"
          @click="confirmation = 'all'"
        >显示剧透内容</KunButton>
        <KunButton
          v-if="revealed.length || revealing"
          size="sm"
          variant="light"
          @click="loadSafe"
        >{{ revealing ? '取消显示' : '隐藏剧透内容' }}</KunButton>
      </div>
    </div>
    <div v-if="error" class="error-state" role="alert">
      <p>{{ error }}</p>
      <KunButton size="sm" variant="bordered" @click="loadSafe">重新加载（不含剧透）</KunButton>
    </div>
    <a-spin :spinning="loading || revealing">
      <p v-if="loading" role="status">正在加载登场角色...</p>
      <KunNull v-else-if="!items.length && !error" message="暂无登场角色" />
      <div class="character-grid">
        <GalgameCharacterCard
          v-for="item in ordinary"
          :key="item.id"
          :item="item"
          :revealed="revealed.some((entry) => entry.id === item.id)"
          :disabled="revealing"
          @reveal="confirmation = item.id ?? null"
        />
      </div>
      <section v-if="appearances.length" class="spoiler-section">
        <div class="section-head">
          <KunHeader name="剧透角色" scale="h3" />
          <KunButton size="sm" variant="bordered" :disabled="revealing" @click="confirmation = 'appearance'">
            显示剧透角色
          </KunButton>
        </div>
        <a-alert type="warning" show-icon message="这些角色的登场本身可能涉及剧透，确认后才会请求角色身份和剧透内容。" />
        <div class="character-grid">
          <GalgameCharacterCard
            v-for="item in appearances"
            :key="item.id"
            :item="item"
            :revealed="revealed.some((entry) => entry.id === item.id)"
            :disabled="revealing"
            @reveal="confirmation = item.id ?? null"
          />
        </div>
      </section>
    </a-spin>
    <a-modal
      :open="confirmation !== null"
      title="剧透警告"
      :ok-text="confirmLabel"
      cancel-text="取消"
      @ok="reveal"
      @cancel="confirmation = null"
    >
      <p>继续将请求包含角色身份和剧情剧透的数据，可能影响游玩体验。确定{{ confirmLabel }}吗？</p>
    </a-modal>
  </KunCard>
</template>

<style scoped>
.characters-section { margin-bottom: 18px; }
.section-head, .actions { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; }
.section-head { justify-content: space-between; margin-bottom: 12px; }
.character-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.spoiler-section { margin-top: 20px; }
.spoiler-section .character-grid { margin-top: 14px; }
.error-state { margin-bottom: 14px; color: var(--color-danger); }
@media (max-width: 767px) { .character-grid { grid-template-columns: minmax(0, 1fr); } }
</style>
