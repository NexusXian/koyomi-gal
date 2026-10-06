<script setup lang="ts">
import type { DtoCompanyResponse } from '~/api/generated/models'
import { companyStatusLabel } from '~/constants/company'
import { stripMarkdownForExcerpt } from '~/utils/markdown'

defineProps<{ company: DtoCompanyResponse }>()
</script>

<template>
  <KunCard padding="md" class-name="company-card">
    <NuxtLink :to="`/companies/${encodeURIComponent(company.slug ?? '')}`" class="company-link">
      <img v-if="company.logo_url" :src="company.logo_url" :alt="`${company.name} Logo`" loading="lazy" class="company-logo" />
      <KunIcon v-else name="lucide:building-2" class="company-logo placeholder" />
      <h2>{{ company.name }}</h2>
    </NuxtLink>
    <div class="company-meta">
      <KunChip size="sm" variant="flat">{{ companyStatusLabel(company.status) }}</KunChip>
      <span v-if="company.country">{{ company.country }}</span>
      <span>{{ company.game_count ?? 0 }} 部游戏</span>
    </div>
    <p>{{ stripMarkdownForExcerpt(company.description ?? '', 120) || '暂无简介' }}</p>
  </KunCard>
</template>

<style scoped>
.company-link { display: flex; align-items: center; gap: 12px; color: inherit; }
.company-logo { width: 56px; height: 56px; object-fit: contain; flex-shrink: 0; border-radius: var(--radius-kun-md); }
.placeholder { padding: 10px; color: var(--color-default-400); }
h2 { margin: 0; font-size: 18px; overflow-wrap: anywhere; }
.company-meta { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; margin-top: 14px; font-size: 13px; color: var(--color-default-500); }
p { margin: 12px 0 0; color: var(--color-default-500); line-height: 1.7; overflow-wrap: anywhere; }
</style>
