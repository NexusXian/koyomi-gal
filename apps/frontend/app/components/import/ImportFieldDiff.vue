<script setup lang="ts">
import type { TableColumnsType } from 'ant-design-vue'
import type { DtoExternalGameDetail } from '~/api/generated/models'

interface ExistingGalgameLike {
  title?: string
  original_title?: string
  romaji_title?: string
  description?: string
  cover_url?: string
  release_date?: string
  developer?: { id?: number; name?: string } | null
}

const props = defineProps<{
  existing: ExistingGalgameLike
  external: DtoExternalGameDetail
}>()

interface DiffRow {
  key: string
  label: string
  current: string
  incoming: string
  changed: boolean
}

function cover(value?: string): string {
  return value?.trim() ? '已设置' : '未设置'
}

function description(value?: string): string {
  const text = (value ?? '').trim()
  if (!text) {
    return '未设置'
  }
  return text.length > 60 ? `${text.slice(0, 60)}…` : text
}

const rows = computed<DiffRow[]>(() => {
  const external = props.external
  const incoming = [
    { key: 'title', label: '标题', current: props.existing.title ?? '', incoming: external.title ?? '' },
    {
      key: 'original_title',
      label: '原始标题',
      current: props.existing.original_title ?? '',
      incoming: external.original_title ?? ''
    },
    {
      key: 'romaji_title',
      label: '罗马音标题',
      current: props.existing.romaji_title ?? '',
      incoming: external.romaji_title ?? ''
    },
    {
      key: 'developer',
      label: '开发商',
      current: props.existing.developer?.name ?? '',
      incoming: external.developer ?? ''
    },
    {
      key: 'release_date',
      label: '发行日期',
      current: props.existing.release_date ?? '',
      incoming: external.release_date ?? ''
    },
    {
      key: 'cover',
      label: '封面',
      current: cover(props.existing.cover_url),
      incoming: cover(external.cover_url)
    },
    {
      key: 'description',
      label: '简介',
      current: description(props.existing.description),
      incoming: description(external.description)
    }
  ]
  return incoming.map((row) => ({
    ...row,
    changed: row.current.trim() !== row.incoming.trim()
  }))
})

const columns: TableColumnsType = [
  { title: '字段', dataIndex: 'label', width: 110 },
  { title: '站内当前值', dataIndex: 'current', ellipsis: true },
  { title: '外部数据', dataIndex: 'incoming', ellipsis: true },
  { title: '差异', dataIndex: 'changed', width: 80 }
]
</script>

<template>
  <a-table
    :columns="columns"
    :data-source="rows"
    :pagination="false"
    size="small"
    row-key="key"
  >
    <template #bodyCell="{ column, record }">
      <template v-if="column.dataIndex === 'changed'">
        <a-tag v-if="record.changed" color="orange">不同</a-tag>
        <a-tag v-else>一致</a-tag>
      </template>
    </template>
  </a-table>
</template>
