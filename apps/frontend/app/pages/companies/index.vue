<script setup lang="ts">
import { listCompanies } from '~/api/generated/companies/companies'
import type { ListCompaniesParams } from '~/api/generated/models'
import { COMPANY_SORTS, COMPANY_STATUSES } from '~/constants/company'

const route = useRoute()
const router = useRouter()
const requestURL = useRequestURL()
const pageSize = 24
const keywordInput = ref(typeof route.query.keyword === 'string' ? route.query.keyword : '')
const countryInput = ref(typeof route.query.country === 'string' ? route.query.country : '')
const filters = computed(() => ({
  keyword: typeof route.query.keyword === 'string' ? route.query.keyword : undefined,
  country: typeof route.query.country === 'string' ? route.query.country : undefined,
  status: COMPANY_STATUSES.find((item) => item.value === route.query.status)?.value,
  sort: COMPANY_SORTS.find((item) => item.value === route.query.sort)?.value ?? 'created_at',
  page: Math.min(1000000, Math.max(1, Math.floor(Number(route.query.page)) || 1)),
  page_size: pageSize
}))
const { data, pending, error, refresh } = await useAsyncData(
  computed(() => `companies-${JSON.stringify(filters.value)}`),
  async () => unwrapApiData(await listCompanies(filters.value), '加载公司列表失败')
)
const items = computed(() => data.value?.items ?? [])
const totalPage = computed(() => Math.max(1, Math.ceil((data.value?.total ?? 0) / pageSize)))

function filter(changes: Partial<Pick<ListCompaniesParams, 'keyword' | 'country' | 'status' | 'sort' | 'page'>>): void {
  void router.replace({ query: { ...route.query, page: undefined, ...changes } })
}

watch(data, (result) => {
  if (!result || result.page !== filters.value.page) return
  const lastPage = Math.max(1, Math.ceil((result.total ?? 0) / pageSize))
  if (filters.value.page > lastPage) filter({ page: lastPage })
})

watch(() => route.query, () => {
  keywordInput.value = filters.value.keyword ?? ''
  countryInput.value = filters.value.country ?? ''
})

useSeoMeta({
  title: '会社 - Koyomi',
  description: '浏览 Galgame 开发商、发行商、品牌及关联作品。',
  ogTitle: '会社 - Koyomi',
  ogDescription: '浏览 Galgame 开发商、发行商、品牌及关联作品。',
  ogType: 'website',
  ogUrl: () => new URL(route.fullPath, requestURL.origin).href
})
</script>

<template>
  <AppPageContainer title="会社" description="发现 Galgame 开发商、发行商与品牌。" width="wide">
    <KunCard padding="md" class-name="filter-card">
      <div class="filters">
        <a-input-search v-model:value="keywordInput" placeholder="搜索名称或别名" allow-clear @search="filter({ keyword: keywordInput.trim() || undefined })" />
        <a-select :value="filters.status" :options="COMPANY_STATUSES" placeholder="运营状态" allow-clear @change="(value: ListCompaniesParams['status']) => filter({ status: value })" />
        <a-input-search v-model:value="countryInput" placeholder="国家 / 地区" allow-clear @search="filter({ country: countryInput.trim() || undefined })" />
        <a-select :value="filters.sort" :options="COMPANY_SORTS" aria-label="公司排序" @change="(value: ListCompaniesParams['sort']) => filter({ sort: value })" />
      </div>
    </KunCard>
    <a-alert v-if="error" type="error" show-icon :message="getApiErrorMessage(error, '加载公司列表失败')">
      <template #action><a-button @click="refresh()">重试</a-button></template>
    </a-alert>
    <div v-else-if="pending" class="company-grid" aria-busy="true">
      <KunSkeleton v-for="index in 6" :key="index" class="company-skeleton" />
    </div>
    <a-empty v-else-if="!items.length" description="没有找到符合条件的公司，请尝试更换筛选条件。" />
    <template v-else>
      <p class="total">共 {{ data?.total ?? 0 }} 家公司</p>
      <div class="company-grid"><CompanyCard v-for="company in items" :key="company.id" :company="company" /></div>
    </template>
    <div v-if="!error && totalPage > 1" class="pagination-row">
      <KunPagination :current-page="filters.page" :total-page="totalPage" :is-loading="pending" @update:current-page="filter({ page: $event })" />
    </div>
  </AppPageContainer>
</template>

<style scoped>
.filter-card { margin-bottom: 18px; }
.filters { display: grid; grid-template-columns: minmax(0, 2fr) repeat(3, minmax(0, 1fr)); gap: 10px; }
.company-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 14px; }
.company-skeleton { min-height: 190px; }
.total { color: var(--color-default-500); }
.pagination-row { display: flex; justify-content: center; margin-top: 24px; }
@media (max-width: 900px) { .filters, .company-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@media (max-width: 560px) { .filters, .company-grid { grid-template-columns: minmax(0, 1fr); } }
</style>
