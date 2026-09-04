<script setup lang="ts">
const props = defineProps<{
  providers: string[]
  loading?: boolean
}>()

const emit = defineEmits<{
  search: [payload: { provider: string; q: string }]
}>()

const provider = ref(props.providers[0] ?? 'vndb')
const keyword = ref('')

watch(
  () => props.providers,
  (value) => {
    if (value.length > 0 && !value.includes(provider.value)) {
      provider.value = value[0] ?? 'vndb'
    }
  }
)

function submit(): void {
  const q = keyword.value.trim()
  if (!q) {
    return
  }
  emit('search', { provider: provider.value, q })
}
</script>

<template>
  <div class="import-search">
    <a-select
      v-model:value="provider"
      class="import-search-provider"
      :options="providers.map((item) => ({ value: item, label: item.toUpperCase() }))"
    />
    <a-input-search
      v-model:value="keyword"
      class="import-search-input"
      placeholder="输入作品名称，例如 Summer Pockets"
      :loading="loading"
      enter-button="搜索"
      allow-clear
      @search="submit"
    />
  </div>
</template>

<style scoped>
.import-search {
  display: flex;
  gap: 12px;
  width: 100%;
}

.import-search-provider {
  width: 130px;
  flex: none;
}

.import-search-input {
  flex: 1;
}
</style>
