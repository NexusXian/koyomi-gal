<script setup lang="ts">
import { message } from 'ant-design-vue'
import { createFeedbackService } from '~/services/feedback'
import type { FeedbackType } from '~/types/feedback'

const props = defineProps<{
  type: FeedbackType
  contactPlaceholder?: string
}>()

const feedbackService = createFeedbackService(useNuxtApp().$api)
const content = ref('')
const contact = ref('')
const submitting = ref(false)

const maxLength = 2000

async function submit(): Promise<void> {
  const trimmed = content.value.trim()
  if (trimmed.length < 5) {
    message.warning('内容至少 5 个字符')
    return
  }

  submitting.value = true
  try {
    await feedbackService.submitFeedback({
      type: props.type,
      content: trimmed,
      contact: contact.value.trim() || undefined
    })
    message.success('提交成功，感谢你的反馈')
    content.value = ''
    contact.value = ''
  } catch (error) {
    message.error(getApiErrorMessage(error, '反馈提交失败，请稍后重试'))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <form class="feedback-form" @submit.prevent="submit">
    <a-form layout="vertical">
      <a-form-item label="内容" required>
        <a-textarea
          v-model:value="content"
          :rows="6"
          :maxlength="maxLength"
          show-count
          :placeholder="
            type === 'copyright'
              ? '请描述被投诉的作品、原始出处和侵权情况'
              : '你想反馈什么？功能建议、问题反馈都很欢迎'
          "
        />
      </a-form-item>
      <a-form-item label="联系方式（可选）">
        <a-input
          v-model:value="contact"
          :maxlength="255"
          :placeholder="contactPlaceholder || '邮箱或其他联系方式，便于我们回复'"
        />
      </a-form-item>
      <KunButton
        type="submit"
        color="primary"
        :loading="submitting"
        :disabled="content.trim().length < 5"
      >
        提交
      </KunButton>
    </a-form>
  </form>
</template>
