<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import {
  createAdminLevel,
  deleteAdminLevel,
  listAdminExperienceRules,
  listAdminLevels,
  updateAdminExperienceRule,
  updateAdminLevel
} from '~/api/generated/admin/admin'
import type {
  LeveldtoCreateLevelConfigRequest,
  LeveldtoExperienceRuleData,
  LeveldtoLevelConfigData,
  LeveldtoUpdateExperienceRuleRequest,
  LeveldtoUpdateLevelConfigRequest
} from '~/api/generated/models'

useSeoMeta({ title: '等级与经验配置 - Koyomi' })

const { has } = usePermissions()
const activeTab = ref<'levels' | 'rules'>('levels')

const levels = ref<LeveldtoLevelConfigData[]>([])
const rules = ref<LeveldtoExperienceRuleData[]>([])
const loading = ref(false)

const levelModalOpen = ref(false)
const editingLevel = ref<LeveldtoLevelConfigData | null>(null)
const levelSaving = ref(false)
const levelForm = reactive({
  level: 1,
  name: '',
  min_exp: 0,
  icon_url: '',
  color: '',
  description: '',
  is_enabled: true
})

const ruleModalOpen = ref(false)
const editingRule = ref<LeveldtoExperienceRuleData | null>(null)
const ruleSaving = ref(false)
const ruleForm = reactive({
  name: '',
  exp: 0,
  daily_limit: 0,
  daily_exp_limit: 0,
  cooldown_seconds: 0,
  enabled: true,
  description: ''
})

async function load(): Promise<void> {
  loading.value = true
  try {
    if (has('level_config:read')) {
      levels.value = unwrapApiData(await listAdminLevels())?.items ?? []
    }
    if (has('experience_rule:read')) {
      rules.value = unwrapApiData(await listAdminExperienceRules())?.items ?? []
    }
  } catch (error) {
    message.error(getApiErrorMessage(error, '加载配置失败'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void load()
})

function openLevelCreate(): void {
  if (!has('level_config:create')) {
    return
  }
  editingLevel.value = null
  Object.assign(levelForm, {
    level: (levels.value.at(-1)?.level ?? 0) + 1,
    name: '',
    min_exp: 0,
    icon_url: '',
    color: '',
    description: '',
    is_enabled: true
  })
  levelModalOpen.value = true
}

function openLevelEdit(level: LeveldtoLevelConfigData): void {
  if (!has('level_config:update')) {
    return
  }
  editingLevel.value = level
  Object.assign(levelForm, {
    level: level.level ?? 1,
    name: level.name ?? '',
    min_exp: level.min_exp ?? 0,
    icon_url: level.icon_url ?? '',
    color: level.color ?? '',
    description: level.description ?? '',
    is_enabled: level.is_enabled ?? true
  })
  levelModalOpen.value = true
}

async function submitLevel(): Promise<void> {
  if (!levelForm.name.trim() || levelForm.min_exp < 0 || levelForm.level < 1) {
    message.warning('请填写正确的等级名称与最低经验')
    return
  }
  levelSaving.value = true
  try {
    if (editingLevel.value?.id) {
      const payload: LeveldtoUpdateLevelConfigRequest = {
        name: levelForm.name.trim(),
        min_exp: levelForm.min_exp,
        icon_url: levelForm.icon_url.trim() || '',
        color: levelForm.color.trim() || '',
        description: levelForm.description.trim() || '',
        is_enabled: levelForm.is_enabled
      }
      await updateAdminLevel(editingLevel.value.id, payload)
      message.success('等级配置已更新')
    } else {
      const payload: LeveldtoCreateLevelConfigRequest = {
        level: levelForm.level,
        name: levelForm.name.trim(),
        min_exp: levelForm.min_exp,
        icon_url: levelForm.icon_url.trim() || '',
        color: levelForm.color.trim() || '',
        description: levelForm.description.trim() || '',
        is_enabled: levelForm.is_enabled
      }
      await createAdminLevel(payload)
      message.success('等级配置已创建')
    }
    levelModalOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '保存等级配置失败'))
  } finally {
    levelSaving.value = false
  }
}

