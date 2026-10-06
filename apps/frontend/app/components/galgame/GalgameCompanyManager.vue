<script setup lang="ts">
import { message } from 'ant-design-vue'
import { getAdminGalgame, replaceGameCompanies } from '~/api/generated/admin/admin'
import type { DtoGalgameResponse } from '~/api/generated/models'

const props = defineProps<{ galgameId: number }>()
const { has } = usePermissions()
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const loaded = ref(false)
const companies = ref<NonNullable<DtoGalgameResponse['companies']>>([])
const relations = ref<Partial<NonNullable<Parameters<typeof replaceGameCompanies>[1]['companies']>[number]>[]>([])
let sequence = 0

async function load(): Promise<void> {
  if (!has('company:manage')) return
  const request = ++sequence
  const id = props.galgameId
  loading.value = true
  saving.value = false
  loaded.value = false
  error.value = ''
  try {
    const data = unwrapApiData(await getAdminGalgame(id))
    if (request !== sequence) return
    companies.value = data.companies ?? []
    relations.value = companies.value.map((item) => ({ company_id: item.id, role: item.role }))
    loaded.value = true
  } catch (cause) {
    if (request === sequence) error.value = getApiErrorMessage(cause, '加载公司关联失败')
  } finally {
    if (request === sequence) loading.value = false
  }
}

async function save(): Promise<void> {
  if (!has('company:manage') || !loaded.value || saving.value || loading.value) return
  if (relations.value.some((item) => !item.company_id || !item.role)) {
    message.warning('请选择每一项的公司和角色')
    return
  }
  const keys = relations.value.map((item) => `${item.company_id}:${item.role}`)
  if (new Set(keys).size !== keys.length) {
    message.warning('同一公司不能重复添加相同角色')
    return
  }
  saving.value = true
  const request = ++sequence
  const id = props.galgameId
  try {
    const response = await replaceGameCompanies(id, {
      companies: relations.value.map((item) => ({ company_id: item.company_id!, role: item.role! }))
    })
    if (request !== sequence) return
    companies.value = unwrapApiData(response) ?? []
    relations.value = companies.value.map((item) => ({ company_id: item.id, role: item.role }))
    message.success('公司关联已保存')
  } catch (cause) {
    if (request === sequence) message.error(getApiErrorMessage(cause, '保存公司关联失败'))
  } finally {
    if (request === sequence) saving.value = false
  }
}

watch(() => props.galgameId, () => void load())
onMounted(() => void load())
onBeforeUnmount(() => { ++sequence })
</script>

<template>
  <KunCard v-if="has('company:manage')" padding="lg" class-name="company-manager">
    <KunHeader name="关联公司" description="按公司名称选择开发商、发行商、品牌或共同开发商，独立保存全部公司关联。" scale="h3" />
    <a-alert v-if="error" type="error" show-icon :message="error">
      <template #action><a-button :disabled="loading" @click="load">重试</a-button></template>
    </a-alert>
    <a-spin :spinning="loading">
      <template v-if="loaded">
        <CompanyRelationsEditor v-model="relations" :initial-options="companies" :disabled="saving" />
        <a-button type="primary" :loading="saving" class="save-button" @click="save">保存公司关联</a-button>
      </template>
    </a-spin>
  </KunCard>
</template>

<style scoped>
.company-manager { margin-top: 18px; }
.save-button { margin-top: 14px; }
</style>
