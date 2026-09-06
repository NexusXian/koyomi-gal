<script setup lang="ts">
import { searchAdminCharacters } from '~/api/generated/admin/admin'
import type { DtoCharacterResponse } from '~/api/generated/models'

const props = defineProps<{ modelValue?: DtoCharacterResponse; disabled?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: DtoCharacterResponse | undefined] }>()
const items = ref<DtoCharacterResponse[]>([])
const loading = ref(false)
const error = ref('')
const query = ref('')
const page = ref(1)
const total = ref(0)
let version = 0
let timer: ReturnType<typeof setTimeout> | undefined
let controller: AbortController | undefined

const options = computed(() => {
  const choices = [...items.value]
  if (props.modelValue?.id && !choices.some((item) => item.id === props.modelValue?.id)) {
    choices.unshift(props.modelValue)
  }
  return choices.map((item) => ({
    value: item.id,
    label: `${item.name}${item.original_name ? ` / ${item.original_name}` : ''} (#${item.id})`
  }))
})

async function search(append = false): Promise<void> {
  const current = ++version
  controller?.abort()
  controller = new AbortController()
  loading.value = true
  error.value = ''
  const nextPage = append ? page.value + 1 : 1
  try {
    const data = unwrapApiData(await searchAdminCharacters({
      q: query.value.trim(), page: nextPage, page_size: 20
    }, { signal: controller.signal, cache: 'no-store' }))
    if (current !== version) return
    items.value = append ? [...items.value, ...(data.items ?? [])] : data.items ?? []
    total.value = data.total ?? 0
    page.value = nextPage
  } catch (cause) {
    if (current === version) error.value = getApiErrorMessage(cause, '搜索角色失败')
  } finally {
    if (current === version) loading.value = false
  }
}

function scheduleSearch(value: string): void {
  clearTimeout(timer)
  ++version
  controller?.abort()
  query.value = value.slice(0, 255)
  items.value = []
  total.value = 0
  loading.value = true
  timer = setTimeout(() => void search(), 250)
}

function select(value: unknown): void {
  emit('update:modelValue', items.value.find((item) => item.id === value) ??
    (props.modelValue?.id === value ? props.modelValue : undefined))
}

onMounted(() => void search())
onBeforeUnmount(() => {
  ++version
  clearTimeout(timer)
  controller?.abort()
})
</script>

<template>
  <div class="character-selector">
    <a-select
      :value="modelValue?.id"
      :options="options"
      :loading="loading"
      :disabled="disabled"
      :filter-option="false"
      show-search
      allow-clear
      placeholder="搜索已有角色的名称、原名或来源 ID"
      class="full-width"
      @search="scheduleSearch"
      @change="select"
    >
      <template #notFoundContent>
        <span>{{ loading ? '搜索中...' : error ? '搜索失败' : '未找到角色' }}</span>
      </template>
    </a-select>
    <div v-if="error" role="alert">
      <span class="error-text">{{ error }}</span>
      <a-button size="small" :disabled="disabled" @click="search()">重试</a-button>
    </div>
    <a-button v-else-if="items.length < total" size="small" :loading="loading" :disabled="disabled" @click="search(true)">
      加载更多角色（{{ items.length }} / {{ total }}）
    </a-button>
  </div>
</template>

<style scoped>
.character-selector { display: grid; gap: 8px; }
.full-width { width: 100%; }
.error-text { color: var(--color-danger); }
</style>
