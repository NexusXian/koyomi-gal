<script setup lang="ts">
import type { replaceGameCompanies } from '~/api/generated/admin/admin'
import type { DtoCompanyResponse } from '~/api/generated/models'
import { COMPANY_ROLES, companyRoleLabel } from '~/constants/company'

type Relations = Partial<NonNullable<Parameters<typeof replaceGameCompanies>[1]['companies']>[number]>[]
const props = defineProps<{
  modelValue: Relations
  initialOptions?: Pick<DtoCompanyResponse, 'id' | 'name'>[]
  disabled?: boolean
}>()
const emit = defineEmits<{ 'update:modelValue': [value: Relations] }>()
const rowKeys = new WeakMap<Relations[number], number>()
let nextKey = 0

function rowKey(item: Relations[number]): number {
  const raw = toRaw(item)
  let key = rowKeys.get(raw)
  if (key === undefined) {
    key = nextKey++
    rowKeys.set(raw, key)
  }
  return key
}

function update(index: number, changes: Partial<Relations[number]>): void {
  emit('update:modelValue', props.modelValue.map((item, i) => {
    if (i !== index) return item
    const updated = { ...item, ...changes }
    rowKeys.set(updated, rowKey(item))
    return updated
  }))
}
</script>

<template>
  <div class="relations-editor">
    <div v-for="(item, index) in modelValue" :key="rowKey(item)" class="relation-row">
      <CompanySelector
        :model-value="item.company_id"
        :initial-options="initialOptions"
        :disabled="disabled"
        admin
        :aria-label="`公司 ${index + 1}`"
        @update:model-value="update(index, { company_id: $event })"
      />
      <a-select
        :value="item.role"
        :options="[...COMPANY_ROLES]"
        :disabled="disabled"
        :aria-label="`公司 ${index + 1} 的角色`"
        @change="(value: Relations[number]['role']) => update(index, { role: value })"
      />
      <a-button :disabled="disabled" danger @click="emit('update:modelValue', modelValue.filter((_, i) => i !== index))">
        移除 {{ companyRoleLabel(item.role) }}
      </a-button>
    </div>
    <a-button :disabled="disabled || modelValue.length >= 100" @click="emit('update:modelValue', [...modelValue, { role: 'developer' }])">添加公司</a-button>
  </div>
</template>

<style scoped>
.relations-editor { display: grid; gap: 10px; }
.relation-row { display: grid; grid-template-columns: minmax(0, 1fr) 140px auto; gap: 8px; }
@media (max-width: 640px) {
  .relation-row { grid-template-columns: minmax(0, 1fr) auto; }
  .relation-row > :first-child { grid-column: 1 / -1; }
}
</style>
