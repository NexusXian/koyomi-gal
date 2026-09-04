<script setup lang="ts">
import { message } from 'ant-design-vue'
import {
  importGame,
  previewImportGame
} from '~/api/generated/admin-import/admin-import'
import { getAdminGalgame } from '~/api/generated/admin/admin'
import type {
  DtoExternalGameDetail,
  DtoImportDuplicateCandidate,
  DtoImportPreviewData,
  DtoImportResultData
} from '~/api/generated/models'
import type { DtoGalgameResponse } from '~/api/generated/models'

const props = defineProps<{
  open: boolean
  provider: string
  externalId: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  imported: [result: DtoImportResultData]
}>()

const loading = ref(false)
const importing = ref(false)
const preview = ref<DtoImportPreviewData | null>(null)
const selectedCandidateId = ref<number | null>(null)
const linking = ref(false)
const forceMetadataUpdate = ref(false)
const existingDetail = ref<DtoGalgameResponse | null>(null)
const existingLoading = ref(false)

const game = computed<DtoExternalGameDetail | null>(() => preview.value?.game ?? null)
const duplicateStatus = computed(() => preview.value?.duplicate_status ?? 'none')
const candidates = computed<DtoImportDuplicateCandidate[]>(
  () => preview.value?.candidates ?? []
)

watch(
  () => [props.open, props.provider, props.externalId],
  ([open]) => {
    if (open) {
      reset()
      void load()
    }
  },
  { immediate: true }
)

function reset(): void {
  preview.value = null
  selectedCandidateId.value = null
  linking.value = false
  forceMetadataUpdate.value = false
  existingDetail.value = null
}

async function load(): Promise<void> {
  loading.value = true
  try {
    preview.value = unwrapApiData(
      await previewImportGame(props.provider, props.externalId)
    )
  } catch (error) {
    message.error(getApiErrorMessage(error, '获取外部作品详情失败'))
    emit('update:open', false)
  } finally {
    loading.value = false
  }
}

function close(): void {
  emit('update:open', false)
}

async function runImport(
  duplicateAction: 'error' | 'create_new' | 'link_existing'
): Promise<void> {
  importing.value = true
  try {
    const result = unwrapApiData(
      await importGame({
        provider: props.provider,
        external_id: props.externalId,
        duplicate_action: duplicateAction,
        existing_galgame_id:
          duplicateAction === 'link_existing'
            ? (selectedCandidateId.value ?? undefined)
            : undefined
      })
    )
    if (duplicateAction === 'error' && result.duplicate_status === 'possible') {
      preview.value = {
        ...preview.value,
        duplicate_status: 'possible',
        candidates: result.candidates ?? []
      }
      selectedCandidateId.value =
        (result.candidates ?? [])[0]?.id ?? null
      message.warning('检测到疑似重复条目，请选择处理方式')
      return
    }
    if (result.duplicate_status === 'already_imported') {
      message.info('该外部作品已导入过')
      emit('imported', result)
      close()
      return
    }
    message.success('导入成功')
    emit('imported', result)
    close()
  } catch (error) {
    message.error(getApiErrorMessage(error, '导入失败'))
  } finally {
    importing.value = false
  }
}

async function startLink(): Promise<void> {
  if (!selectedCandidateId.value) {
    message.warning('请先选择要关联的站内作品')
    return
  }
  linking.value = true
  existingLoading.value = true
  try {
    existingDetail.value = unwrapApiData(
      await getAdminGalgame(selectedCandidateId.value)
    )
  } catch (error) {
    message.error(getApiErrorMessage(error, '获取站内作品详情失败'))
  } finally {
    existingLoading.value = false
  }
}

async function confirmLink(): Promise<void> {
  importing.value = true
  try {
    const result = unwrapApiData(
      await importGame({
        provider: props.provider,
        external_id: props.externalId,
        duplicate_action: 'link_existing',
        existing_galgame_id: selectedCandidateId.value ?? undefined,
        force_metadata_update: forceMetadataUpdate.value
      })
    )
    message.success('已关联站内作品')
    emit('imported', result)
    close()
  } catch (error) {
    message.error(getApiErrorMessage(error, '关联失败'))
  } finally {
    importing.value = false
  }
}

function formatRating(rating?: number | null): string {
  return rating === null || rating === undefined ? '暂无' : rating.toFixed(2)
}

function formatLength(minutes?: number | null): string {
  if (!minutes) {
    return '未知'
  }
  const hours = Math.round(minutes / 60)
  return `约 ${hours} 小时`
}
</script>

