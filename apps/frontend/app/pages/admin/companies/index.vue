<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { createCompany, deleteCompany, getAdminCompany, listAdminCompanies, updateCompany } from '~/api/generated/admin/admin'
import type { DtoCompanyResponse } from '~/api/generated/models'
import { COMPANY_SORTS, COMPANY_STATUSES, companyStatusLabel } from '~/constants/company'

useSeoMeta({ title: '会社管理 - Koyomi' })
const { has, load: loadPermissions } = usePermissions()
const { initialize } = useAuth()
const ready = ref(false)
const items = ref<DtoCompanyResponse[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const country = ref('')
const appliedKeyword = ref('')
const appliedCountry = ref('')
const status = ref<NonNullable<Parameters<typeof listAdminCompanies>[0]>['status']>()
const sort = ref<NonNullable<Parameters<typeof listAdminCompanies>[0]>['sort']>('created_at')
const loading = ref(false)
const error = ref('')
const modalOpen = ref(false)
const editing = ref<DtoCompanyResponse | null>(null)
const openingId = ref<number>()
const deletingId = ref<number>()
const saving = ref(false)
let sequence = 0
let disposed = false

function emptyForm() {
  return {
    name: '', name_ja: '', name_zh: '', name_en: '', slug: '', description: '',
    logo_url: '', banner_url: '', website_url: '', country: '', founded_date: '', closed_date: '',
    status: 'active' as NonNullable<DtoCompanyResponse['status']>,
    parent_company_id: undefined as number | undefined, aliases: [] as string[]
  }
}
const form = reactive(emptyForm())
const columns: TableColumnsType = [
  { title: 'ID', dataIndex: 'id', width: 75 },
  { title: 'Logo', dataIndex: 'logo_url', width: 70 },
  { title: '名称', dataIndex: 'name' },
  { title: 'Slug', dataIndex: 'slug' },
  { title: '国家 / 地区', dataIndex: 'country' },
  { title: '状态', dataIndex: 'status', width: 110 },
  { title: '作品', dataIndex: 'game_count', width: 85 },
  { title: '更新时间', dataIndex: 'updated_at', width: 160 },
  { title: '操作', key: 'actions', width: 150 }
]

async function load(): Promise<void> {
  if (!has('company:manage') || disposed) return
  const request = ++sequence
  loading.value = true
  error.value = ''
  try {
    const data = unwrapApiData(await listAdminCompanies({
      keyword: appliedKeyword.value || undefined, country: appliedCountry.value || undefined,
      status: status.value, sort: sort.value, page: page.value, page_size: pageSize.value
    }))
    if (request !== sequence) return
    total.value = data.total ?? 0
    const lastPage = Math.max(1, Math.ceil(total.value / pageSize.value))
    if (page.value > lastPage) {
      page.value = lastPage
      await load()
      return
    }
    items.value = data.items ?? []
  } catch (cause) {
    if (request === sequence) error.value = getApiErrorMessage(cause, '加载公司失败')
  } finally {
    if (request === sequence) loading.value = false
  }
}

function search(): void {
  appliedKeyword.value = keyword.value.trim()
  appliedCountry.value = country.value.trim()
  page.value = 1
  void load()
}

function paginate(next: number, size: number): void {
  page.value = size === pageSize.value ? next : 1
  pageSize.value = size
  void load()
}

function openCreate(): void {
  if (!has('company:manage') || openingId.value !== undefined || saving.value) return
  editing.value = null
  Object.assign(form, emptyForm())
  modalOpen.value = true
}

async function openEdit(id: number): Promise<void> {
  if (!has('company:manage') || openingId.value !== undefined || saving.value || modalOpen.value) return
  openingId.value = id
  try {
    const company = unwrapApiData(await getAdminCompany(id))
    if (disposed) return
    editing.value = company
    Object.assign(form, {
      name: company.name ?? '', name_ja: company.name_ja ?? '', name_zh: company.name_zh ?? '', name_en: company.name_en ?? '',
      slug: company.slug ?? '', description: company.description ?? '', logo_url: company.logo_url ?? '',
      banner_url: company.banner_url ?? '', website_url: company.website_url ?? '', country: company.country ?? '',
      status: company.status ?? 'active', aliases: [...(company.aliases ?? [])],
      parent_company_id: company.parent_company_id ?? undefined,
      founded_date: company.founded_date ?? '', closed_date: company.closed_date ?? ''
    })
    modalOpen.value = true
  } catch (cause) {
    message.error(getApiErrorMessage(cause, '加载公司详情失败'))
  } finally { openingId.value = undefined }
}

async function submit(): Promise<void> {
  if (!has('company:manage') || saving.value) return
  if (!form.name.trim() || !form.slug.trim()) {
    message.warning('请填写名称和 Slug')
    return
  }
  if (form.slug.trim() === '.' || form.slug.trim() === '..' || /[/\\\p{Cc}]/u.test(form.slug)) {
    message.warning('Slug 不能包含路径分隔符、控制字符，也不能为 . 或 ..')
    return
  }
  for (const date of [form.founded_date, form.closed_date]) {
    const parsed = new Date(date)
    if (date && (!/^\d{4}-\d{2}-\d{2}$/.test(date) || Number.isNaN(parsed.getTime()) || parsed.toISOString().slice(0, 10) !== date)) {
      message.warning('日期格式应为 YYYY-MM-DD')
      return
    }
  }
  if (form.founded_date && form.closed_date && form.closed_date < form.founded_date) {
    message.warning('关闭日期不能早于成立日期')
    return
  }
  const payload: Parameters<typeof createCompany>[0] = {
    name: form.name.trim(), name_ja: form.name_ja.trim(), name_zh: form.name_zh.trim(), name_en: form.name_en.trim(),
    slug: form.slug.trim(), description: form.description, logo_url: form.logo_url.trim(), banner_url: form.banner_url.trim(),
    website_url: form.website_url.trim(), country: form.country.trim(), status: form.status,
    founded_date: form.founded_date || undefined, closed_date: form.closed_date || undefined,
    parent_company_id: form.parent_company_id,
    aliases: [...new Set(form.aliases.map((alias) => alias.trim()).filter(Boolean))]
  }
  saving.value = true
  try {
    if (editing.value?.id) await updateCompany(editing.value.id, payload)
    else await createCompany(payload)
    message.success('公司已保存')
    modalOpen.value = false
    await load()
  } catch (cause) {
    message.error(getApiErrorMessage(cause, '保存公司失败'))
  } finally { saving.value = false }
}

async function remove(id: number): Promise<void> {
  if (!has('company:manage') || deletingId.value !== undefined || saving.value) return
  deletingId.value = id
  try {
    await deleteCompany(id)
    message.success('公司已删除')
    if (items.value.length === 1 && page.value > 1) page.value -= 1
    await load()
  } catch (cause) {
    message.error(getApiErrorMessage(cause, '删除公司失败'))
  } finally { deletingId.value = undefined }
}

onMounted(async () => {
  await initialize()
  await loadPermissions()
  if (disposed) return
  ready.value = true
  await load()
})
onBeforeUnmount(() => { disposed = true; ++sequence })
</script>

<template>
  <div>
    <a-spin v-if="!ready" tip="正在加载权限..." />
    <a-alert v-else-if="!has('company:manage')" type="warning" show-icon message="没有公司管理权限" />
    <template v-else>
      <div class="table-toolbar">
        <a-input-search v-model:value="keyword" placeholder="搜索名称或别名" allow-clear @search="search" />
        <a-input-search v-model:value="country" placeholder="国家 / 地区" allow-clear @search="search" />
        <a-select v-model:value="status" :options="COMPANY_STATUSES" placeholder="状态" allow-clear @change="search" />
        <a-select v-model:value="sort" :options="COMPANY_SORTS" aria-label="公司排序" @change="search" />
        <a-button type="primary" :disabled="openingId !== undefined || saving" @click="openCreate">新建公司</a-button>
      </div>
      <a-alert v-if="error" type="error" show-icon :message="error" class="error-alert">
        <template #action><a-button @click="load">重试</a-button></template>
      </a-alert>
       <a-table :columns="columns" :data-source="items" :loading="loading" :pagination="false" :scroll="{ x: 1050 }" row-key="id">
        <template #bodyCell="{ column, record }">
          <NuxtLink v-if="column.dataIndex === 'name'" :to="`/companies/${encodeURIComponent(record.slug)}`">{{ record.name }}</NuxtLink>
          <img v-else-if="column.dataIndex === 'logo_url' && record.logo_url" :src="record.logo_url" :alt="`${record.name} Logo`" class="table-logo" loading="lazy" />
          <template v-else-if="column.dataIndex === 'logo_url'">—</template>
          <template v-else-if="column.dataIndex === 'updated_at'">{{ record.updated_at?.slice(0, 16).replace('T', ' ') ?? '—' }}</template>
          <template v-else-if="column.dataIndex === 'status'">{{ companyStatusLabel(record.status) }}</template>
          <div v-else-if="column.key === 'actions'" class="table-actions">
            <a-button size="small" :loading="openingId === record.id" :disabled="saving || loading || openingId !== undefined" @click="openEdit(record.id)">编辑</a-button>
            <a-popconfirm :title="`确定删除「${record.name}」吗？`" ok-text="删除" cancel-text="取消" @confirm="remove(record.id)">
              <a-button size="small" danger :loading="deletingId === record.id" :disabled="saving || loading || deletingId !== undefined">删除</a-button>
            </a-popconfirm>
          </div>
        </template>
      </a-table>
      <a-pagination :current="page" :page-size="pageSize" :total="total" :disabled="loading" show-size-changer :page-size-options="['20', '50', '100']" class="pagination-row" @change="paginate" />
      <a-modal v-model:open="modalOpen" :title="editing ? '编辑公司' : '新建公司'" :confirm-loading="saving" :cancel-button-props="{ disabled: saving }" :mask-closable="!saving" :closable="!saving" :keyboard="!saving" :width="800" ok-text="保存" cancel-text="取消" @ok="submit">
        <a-form layout="vertical" :disabled="saving">
          <div class="form-grid">
            <a-form-item label="名称" required><a-input v-model:value="form.name" :maxlength="255" /></a-form-item>
            <a-form-item label="Slug" required><a-input v-model:value="form.slug" :maxlength="255" placeholder="唯一的公司网址标识" /></a-form-item>
            <a-form-item label="日文名称"><a-input v-model:value="form.name_ja" :maxlength="255" /></a-form-item>
            <a-form-item label="中文名称"><a-input v-model:value="form.name_zh" :maxlength="255" /></a-form-item>
            <a-form-item label="英文名称"><a-input v-model:value="form.name_en" :maxlength="255" /></a-form-item>
            <a-form-item label="状态" required><a-select v-model:value="form.status" :options="COMPANY_STATUSES" /></a-form-item>
            <a-form-item label="国家 / 地区"><a-input v-model:value="form.country" :maxlength="100" /></a-form-item>
            <a-form-item label="母公司"><CompanySelector v-model="form.parent_company_id" :exclude-id="editing?.id" :initial-options="editing?.parent_company ? [editing.parent_company] : []" :disabled="saving" admin placeholder="搜索母公司名称" /></a-form-item>
            <a-form-item label="成立日期"><a-input v-model:value="form.founded_date" type="date" /></a-form-item>
            <a-form-item label="关闭日期"><a-input v-model:value="form.closed_date" type="date" /></a-form-item>
            <a-form-item label="Logo 地址"><a-input v-model:value="form.logo_url" placeholder="https://" /></a-form-item>
            <a-form-item label="横幅地址"><a-input v-model:value="form.banner_url" placeholder="https://" /></a-form-item>
          </div>
          <a-form-item label="官方网站"><a-input v-model:value="form.website_url" placeholder="https://" /></a-form-item>
          <a-form-item label="别名"><KunTagInput v-model="form.aliases" :max-tags="100" :disabled="saving" placeholder="输入别名后回车" /></a-form-item>
          <a-form-item label="简介（Markdown）"><a-textarea v-model:value="form.description" :rows="8" /></a-form-item>
        </a-form>
      </a-modal>
    </template>
  </div>
</template>

<style scoped>
.table-toolbar { display: grid; grid-template-columns: minmax(160px, 2fr) minmax(120px, 1fr) repeat(2, minmax(100px, 1fr)) auto; gap: 10px; margin-bottom: 14px; }
.table-actions { display: flex; gap: 6px; }
.table-logo { width: 40px; height: 40px; object-fit: contain; }
.error-alert { margin-bottom: 14px; }
.pagination-row { display: flex; justify-content: flex-end; margin-top: 18px; }
.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 18px; }
@media (max-width: 640px) { .table-toolbar, .form-grid { grid-template-columns: minmax(0, 1fr); } }
</style>
