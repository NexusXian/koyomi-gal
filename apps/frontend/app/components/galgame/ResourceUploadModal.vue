<script setup lang="ts">
import { message } from 'ant-design-vue'
import { RESOURCE_TYPES } from '~/constants/domain'
import { createResource } from '~/api/generated/resources/resources'

const props = withDefaults(
  defineProps<{
    open: boolean
    galgameId?: number
    targetType?: 'galgame' | 'novel'
    targetId?: number
  }>(),
  { galgameId: undefined, targetType: 'galgame', targetId: undefined }
)

const emit = defineEmits<{
  'update:open': [value: boolean]
  created: []
}>()

const effectiveTargetId = computed(() => props.targetId ?? props.galgameId ?? 0)

const defaultType = computed(() => (props.targetType === 'novel' ? 9 : 1))

const typeOptions = computed(() =>
  RESOURCE_TYPES.filter((item) =>
    props.targetType === 'novel' ? item.value === 0 || item.value >= 7 : item.value <= 6
  )
)

const formState = reactive({
  title: '',
  type: 1,
  description: '',
  links: ''
})
const submitting = ref(false)

watch(defaultType, (value) => {
  formState.type = value
})

function close(): void {
  emit('update:open', false)
}

async function submit(): Promise<void> {
  const links = formState.links
    .split(/[\n,，]/)
    .map((link) => link.trim())
    .filter(Boolean)

  if (!formState.title.trim()) {
    message.warning('请输入资源标题')
    return
  }

  if (links.length === 0) {
    message.warning('请至少填写一个下载链接')
    return
  }

  submitting.value = true
  try {
    await createResource({
      target_type: props.targetType,
      target_id: effectiveTargetId.value,
      title: formState.title.trim(),
      type: formState.type as 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12,
      description: formState.description.trim() || undefined,
      links
    })
    message.success('资源已提交，等待审核')
    Object.assign(formState, {
      title: '',
      type: defaultType.value,
      description: '',
      links: ''
    })
    emit('created')
    close()
  } catch (error) {
    message.error(getApiErrorMessage(error, '提交资源失败'))
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  formState.type = defaultType.value
})
</script>

<template>
  <a-modal
    :open="open"
    title="上传资源"
    :confirm-loading="submitting"
    ok-text="提交"
    cancel-text="取消"
    @ok="submit"
    @cancel="close"
  >
    <a-form layout="vertical">
      <a-form-item label="标题" required>
        <a-input v-model:value="formState.title" :maxlength="255" />
      </a-form-item>

      <a-form-item label="类型">
        <a-select
          v-model:value="formState.type"
          :options="
            typeOptions.map((item) => ({
              value: item.value,
              label: item.label
            }))
          "
        />
      </a-form-item>

      <a-form-item label="说明">
        <a-textarea v-model:value="formState.description" :rows="3" />
      </a-form-item>

      <a-form-item label="下载链接（每行一个，最多 50 个）" required>
        <a-textarea
          v-model:value="formState.links"
          :rows="4"
          placeholder="https://example.com/file.7z"
        />
      </a-form-item>
    </a-form>
  </a-modal>
</template>