async function submitLevelDelete(level: LeveldtoLevelConfigData): Promise<void> {
  if (!level.id || !has('level_config:update')) {
    return
  }
  try {
    await deleteAdminLevel(level.id)
    message.success('等级配置已删除或已禁用')
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '删除等级配置失败'))
  }
}

function openRuleEdit(rule: LeveldtoExperienceRuleData): void {
  if (!has('experience_rule:update')) {
    return
  }
  editingRule.value = rule
  Object.assign(ruleForm, {
    name: rule.name ?? '',
    exp: rule.exp ?? 0,
    daily_limit: rule.daily_limit ?? 0,
    daily_exp_limit: rule.daily_exp_limit ?? 0,
    cooldown_seconds: rule.cooldown_seconds ?? 0,
    enabled: rule.enabled ?? true,
    description: rule.description ?? ''
  })
  ruleModalOpen.value = true
}

async function submitRule(): Promise<void> {
  if (!editingRule.value?.id) {
    return
  }
  if (!ruleForm.name.trim() || ruleForm.exp < 0) {
    message.warning('请填写正确的规则名称与经验值')
    return
  }
  ruleSaving.value = true
  try {
    const payload: LeveldtoUpdateExperienceRuleRequest = {
      name: ruleForm.name.trim(),
      exp: ruleForm.exp,
      daily_limit: ruleForm.daily_limit,
      daily_exp_limit: ruleForm.daily_exp_limit,
      cooldown_seconds: ruleForm.cooldown_seconds,
      enabled: ruleForm.enabled,
      description: ruleForm.description.trim() || ''
    }
    await updateAdminExperienceRule(editingRule.value.id, payload)
    message.success('经验规则已更新')
    ruleModalOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '保存经验规则失败'))
  } finally {
    ruleSaving.value = false
  }
}

const levelColumns = computed<TableColumnsType>(() => [
  { title: '等级', dataIndex: 'level', width: 80 },
  { title: '名称', dataIndex: 'name', width: 140 },
  { title: '最低经验', dataIndex: 'min_exp', width: 120 },
  { title: '颜色', dataIndex: 'color', width: 100 },
  { title: '描述', dataIndex: 'description', ellipsis: true },
  { title: '状态', key: 'enabled', width: 90 },
  ...(has('level_config:update') || has('level_config:create')
    ? [{ title: '操作', key: 'actions', width: 140 }]
    : [])
])

const ruleColumns = computed<TableColumnsType>(() => [
  { title: '事件', dataIndex: 'event_type', width: 220 },
  { title: '名称', dataIndex: 'name', width: 150 },
  { title: '经验', dataIndex: 'exp', width: 80 },
  { title: '每日次数', key: 'daily_limit', width: 100 },
  { title: '每日经验上限', key: 'daily_exp_limit', width: 130 },
  { title: '冷却（秒）', dataIndex: 'cooldown_seconds', width: 110 },
  { title: '状态', key: 'enabled', width: 90 },
  ...(has('experience_rule:update') ? [{ title: '操作', key: 'actions', width: 90 }] : [])
])

function limitLabel(value: number | undefined): string {
  return value && value > 0 ? `${value}` : '不限'
}
</script>

