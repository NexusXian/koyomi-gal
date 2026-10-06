<script setup lang="ts">
import { listCompanyGames } from '~/api/generated/companies/companies'
import { COMPANY_ROLES } from '~/constants/company'
import { GALGAME_SORTS } from '~/constants/domain'

const props = defineProps<{ slug: string }>()
const page = ref(1)
const pageSize = 20
const role = ref<NonNullable<Parameters<typeof listCompanyGames>[1]>['role']>()
const keywordInput = ref('')
const keyword = ref('')
const sort = ref<NonNullable<Parameters<typeof listCompanyGames>[1]>['sort']>('latest')
watch([role, keyword, sort, () => props.slug], () => { page.value = 1 }, { flush: 'sync' })
const { data, pending, error, refresh } = await useAsyncData(
  computed(() => `company-games-${JSON.stringify([props.slug, page.value, role.value, keyword.value, sort.value])}`),
  async () => unwrapApiData(await listCompanyGames(encodeURIComponent(props.slug), {
    page: page.value, page_size: pageSize, role: role.value,
    keyword: keyword.value.trim() || undefined, sort: sort.value
  }), '加载公司作品失败')
)
const totalPage = computed(() => Math.max(1, Math.ceil((data.value?.total ?? 0) / pageSize)))
watch(data, (result) => {
  if (result?.page === page.value && page.value > totalPage.value) page.value = totalPage.value
})
</script>

<template>
  <section class="company-games">
    <KunHeader name="关联作品" :description="`共 ${data?.total ?? 0} 部游戏`" scale="h2" />
    <div class="games-filters">
      <a-input-search v-model:value="keywordInput" placeholder="搜索作品" allow-clear @search="keyword = keywordInput" />
      <a-select v-model:value="role" :options="[...COMPANY_ROLES]" placeholder="全部角色" allow-clear />
      <a-select v-model:value="sort" :options="GALGAME_SORTS.map((item) => ({ value: item.slug, label: item.label }))" aria-label="作品排序" />
    </div>
    <a-alert v-if="error" type="error" show-icon :message="getApiErrorMessage(error, '加载公司作品失败')">
      <template #action><a-button @click="refresh()">重试</a-button></template>
    </a-alert>
    <div v-else-if="pending" class="games-grid" aria-busy="true"><KunSkeleton v-for="index in 4" :key="index" class="game-skeleton" /></div>
    <a-empty v-else-if="!data?.items?.length" description="没有找到符合条件的作品" />
    <div v-else class="games-grid"><GalgameCard v-for="game in data.items" :key="game.id" :galgame="game" /></div>
    <div v-if="!error && totalPage > 1" class="pagination-row">
      <KunPagination v-model:current-page="page" :total-page="totalPage" :is-loading="pending" />
    </div>
  </section>
</template>

<style scoped>
.company-games { margin-top: 24px; }
.games-filters { display: grid; grid-template-columns: minmax(0, 2fr) repeat(2, minmax(0, 1fr)); gap: 10px; margin: 16px 0; }
.games-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; }
.game-skeleton { min-height: 320px; }
.pagination-row { display: flex; justify-content: center; margin-top: 24px; }
@media (max-width: 900px) { .games-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
@media (max-width: 640px) { .games-filters { grid-template-columns: minmax(0, 1fr); } .games-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
