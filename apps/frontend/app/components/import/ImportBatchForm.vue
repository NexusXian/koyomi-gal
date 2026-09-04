<script setup lang="ts">
import { message } from 'ant-design-vue'
import { createImportBatch } from '~/api/generated/admin-import/admin-import'
import type { DtoImportJobData } from '~/api/generated/models'

const props = defineProps<{
  providers: string[]
}>()

const emit = defineEmits<{
  created: [job: DtoImportJobData]
}>()

const LANGUAGE_OPTIONS = [
  { value: '', label: '全部语言' },
  { value: 'ja', label: '日语' },
  { value: 'en', label: '英语' },
  { value: 'zh-Hans', label: '简体中文' },
  { value: 'zh-Hant', label: '繁体中文' },
  { value: 'ko', label: '韩语' }
]

const formState = reactive({
  provider: props.providers[0] ?? 'vndb',
  min_rating: 7 as number | undefined,
  min_vote_count: 100 as number | undefined,
  from_year: undefined as number | undefined,
  to_year: undefined as number | undefined,
  original_language: '',
  limit: 200 as number | undefined
})

const submitting = ref(false)

watch(
  () => props.providers,
  (value) => {
    if (value.length > 0 && !value.includes(formState.provider)) {
      formState.provider = value[0] ?? 'vndb'
    }
  }
)

async function submit(): Promise<void> {
  if (!formState.limit || formState.limit < 1) {
    message.warning('请填写最大导入数量')
    return
  }
  submitting.value = true
  try {
    const job = unwrapApiData(
      await createImportBatch({
        provider: formState.provider,
        min_rating: formState.min_rating || undefined,
        min_vote_count: formState.min_vote_count || undefined,
        from_year: formState.from_year || undefined,
        to_year: formState.to_year || undefined,
        original_language: formState.original_language || undefined,
        limit: formState.limit
      })
    )
    message.success(`批量导入任务 #${job.id ?? ''} 已创建`)
    emit('created', job)
  } catch (error) {
    message.error(getApiErrorMessage(error, '创建批量导入任务失败'))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <a-form layout="inline" class="import-batch-form" @submit.prevent>
    <a-form-item label="数据源">
      <a-select
        v-model:value="formState.provider"
        class="import-batch-field"
        :options="providers.map((item) => ({ value: item, label: item.toUpperCase() }))"
      />
    </a-form-item>
    <a-form-item label="最低评分">
      <a-input-number
        v-model:value="formState.min_rating"
        class="import-batch-field"
        :min="0"
        :max="10"
        :step="0.5"
        placeholder="0-10"
      />
    </a-form-item>
    <a-form-item label="最低投票数">
      <a-input-number
        v-model:value="formState.min_vote_count"
        class="import-batch-field"
        :min="0"
        :step="50"
        placeholder="100"
      />
    </a-form-item>
    <a-form-item label="起始年份">
      <a-input-number
        v-model:value="formState.from_year"
        class="import-batch-field"
        :min="1950"
        :max="9999"
        placeholder="1990"
      />
    </a-form-item>
    <a-form-item label="结束年份">
      <a-input-number
        v-model:value="formState.to_year"
        class="import-batch-field"
        :min="1950"
        :max="9999"
        placeholder="2024"
      />
    </a-form-item>
    <a-form-item label="原始语言">
      <a-select
        v-model:value="formState.original_language"
        class="import-batch-field"
        :options="LANGUAGE_OPTIONS"
      />
    </a-form-item>
    <a-form-item label="最大数量">
      <a-input-number
        v-model:value="formState.limit"
        class="import-batch-field"
        :min="1"
        :max="5000"
        :step="100"
      />
    </a-form-item>
    <a-form-item>
      <a-button type="primary" :loading="submitting" @click="submit">
        开始导入
      </a-button>
    </a-form-item>
  </a-form>
</template>

<style scoped>
.import-batch-form {
  row-gap: 8px;
}

.import-batch-field {
  width: 130px;
}
</style>
