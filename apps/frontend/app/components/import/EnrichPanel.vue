<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { listAdminGalgames } from '~/api/generated/admin/admin'
import {
  enrichGalgame,
  listExternalCandidates
} from '~/api/generated/admin-import/admin-import'
import type {
  DtoExternalCandidateItem,
  DtoGalgameListItem
} from '~/api/generated/models'

const ENRICH_PROVIDER = 'bangumi'

const FIELD_OPTIONS = [
  { value: 'title', label: '中文标题', disabled: false },
  { value: 'description', label: '中文简介', disabled: false },
  { value: 'aliases', label: '中文别名', disabled: false },
  { value: 'cover', label: '封面', disabled: false },
  { value: 'tags', label: '标签', disabled: false }
]

const REASON_LABELS: Record<string, string> = {
  original_title_match: '原始标题一致',
  alias_match: '别名一致',
  normalized_title_match: '标题归一化一致',
  release_year_match: '发行年份一致',
  developer_match: '开发商一致'
}

const galgameId = ref<number | null>(null)
const galgameOptions = ref<DtoGalgameListItem[]>([])
const searching = ref(false)
const candidates = ref<DtoExternalCandidateItem[]>([])
const candidatesLoading = ref(false)
const searched = ref(false)
const enrichingId = ref<string | null>(null)
const selectedFields = ref<string[]>([
  'title',
  'description',
  'aliases',
  'cover',
  'tags'
])
const forceUpdate = ref(false)

watch(galgameId, () => {
  candidates.value = []
  searched.value = false
})

async function searchGalgames(keyword: string): Promise<void> {
  if (!keyword.trim()) {
    galgameOptions.value = []
    return
  }
  searching.value = true
  try {
    const data = unwrapApiData(
      await listAdminGalgames({ keyword: keyword.trim(), page: 1, limit: 20 })
    )
    galgameOptions.value = data.items ?? []
  } catch {
    galgameOptions.value = []
  } finally {
    searching.value = false
  }
}

async function match(): Promise<void> {
  if (!galgameId.value) {
    message.warning('请先选择站内作品')
    return
  }
  candidatesLoading.value = true
  try {
    const data = unwrapApiData(
      await listExternalCandidates(galgameId.value, {
        provider: ENRICH_PROVIDER
      })
    )
    candidates.value = data.items ?? []
    searched.value = true
    if ((data.items ?? []).length === 0) {
      message.info('没有找到达到置信度门槛的候选')
    }
  } catch (error) {
    message.error(getApiErrorMessage(error, '匹配候选搜索失败'))
  } finally {
    candidatesLoading.value = false
  }
}

async function enrich(item: DtoExternalCandidateItem): Promise<void> {
  if (!galgameId.value || !item.external_id) {
    return
  }
  enrichingId.value = item.external_id
  try {
    const result = unwrapApiData(
      await enrichGalgame(galgameId.value, {
        provider: ENRICH_PROVIDER,
        external_id: item.external_id,
        fields: selectedFields.value.length > 0 ? selectedFields.value : undefined,
        force: forceUpdate.value
      })
    )
    const updated = (result.updated_fields ?? []).length
    message.success(
      updated > 0
        ? `已关联并补全 ${updated} 个字段`
        : '已关联外部条目（没有需要补全的字段）'
    )
    item.linked = true
  } catch (error) {
    message.error(getApiErrorMessage(error, '关联补全失败'))
  } finally {
    enrichingId.value = null
  }
}

function formatConfidence(value?: number): string {
  return value === null || value === undefined
    ? '-'
    : `${Math.round(value * 100)}%`
}

function reasonText(reasons?: string[]): string {
  return (reasons ?? [])
    .map((reason) => REASON_LABELS[reason] ?? reason)
    .join('、')
}

const columns: TableColumnsType = [
  { title: '封面', key: 'cover', width: 64 },
  { title: '候选条目', key: 'title', ellipsis: true },
  { title: '匹配度', key: 'confidence', width: 90 },
  { title: '匹配依据', key: 'reasons', ellipsis: true },
  { title: '评分', key: 'rating', width: 110 },
  { title: '操作', key: 'actions', width: 130 }
]
</script>

