<script setup lang="ts">
import { message } from 'ant-design-vue'
import { REPORT_REASONS } from '~/constants/domain'
import { createResourceReport } from '~/api/generated/resources/resources'

const props = defineProps<{
  open: boolean
  resourceId: number | null
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const formState = reactive({
  reason: 0,
  description: ''
})
const submitting = ref(false)

function close(): void {
  emit('update:open', false)
}

async function submit(): Promise<void> {
  if (!props.resourceId) {
    return
  }

  submitting.value = true
  try {
    await createResourceReport(props.resourceId, {
      reason: formState.reason as 0 | 1 | 2 | 3 | 4 | 5 | 6,
      description: formState.description.trim() || undefined
    })
    message.success('举报已提交，管理员会尽快处理')
    Object.assign(formState, { reason: 0, description: '' })
    close()
  } catch (error) {
    message.error(getApiErrorMessage(error, '提交举报失败'))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <a-modal
    :open="open"
    title="举报资源"
    :confirm-loading="submitting"
    ok-text="提交举报"
    cancel-text="取消"
    @ok="submit"
    @cancel="close"
  >
    <a-form layout="vertical">
      <a-form-item label="举报原因">
        <a-select
          v-model:value="formState.reason"
          :options="
            REPORT_REASONS.map((item) => ({
              value: item.value,
              label: item.label
            }))
          "
        />
      </a-form-item>

      <a-form-item label="补充说明">
        <a-textarea
          v-model:value="formState.description"
          :rows="4"
          :maxlength="1000"
          placeholder="补充描述问题（可选）"
        />
      </a-form-item>
    </a-form>
  </a-modal>
</template>
