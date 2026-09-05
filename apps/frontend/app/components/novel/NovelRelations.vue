<script setup lang="ts">
import { message } from 'ant-design-vue'
import { listGalgames } from '~/api/generated/galgames/galgames'
import {
  createNovelRelation,
  deleteNovelRelation,
  listNovelRelations
} from '~/api/generated/novels/novels'
import type {
  DtoGalgameListItem,
  DtoRelationData
} from '~/api/generated/models'
import { NOVEL_RELATION_TYPES } from '~/constants/domain'

const props = defineProps<{
  novelId: number
}>()

const emit = defineEmits<{
  changed: []
}>()

const relations = ref<DtoRelationData[]>([])
const loading = ref(false)
const submitting = ref(false)

const targetType = ref<'galgame' | 'novel'>('galgame')
const targetId = ref<number | undefined>(undefined)
const relationType = ref('adaptation')
const searching = ref(false)
const galgameOptions = ref<DtoGalgameListItem[]>([])

const relationLabel = (slug: string | undefined): string =>
  NOVEL_RELATION_TYPES.find((item) => item.slug === slug)?.label ?? slug ?? '相关'

async function loadRelations(): Promise<void> {
  loading.value = true
  try {
    const data = unwrapApiData(await listNovelRelations(props.novelId))
    relations.value = data.items ?? []
  } catch (error) {
    message.error(getApiErrorMessage(error, '加载关联作品失败'))
  } finally {
    loading.value = false
  }
}

async function searchGalgames(keyword: string): Promise<void> {
  if (!keyword.trim()) {
    galgameOptions.value = []
    return
  }
  searching.value = true
  try {
    const data = unwrapApiData(await listGalgames({ keyword: keyword.trim(), limit: 20 }))
    galgameOptions.value = data.items ?? []
  } catch {
    galgameOptions.value = []
  } finally {
    searching.value = false
  }
}

async function addRelation(): Promise<void> {
  if (!targetId.value) {
    message.warning('请选择要关联的作品')
    return
  }
  submitting.value = true
  try {
    await createNovelRelation(props.novelId, {
      target_type: targetType.value,
      target_id: targetId.value,
      relation_type: relationType.value as 'adaptation'
    })
    message.success('关联已添加')
    targetId.value = undefined
    galgameOptions.value = []
    await loadRelations()
    emit('changed')
  } catch (error) {
    message.error(getApiErrorMessage(error, '添加关联失败'))
  } finally {
    submitting.value = false
  }
}

async function removeRelation(relationId: number | undefined): Promise<void> {
  if (!relationId) {
    return
  }
  try {
    await deleteNovelRelation(props.novelId, relationId)
    message.success('关联已删除')
    await loadRelations()
    emit('changed')
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除关联失败'))
  }
}

function targetHref(relation: DtoRelationData): string {
  const isSource = relation.source_type === 'novel' && relation.source_id === props.novelId
  const otherType = isSource ? relation.target_type : relation.source_type
  const otherId = isSource ? relation.target_id : relation.source_id
  return otherType === 'galgame' ? `/galgames/${otherId}` : `/novels/${otherId}`
}

onMounted(() => {
  void loadRelations()
})
</script>

<template>
  <KunCard padding="md">
    <template #header>
      <KunHeader name="关联作品" scale="h3" />
    </template>

    <a-spin :spinning="loading">
      <div class="relation-form">
        <a-select
          v-model:value="targetType"
          class="type-select"
          :options="[
            { value: 'galgame', label: 'Galgame' },
            { value: 'novel', label: '小说' }
          ]"
        />
        <a-select
          v-if="targetType === 'galgame'"
          v-model:value="targetId"
          class="target-select"
          show-search
          :filter-option="false"
          placeholder="搜索 Galgame 标题"
          :options="galgameOptions.map((item) => ({
            value: item.id,
            label: item.title
          }))"
          :not-found-content="searching ? undefined : null"
          @search="searchGalgames"
        />
        <a-input-number
          v-else
          v-model:value="targetId"
          class="target-select"
          placeholder="小说 ID"
          :min="1"
        />
        <a-select
          v-model:value="relationType"
          class="relation-select"
          :options="
            NOVEL_RELATION_TYPES.map((item) => ({
              value: item.slug,
              label: item.label
            }))
          "
        />
        <a-button type="primary" :loading="submitting" @click="addRelation">
          添加
        </a-button>
      </div>

      <KunNull v-if="relations.length === 0" message="暂无关联作品" />
      <div v-else class="relation-list">
        <div v-for="relation in relations" :key="relation.id" class="relation-row">
          <NuxtLink :to="targetHref(relation)" class="relation-link">
            <KunChip size="sm" variant="flat">
              {{ relation.source_type === 'novel' && relation.source_id === props.novelId ? relation.target_type : relation.source_type }}
            </KunChip>
            <span>
              {{
                relation.source_type === 'novel' && relation.source_id === props.novelId
                  ? relation.target_id
                  : relation.source_id
              }}
            </span>
            <span class="relation-type">{{ relationLabel(relation.relation_type) }}</span>
          </NuxtLink>
          <a-popconfirm title="确认删除该关联？" @confirm="removeRelation(relation.id)">
            <a-button size="small" danger>
              <template #icon><KunIcon name="lucide:trash-2" /></template>
            </a-button>
          </a-popconfirm>
        </div>
      </div>
    </a-spin>
  </KunCard>
</template>

<style scoped>
.relation-form {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 16px;
}

.type-select {
  width: 110px;
}

.target-select {
  flex: 1;
  min-width: 200px;
}

.relation-select {
  width: 130px;
}

.relation-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.relation-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--color-content3);
  border-radius: var(--radius-kun-md);
}

.relation-link {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--color-foreground);
}

.relation-type {
  color: var(--color-default-500);
  font-size: 13px;
}
</style>