<template>
  <div class="enrich-panel">
    <div class="enrich-panel-form">
      <a-select
        v-model:value="galgameId"
        class="enrich-panel-select"
        show-search
        :filter-option="false"
        placeholder="搜索站内作品，例如 Summer Pockets"
        :options="
          galgameOptions.map((item) => ({
            value: item.id,
            label: `#${item.id} ${item.title ?? ''}`
          }))
        "
        :loading="searching"
        @search="searchGalgames"
      />
      <a-button
        type="primary"
        :disabled="!galgameId"
        :loading="candidatesLoading"
        @click="match"
      >
        搜索匹配
      </a-button>
    </div>

    <template v-if="searched">
      <a-table
        class="enrich-panel-candidates"
        :columns="columns"
        :data-source="candidates"
        :loading="candidatesLoading"
        :pagination="false"
        row-key="external_id"
        size="middle"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'cover'">
            <img
              v-if="record.cover_url"
              :src="record.cover_url"
              class="enrich-panel-cover"
              loading="lazy"
              referrerpolicy="no-referrer"
              alt=""
            />
            <span v-else>-</span>
          </template>
          <template v-else-if="column.key === 'title'">
            <div class="enrich-panel-title-cell">
              <span class="enrich-panel-title-main">
                {{ record.title || record.original_title || '-' }}
                <a-tag v-if="record.linked" color="blue" class="enrich-panel-linked">
                  已关联
                </a-tag>
              </span>
              <span class="enrich-panel-title-sub">
                {{ record.original_title }}
                {{ record.release_date ? ` · ${record.release_date}` : '' }}
              </span>
            </div>
          </template>
          <template v-else-if="column.key === 'confidence'">
            <a-tag
              :color="(record.confidence ?? 0) >= 0.85 ? 'green' : 'orange'"
            >
              {{ formatConfidence(record.confidence) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'reasons'">
            {{ reasonText(record.reasons) || '-' }}
          </template>
          <template v-else-if="column.key === 'rating'">
            <template v-if="record.rating != null">
              {{ record.rating.toFixed(1) }}（{{ record.rating_count ?? 0 }} 票）
            </template>
            <span v-else>暂无</span>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-popconfirm
              title="确认关联该外部条目并补全元数据？"
              ok-text="确认"
              cancel-text="取消"
              @confirm="enrich(record)"
            >
              <a-button
                size="small"
                type="primary"
                :loading="enrichingId === record.external_id"
              >
                关联并补全
              </a-button>
            </a-popconfirm>
          </template>
        </template>
        <template #emptyText>
          <KunNull text="没有匹配候选" />
        </template>
      </a-table>

      <div class="enrich-panel-options">
        <span class="enrich-panel-options-label">补全字段：</span>
        <a-checkbox-group
          v-model:value="selectedFields"
          :options="FIELD_OPTIONS"
        />
        <a-checkbox
          v-model:checked="forceUpdate"
          class="enrich-panel-force"
        >
          强制覆盖已维护数据
        </a-checkbox>
      </div>
      <a-alert
        type="info"
        show-icon
        message="默认只填充空缺字段，不会覆盖站内已维护的标题、简介与封面。"
      />
    </template>
  </div>
</template>

<style scoped>
.enrich-panel-form {
  display: flex;
  gap: 12px;
}

.enrich-panel-select {
  flex: 1;
}

.enrich-panel-candidates {
  margin-top: 14px;
}

.enrich-panel-cover {
  width: 40px;
  height: 56px;
  object-fit: cover;
  border-radius: 4px;
  display: block;
}

.enrich-panel-title-cell {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.enrich-panel-title-main {
  font-weight: 600;
}

.enrich-panel-title-sub {
  font-size: 12px;
  opacity: 0.65;
}

.enrich-panel-linked {
  margin-left: 6px;
}

.enrich-panel-options {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
}

.enrich-panel-options-label {
  font-size: 13px;
}

.enrich-panel-force {
  margin-left: 12px;
}
</style>
