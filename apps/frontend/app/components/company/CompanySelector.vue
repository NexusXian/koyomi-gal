<script setup lang="ts">
import { listCompanies } from '~/api/generated/companies/companies'
import { listAdminCompanies } from '~/api/generated/admin/admin'
import type { DtoCompanyResponse } from '~/api/generated/models'

const props = withDefaults(defineProps<{
  modelValue?: number
  initialOptions?: Pick<DtoCompanyResponse, 'id' | 'name'>[]
  excludeId?: number
  admin?: boolean
  disabled?: boolean
  placeholder?: string
}>(), { placeholder: '搜索公司名称', initialOptions: () => [] })
const emit = defineEmits<{ 'update:modelValue': [value: number | undefined] }>()
const results = ref<Pick<DtoCompanyResponse, 'id' | 'name'>[]>([])
const selected = ref<Pick<DtoCompanyResponse, 'id' | 'name'>>()
const loading = ref(false)
const error = ref('')
const keyword = ref('')
let page = 1
let total = 0
let sequence = 0
let timer: ReturnType<typeof setTimeout> | undefined
let failedAppend = false

const options = computed(() => {
  const items = new Map([...(selected.value ? [selected.value] : []), ...props.initialOptions, ...results.value].map((item) => [item.id, item]))
  if (props.modelValue && !items.has(props.modelValue)) items.set(props.modelValue, { id: props.modelValue, name: `公司 #${props.modelValue}` })
  return [...items.values()]
    .filter((item) => item.id && item.id !== props.excludeId)
    .map((item) => ({ value: item.id, label: item.name }))
})

function change(value: unknown): void {
  const id = typeof value === 'number' ? value : undefined
  selected.value = [...results.value, ...props.initialOptions, ...(selected.value ? [selected.value] : [])].find((item) => item.id === id)
  emit('update:modelValue', id)
}

async function load(append = false): Promise<void> {
  clearTimeout(timer)
  const request = ++sequence
  loading.value = true
  error.value = ''
  const nextPage = append ? page + 1 : 1
  try {
    const fetchCompanies = props.admin ? listAdminCompanies : listCompanies
    const data = unwrapApiData(await fetchCompanies({ keyword: keyword.value.trim() || undefined, page: nextPage, page_size: 20, sort: 'name' }))
    if (request !== sequence) return
    results.value = append ? [...results.value, ...(data.items ?? [])] : data.items ?? []
    selected.value = results.value.find((item) => item.id === props.modelValue) ?? selected.value
    page = nextPage
    total = data.total ?? 0
    failedAppend = false
  } catch (cause) {
    if (request === sequence) {
      failedAppend = append
      error.value = getApiErrorMessage(cause, '加载公司失败')
    }
  } finally {
    if (request === sequence) loading.value = false
  }
}

function search(value: string): void {
  keyword.value = value
  ++sequence
  loading.value = true
  error.value = ''
  failedAppend = false
  results.value = []
  page = 1
  total = 0
  clearTimeout(timer)
  timer = setTimeout(() => void load(), 300)
}

function scroll(event: UIEvent): void {
  const target = event.target as HTMLElement
  if (!loading.value && !error.value && results.value.length < total && target.scrollTop + target.clientHeight >= target.scrollHeight - 24) {
    void load(true)
  }
}

onMounted(() => void load())
onBeforeUnmount(() => { clearTimeout(timer); ++sequence })
</script>

<template>
  <div class="company-selector">
    <a-select
      :value="modelValue"
      :options="options"
      :loading="loading"
      :disabled="disabled"
      :placeholder="placeholder"
      :filter-option="false"
      show-search
      allow-clear
      @search="search"
      @popup-scroll="scroll"
      @change="change"
    />
    <span v-if="error" class="selector-error">{{ error }} <a-button type="link" size="small" :disabled="loading || disabled" @click="load(failedAppend)">重试</a-button></span>
  </div>
</template>

<style scoped>
.company-selector { width: 100%; min-width: 0; }
.company-selector :deep(.ant-select) { width: 100%; }
.selector-error { color: var(--color-danger); font-size: 12px; }
</style>
