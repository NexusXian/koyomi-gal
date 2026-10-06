<script setup lang="ts">
import { getCompany } from '~/api/generated/companies/companies'
import { companyStatusLabel } from '~/constants/company'
import { stripMarkdownForExcerpt } from '~/utils/markdown'

const route = useRoute()
const requestURL = useRequestURL()
const slug = computed(() => String(route.params.slug))
const { data: company, pending, error, refresh } = await useAsyncData(
  computed(() => `company-${slug.value}`),
  async () => unwrapApiData(await getCompany(encodeURIComponent(slug.value)), '加载公司详情失败')
)
const title = computed(() => `${company.value?.name ?? '公司'} - Koyomi`)
const description = computed(() => stripMarkdownForExcerpt(company.value?.description ?? '', 160) || `${company.value?.name ?? '公司'}的资料与关联 Galgame。`)
const { has } = usePermissions()
const website = computed(() => {
  try {
    const url = new URL(company.value?.website_url ?? '')
    return ['https:', 'http:'].includes(url.protocol) ? url.href : undefined
  } catch { return undefined }
})
useSeoMeta({
  title: () => title.value, description: () => description.value,
  ogTitle: () => title.value, ogDescription: () => description.value,
  ogType: 'website', ogImage: () => company.value?.banner_url || company.value?.logo_url || undefined,
  ogUrl: () => new URL(`/companies/${encodeURIComponent(slug.value)}`, requestURL.origin).href,
  twitterCard: 'summary_large_image'
})
</script>

<template>
  <AppPageContainer :title="company?.name ?? '公司详情'" description="公司资料与关联作品" width="wide">
    <template #actions>
      <KunButton href="/companies" variant="bordered">全部公司</KunButton>
      <KunButton v-if="has('company:manage')" href="/admin/companies" color="primary">管理公司</KunButton>
    </template>
    <KunSkeleton v-if="pending" class="detail-skeleton" />
    <a-alert v-else-if="error || !company" type="error" show-icon :message="getApiErrorMessage(error, '公司不存在或加载失败')">
      <template #action><a-button @click="refresh()">重试</a-button></template>
    </a-alert>
    <template v-else>
      <KunCard padding="lg">
        <img v-if="company.banner_url" :src="company.banner_url" :alt="`${company.name} 横幅`" class="company-banner" />
        <div class="company-heading">
          <img v-if="company.logo_url" :src="company.logo_url" :alt="`${company.name} Logo`" class="company-logo" />
          <div><h1>{{ company.name }}</h1><KunChip variant="flat" size="sm">{{ companyStatusLabel(company.status) }}</KunChip></div>
        </div>
        <dl class="company-metadata">
          <div v-if="company.name_ja"><dt>日文名称</dt><dd>{{ company.name_ja }}</dd></div>
          <div v-if="company.name_zh"><dt>中文名称</dt><dd>{{ company.name_zh }}</dd></div>
          <div v-if="company.name_en"><dt>英文名称</dt><dd>{{ company.name_en }}</dd></div>
          <div v-if="company.country"><dt>国家 / 地区</dt><dd>{{ company.country }}</dd></div>
          <div v-if="company.founded_date"><dt>成立日期</dt><dd>{{ company.founded_date }}</dd></div>
          <div v-if="company.closed_date"><dt>关闭日期</dt><dd>{{ company.closed_date }}</dd></div>
          <div v-if="website"><dt>官方网站</dt><dd><a :href="website" target="_blank" rel="noopener noreferrer nofollow">{{ company.website_url }}</a></dd></div>
          <div v-if="company.parent_company"><dt>母公司</dt><dd><NuxtLink :to="`/companies/${encodeURIComponent(company.parent_company.slug ?? '')}`">{{ company.parent_company.name }}</NuxtLink></dd></div>
          <div><dt>作品统计</dt><dd>全部 {{ company.game_count ?? 0 }} · 开发 {{ company.developer_game_count ?? 0 }} · 发行 {{ company.publisher_game_count ?? 0 }}</dd></div>
          <div v-if="company.created_at"><dt>收录时间</dt><dd>{{ company.created_at.slice(0, 10) }}</dd></div>
          <div v-if="company.updated_at"><dt>更新时间</dt><dd>{{ company.updated_at.slice(0, 10) }}</dd></div>
        </dl>
        <div v-if="company.aliases?.length" class="company-tags"><span>别名：</span><KunChip v-for="alias in company.aliases" :key="alias" size="sm" variant="flat">{{ alias }}</KunChip></div>
        <div v-if="company.children?.length" class="company-tags"><span>旗下公司：</span><NuxtLink v-for="child in company.children" :key="child.id" :to="`/companies/${encodeURIComponent(child.slug ?? '')}`">{{ child.name }}</NuxtLink></div>
        <KunHeader name="公司简介" scale="h2" class="description-heading" />
        <PostContent v-if="company.description" :content="company.description" mode="markdown" />
        <p v-else class="empty-description">暂无公司简介</p>
      </KunCard>
      <CompanyGames :key="slug" :slug="slug" />
    </template>
  </AppPageContainer>
</template>

<style scoped>
.detail-skeleton { min-height: 400px; }
.company-banner { width: 100%; max-height: 300px; object-fit: cover; border-radius: var(--radius-kun-md); margin-bottom: 20px; }
.company-heading { display: flex; align-items: center; gap: 18px; }
.company-logo { width: 96px; height: 96px; object-fit: contain; }
h1 { font-size: 26px; margin: 0 0 10px; overflow-wrap: anywhere; }
.company-metadata { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px 24px; margin: 24px 0; }
.company-metadata > div { display: flex; gap: 12px; }
dt { color: var(--color-default-500); flex-shrink: 0; }
dd { margin: 0; overflow-wrap: anywhere; }
.company-tags { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; margin: 14px 0; }
.description-heading { margin: 24px 0 14px; }
.empty-description { color: var(--color-default-500); }
@media (max-width: 640px) { .company-metadata { grid-template-columns: minmax(0, 1fr); } .company-logo { width: 64px; height: 64px; } }
</style>