<template>
  <div>
    <a-tabs v-model:active-key="activeTab">
      <a-tab-pane v-if="has('level_config:read')" key="levels" tab="等级配置" />
      <a-tab-pane v-if="has('experience_rule:read')" key="rules" tab="经验规则" />
    </a-tabs>

    <template v-if="activeTab === 'levels'">
      <div v-if="has('level_config:create')" class="table-toolbar">
        <a-button type="primary" @click="openLevelCreate">
          新建等级
        </a-button>
      </div>
      <a-table
        :columns="levelColumns"
        :data-source="levels"
        :loading="loading"
        :pagination="false"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'enabled'">
            <a-tag :color="record.is_enabled ? 'green' : 'default'">
              {{ record.is_enabled ? '启用' : '禁用' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'actions'">
            <div class="table-actions">
              <a-button
                v-if="has('level_config:update')"
                size="small"
                @click="openLevelEdit(record)"
              >
                编辑
              </a-button>
              <a-popconfirm
                title="删除该等级？仍有用户持有时会改为禁用"
                @confirm="submitLevelDelete(record)"
              >
                <a-button v-if="has('level_config:update')" size="small" danger>
                  删除
                </a-button>
              </a-popconfirm>
            </div>
          </template>
          <template v-else-if="column.dataIndex === 'color'">
            <span class="color-cell">
              <span
                class="color-dot"
                :style="{ backgroundColor: record.color || 'transparent' }"
              />
              {{ record.color || '-' }}
            </span>
          </template>
          <template v-else-if="column.dataIndex === 'description'">
            {{ record.description || '-' }}
          </template>
        </template>
      </a-table>
    </template>

    <template v-else>
      <a-table
        :columns="ruleColumns"
        :data-source="rules"
        :loading="loading"
        :pagination="false"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'daily_limit'">
            {{ limitLabel(record.daily_limit) }}
          </template>
          <template v-else-if="column.key === 'daily_exp_limit'">
            {{ limitLabel(record.daily_exp_limit) }}
          </template>
          <template v-else-if="column.key === 'enabled'">
            <a-tag :color="record.enabled ? 'green' : 'default'">
              {{ record.enabled ? '启用' : '禁用' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-button
              v-if="has('experience_rule:update')"
              size="small"
              @click="openRuleEdit(record)"
            >
              编辑
            </a-button>
          </template>
        </template>
      </a-table>
    </template>

    <a-modal
      v-model:open="levelModalOpen"
      :title="editingLevel ? `编辑 LV${levelForm.level}` : '新建等级'"
      :confirm-loading="levelSaving"
      ok-text="保存"
      cancel-text="取消"
      @ok="submitLevel"
    >
      <a-form layout="vertical">
        <a-form-item v-if="!editingLevel" label="等级序号" required>
          <a-input-number v-model:value="levelForm.level" :min="1" :max="1000" />
        </a-form-item>
        <a-form-item label="等级名称" required>
          <a-input v-model:value="levelForm.name" :maxlength="50" />
        </a-form-item>
        <a-form-item label="所需最低经验" required>
          <a-input-number v-model:value="levelForm.min_exp" :min="0" />
        </a-form-item>
        <a-form-item label="颜色">
          <a-input v-model:value="levelForm.color" :maxlength="32" placeholder="#f59e0b" />
        </a-form-item>
        <a-form-item label="图标 URL">
          <a-input v-model:value="levelForm.icon_url" :maxlength="2048" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="levelForm.description" :rows="2" :maxlength="255" />
        </a-form-item>
        <a-form-item label="启用状态">
          <a-switch v-model:checked="levelForm.is_enabled" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="ruleModalOpen"
      :title="editingRule ? `编辑规则：${editingRule.name}` : '编辑规则'"
      :confirm-loading="ruleSaving"
      ok-text="保存"
      cancel-text="取消"
      @ok="submitRule"
    >
      <a-form layout="vertical">
        <a-form-item label="规则名称" required>
          <a-input v-model:value="ruleForm.name" :maxlength="100" />
        </a-form-item>
        <a-form-item label="经验值" required>
          <a-input-number v-model:value="ruleForm.exp" :min="0" />
        </a-form-item>
        <a-form-item label="每日次数上限（0 为不限）">
          <a-input-number v-model:value="ruleForm.daily_limit" :min="0" />
        </a-form-item>
        <a-form-item label="每日经验上限（0 为不限）">
          <a-input-number v-model:value="ruleForm.daily_exp_limit" :min="0" />
        </a-form-item>
        <a-form-item label="冷却时间（秒，0 为不限）">
          <a-input-number v-model:value="ruleForm.cooldown_seconds" :min="0" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="ruleForm.description" :rows="2" :maxlength="255" />
        </a-form-item>
        <a-form-item label="启用状态">
          <a-switch v-model:checked="ruleForm.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.table-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 14px;
}

.table-actions {
  display: flex;
  gap: 6px;
}

.color-cell {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.color-dot {
  width: 12px;
  height: 12px;
  border: 1px solid var(--app-glass-border);
  border-radius: 50%;
}
</style>
