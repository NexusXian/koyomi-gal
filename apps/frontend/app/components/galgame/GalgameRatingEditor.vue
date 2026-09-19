<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { DtoRatingRecordData } from '~/api/generated/models'
import type { GalgameRatingDraft } from '~/composables/useGalgameRatings'
import {
  RATING_DIMENSIONS,
  RECOMMENDATION_OPTIONS,
  SPOILER_OPTIONS
} from '~/constants/rating'

const props = defineProps<{
  open: boolean
  initial: DtoRatingRecordData | null
  saving?: boolean
  deleting?: boolean
}>()

const emit = defineEmits<{
  'update:open': [open: boolean]
  submit: [draft: GalgameRatingDraft]
  delete: []
}>()

const overall = ref<number | null>(null)
const dimensions = ref<Record<string, number | null>>({})
const recommendation = ref<number | null>(null)
const spoilerLevel = ref(0)
const reviewText = ref('')
const dimensionsOpen = ref<string[]>([])

function hydrate(): void {
  const initial = props.initial
  overall.value = initial?.overall ?? null
  recommendation.value = initial?.recommendation ?? null
  spoilerLevel.value = initial?.spoiler_level ?? 0
  reviewText.value = initial?.review_text ?? ''
  const nextDimensions: Record<string, number | null> = {}
  let hasDimensions = false
  for (const { key } of RATING_DIMENSIONS) {
    const value = initial?.dimensions?.[key] ?? null
    nextDimensions[key] = value
    if (value !== null) {
      hasDimensions = true
    }
  }
  dimensions.value = nextDimensions
  dimensionsOpen.value = hasDimensions ? ['dimensions'] : []
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      hydrate()
    }
  }
)

function setDimension(key: string, value: number | null): void {
  dimensions.value = { ...dimensions.value, [key]: value }
}

function submit(): void {
  if (overall.value === null) {
    message.warning('请先选择总体评分')
    return
  }
  const dimensionDraft: GalgameRatingDraft['dimensions'] = {}
  for (const { key } of RATING_DIMENSIONS) {
    dimensionDraft[key] = dimensions.value[key] ?? null
  }
  emit('submit', {
    overall: overall.value,
    dimensions: dimensionDraft,
    recommendation: recommendation.value,
    reviewText: reviewText.value,
    spoilerLevel: spoilerLevel.value
  })
}

function close(): void {
  emit('update:open', false)
}
</script>

<template>
  <a-modal
    :open="open"
    :title="initial ? '修改评价' : '发表评价'"
    :width="720"
    :footer="null"
    :mask-closable="false"
    @cancel="close"
  >
    <div class="rating-editor">
      <div class="rating-editor-field">
        <div class="rating-editor-label">
          总体评分 <span class="required">*</span>
        </div>
        <div class="rating-editor-slider">
          <a-slider
            :value="overall ?? undefined"
            :min="1"
            :max="10"
            :marks="{ 1: '1', 5: '5', 10: '10' }"
            @change="(value: number | number[]) => (overall = Number(value))"
          />
          <span class="rating-editor-value">
            {{ overall === null ? '未评分' : `${overall} / 10` }}
          </span>
        </div>
      </div>

      <a-collapse
        v-model:active-key="dimensionsOpen"
        :bordered="false"
        class="rating-editor-collapse"
      >
        <a-collapse-panel key="dimensions" header="详细评分（可选，1-10 分）">
          <div
            v-for="dimension in RATING_DIMENSIONS"
            :key="dimension.key"
            class="rating-editor-field dim-row"
          >
            <span class="dim-row-label">{{ dimension.label }}</span>
            <div class="rating-editor-slider">
              <a-slider
                :value="dimensions[dimension.key] ?? undefined"
                :min="1"
                :max="10"
                @change="
                  (value: number | number[]) =>
                    setDimension(dimension.key, Number(value))
                "
              />
              <span class="rating-editor-value">
                {{
                  dimensions[dimension.key] === null
                    ? '未评分'
                    : `${dimensions[dimension.key]} / 10`
                }}
              </span>
              <KunButton
                v-if="dimensions[dimension.key] !== null"
                size="sm"
                color="default"
                variant="light"
                class="dim-clear"
                @click="setDimension(dimension.key, null)"
              >
                <KunIcon name="lucide:x" />
                清除
              </KunButton>
            </div>
          </div>
        </a-collapse-panel>
      </a-collapse>

      <div class="rating-editor-row">
        <div class="rating-editor-field">
          <div class="rating-editor-label">推荐程度</div>
          <a-select
            :value="recommendation ?? undefined"
            allow-clear
            placeholder="不填写"
            :options="
              RECOMMENDATION_OPTIONS.map((option) => ({
                value: option.value,
                label: option.label
              }))
            "
            @change="
              (value: unknown) =>
                (recommendation =
                  value === undefined || value === null ? null : Number(value))
            "
            @clear="() => (recommendation = null)"
          />
        </div>

        <div class="rating-editor-field">
          <div class="rating-editor-label">剧透等级</div>
          <a-radio-group
            :value="spoilerLevel"
            :options="
              SPOILER_OPTIONS.map((option) => ({
                value: option.value,
                label: option.label
              }))
            "
            @change="
              (event: { target: { value: number | string | boolean } }) =>
                (spoilerLevel = Number(event.target.value))
            "
          />
        </div>
      </div>

      <div class="rating-editor-field">
        <div class="rating-editor-label">评价正文（Markdown）</div>
        <ClientOnly>
          <MarkdownEditor v-model="reviewText" upload-category="galgames" />
          <template #fallback>
            <div class="rating-editor-loading">编辑器加载中…</div>
          </template>
        </ClientOnly>
      </div>

      <div class="rating-editor-actions">
        <a-button @click="close">取消</a-button>
        <a-popconfirm
          v-if="initial"
          title="确定删除自己的评价吗？"
          ok-text="删除"
          cancel-text="取消"
          @confirm="emit('delete')"
        >
          <a-button danger :loading="deleting">删除评价</a-button>
        </a-popconfirm>
        <a-button type="primary" :loading="saving" @click="submit">
          {{ initial ? '保存修改' : '发布评价' }}
        </a-button>
      </div>
    </div>
  </a-modal>
</template>

<style scoped>
.rating-editor {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.rating-editor-field {
  min-width: 0;
}

.rating-editor-label {
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-foreground);
}

.required {
  color: var(--color-danger, #ef4444);
}

.rating-editor-slider {
  display: flex;
  align-items: center;
  gap: 10px;
}

.rating-editor-slider :deep(.ant-slider) {
  flex: 1;
  margin: 8px 0;
}

.rating-editor-value {
  flex: 0 0 auto;
  min-width: 64px;
  font-size: 13px;
  color: var(--color-default-500);
}

.dim-clear {
  flex: 0 0 auto;
}

.rating-editor-collapse {
  background: transparent;
}

.dim-row .dim-row-label {
  flex: 0 0 48px;
  font-size: 14px;
  color: var(--color-default-600);
}

.dim-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.rating-editor-row {
  display: grid;
  gap: 16px;
}

.rating-editor-row :deep(.ant-select) {
  width: 100%;
}

.rating-editor-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: var(--color-default-400);
  border: 1px solid var(--color-default-200);
  border-radius: var(--radius-kun-md, 8px);
}

.rating-editor-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

@media (min-width: 640px) {
  .rating-editor-row {
    grid-template-columns: 200px 1fr;
  }
}
</style>