<template>
  <a-drawer
    :open="open"
    title="外部作品预览"
    width="620"
    :destroy-on-close="true"
    @close="close"
  >
    <a-spin :spinning="loading">
      <div v-if="game" class="import-preview">
        <div class="import-preview-header">
          <div class="import-preview-cover">
            <img
              v-if="game.cover_url"
              :src="game.cover_url"
              :alt="game.title"
              loading="lazy"
              referrerpolicy="no-referrer"
            />
            <KunNull v-else text="无封面" />
          </div>
          <div class="import-preview-meta">
            <h3 class="import-preview-title">{{ game.title }}</h3>
            <p v-if="game.original_title" class="import-preview-subtitle">
              {{ game.original_title }}
            </p>
            <p v-if="game.romaji_title" class="import-preview-subtitle">
              {{ game.romaji_title }}
            </p>
            <a-descriptions :column="1" size="small" class="import-preview-desc">
              <a-descriptions-item label="数据源">
                {{ game.source?.toUpperCase() }} · {{ game.external_id }}
              </a-descriptions-item>
              <a-descriptions-item label="开发商">
                {{ game.developer || '未知' }}
              </a-descriptions-item>
              <a-descriptions-item label="发行日期">
                {{ game.release_date ?? '未知' }}
              </a-descriptions-item>
              <a-descriptions-item label="外部评分">
                {{ formatRating(game.rating) }}（{{ game.rating_count ?? 0 }} 票）
              </a-descriptions-item>
              <a-descriptions-item label="游戏时长">
                {{ formatLength(game.length_minutes) }}
              </a-descriptions-item>
              <a-descriptions-item label="原始语言">
                {{ game.original_language || '未知' }}
              </a-descriptions-item>
            </a-descriptions>
          </div>
        </div>

        <a-divider>简介</a-divider>
        <p class="import-preview-description">
          {{ game.description?.trim() || '暂无简介' }}
        </p>

        <template v-if="(game.tags ?? []).length > 0">
          <a-divider>标签</a-divider>
          <div class="import-preview-tags">
            <a-tag v-for="tag in game.tags" :key="tag">{{ tag }}</a-tag>
          </div>
        </template>

        <a-divider>导入</a-divider>

        <a-alert
          v-if="duplicateStatus === 'already_imported'"
          type="info"
          show-icon
          message="该外部作品已导入站内"
          :description="
            candidates.length > 0 && candidates[0]?.id
              ? `关联的站内作品 ID：${candidates[0].id}`
              : ''
          "
        />

        <template v-else-if="duplicateStatus === 'possible'">
          <a-alert
            type="warning"
            show-icon
            message="发现可能重复条目"
            description="站内已有相似作品，默认禁止直接创建重复记录，请选择处理方式。"
          />
          <a-radio-group
            v-model:value="selectedCandidateId"
            class="import-preview-candidates"
          >
            <a-radio
              v-for="candidate in candidates"
              :key="candidate.id"
              :value="candidate.id"
              class="import-preview-candidate"
            >
              <span class="import-preview-candidate-title">
                {{ candidate.title || `#${candidate.id}` }}
              </span>
              <span class="import-preview-candidate-sub">
                ID #{{ candidate.id }}
                {{ candidate.original_title ? `· ${candidate.original_title}` : '' }}
                {{ candidate.release_date ? `· ${candidate.release_date.slice(0, 10)}` : '' }}
              </span>
            </a-radio>
          </a-radio-group>

          <div v-if="!linking" class="import-preview-actions">
            <a-button
              type="primary"
              :disabled="!selectedCandidateId"
              @click="startLink"
            >
              关联已有作品
            </a-button>
            <a-popconfirm
              title="确认要忽略疑似重复并创建新作品吗？"
              ok-text="确认创建"
              cancel-text="取消"
              @confirm="runImport('create_new')"
            >
              <a-button :loading="importing">仍然创建新作品</a-button>
            </a-popconfirm>
            <a-button @click="close">取消</a-button>
          </div>

          <div v-else class="import-preview-link">
            <a-spin :spinning="existingLoading">
              <template v-if="existingDetail">
                <ImportFieldDiff
                  :existing="existingDetail"
                  :external="game"
                />
                <a-alert
                  class="import-preview-force-hint"
                  type="info"
                  show-icon
                  message="默认只同步外部来源、外部评分与同步时间，不会覆盖站内已维护的标题、简介、封面、开发商与标签。勾选下方选项后才强制覆盖。"
                />
                <a-checkbox v-model:checked="forceMetadataUpdate">
                  强制使用外部数据覆盖站内元数据
                </a-checkbox>
                <div class="import-preview-actions">
                  <a-button
                    type="primary"
                    :loading="importing"
                    @click="confirmLink"
                  >
                    确认关联
                  </a-button>
                  <a-button :disabled="importing" @click="linking = false">
                    返回
                  </a-button>
                </div>
              </template>
            </a-spin>
          </div>
        </template>

        <div v-else class="import-preview-actions">
          <a-button
            type="primary"
            :loading="importing"
            @click="runImport('error')"
          >
            导入
          </a-button>
          <a-button @click="close">取消</a-button>
        </div>
      </div>
      <KunNull v-else-if="!loading" text="未找到外部作品" />
    </a-spin>
  </a-drawer>
</template>

<style scoped>
.import-preview-header {
  display: flex;
  gap: 16px;
}

.import-preview-cover {
  width: 160px;
  flex: none;
  overflow: hidden;
  border-radius: 8px;
}

.import-preview-cover img {
  width: 100%;
  display: block;
  object-fit: cover;
}

.import-preview-meta {
  flex: 1;
  min-width: 0;
}

.import-preview-title {
  margin: 0 0 4px;
}

.import-preview-subtitle {
  margin: 0 0 4px;
  opacity: 0.75;
  font-size: 13px;
}

.import-preview-desc {
  margin-top: 8px;
}

.import-preview-description {
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
}

.import-preview-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.import-preview-candidates {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 14px 0;
  width: 100%;
}

.import-preview-candidate {
  margin: 0;
}

.import-preview-candidate-title {
  font-weight: 600;
}

.import-preview-candidate-sub {
  margin-left: 8px;
  font-size: 12px;
  opacity: 0.7;
}

.import-preview-actions {
  display: flex;
  gap: 10px;
  margin-top: 14px;
}

.import-preview-link {
  margin-top: 14px;
}

.import-preview-force-hint {
  margin: 12px 0;
}
</style>
