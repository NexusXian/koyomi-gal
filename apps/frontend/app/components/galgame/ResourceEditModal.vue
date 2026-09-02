<script setup lang="ts">
import { message } from 'ant-design-vue'
import { updateResource } from '~/api/generated/resources/resources'
import type { DtoResourceData } from '~/api/generated/models'
import { RESOURCE_TYPES } from '~/constants/domain'

const props = defineProps<{
  open: boolean
  resource: DtoResourceData | null
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  updated: []
}>()

const formState = reactive({
  title: '',
  type: 1,
  description: '',
  links: ''
})
const submitting = ref(false)

watch(
  () => [props.open, props.resource] as const,
  ([open, resource]) => {
    if (!open || !resource) {
      return
    }

    Object.assign(formState, {
      title: resource.title ?? '',
      type: resource.type ?? 1,
      description: resource.description ?? '',
      links: (resource.links ?? []).map((link) => link.url).filter(Boolean).join('\n')
    })
  },
  { immediate: true }
)

function close(): void {
  emit('update:open', false)
}

async function submit(): Promise<void> {
  if (!props.resource?.id) {
    return
  }

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
  if (props.resource.status === undefined) {
    message.error('资源状态缺失，无法更新')
    return
  }

  submitting.value = true
  try {
    await updateResource(props.resource.id, {
      title: formState.title.trim(),
      type: formState.type as 0 | 1 | 2 | 3 | 4 | 5 | 6,
      description: formState.description.trim() || undefined,
      status: props.resource.status as 0 | 1 | 2 | 3,
      links
    })
    message.success('资源已更新')
    emit('updated')
    close()
  } catch (error) {
    message.error(getApiErrorMessage(error, '更新资源失败'))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <a-modal
    :open="open"
    title="编辑资源"
    :confirm-loading="submitting"
    ok-text="保存"
    cancel-text="取消"
    destroy-on-close
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
            RESOURCE_TYPES.map((item) => ({
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
