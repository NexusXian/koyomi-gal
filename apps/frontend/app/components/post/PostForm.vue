<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { Rule } from 'ant-design-vue/es/form'
import {
  createPost,
  getPost,
  updatePost
} from '~/api/generated/posts/posts'
import { listGalgames } from '~/api/generated/galgames/galgames'
import type { DtoGalgameListItem, DtoPostData } from '~/api/generated/models'
import { EDITOR_MODE_OPTIONS, normalizeEditorMode, type EditorMode } from '~/types/post'

const props = defineProps<{
  postId?: number
  presetGalgameId?: number
}>()

const emit = defineEmits<{
  submitted: [post: DtoPostData]
}>()

const router = useRouter()
const { isAuthenticated } = storeToRefs(useUserStore())

const formState = reactive({
  title: '',
  content: '',
  editor_mode: 'plain' as EditorMode,
  galgame_id: undefined as number | undefined
})

const galgameOptions = ref<{ value: number; label: string }[]>([])
const searching = ref(false)
const submitting = ref(false)
const loading = ref(false)

const editorModeOptions = EDITOR_MODE_OPTIONS

const rules: Record<string, Rule[]> = {
  title: [{ required: true, message: '请输入标题', max: 255 }],
  content: [{ required: true, message: '请输入内容' }]
}

async function searchGalgames(keyword: string): Promise<void> {
  searching.value = true
  try {
    const data = unwrapApiData(
      await listGalgames({ keyword: keyword || undefined, limit: 20 })
    )
    galgameOptions.value = (data.items ?? []).map((item: DtoGalgameListItem) => ({
      value: item.id ?? 0,
      label: item.title ?? `#${item.id}`
    }))
  } catch {
    galgameOptions.value = []
  } finally {
    searching.value = false
  }
}

async function loadPost(): Promise<void> {
  if (!props.postId) {
    return
  }

  loading.value = true
  try {
    const data = unwrapApiData(await getPost(props.postId))
    formState.title = data.title ?? ''
    formState.content = data.content ?? ''
    formState.editor_mode = normalizeEditorMode(data.editor_mode)
    formState.galgame_id = data.galgame_id
    if (data.galgame_id) {
      await searchGalgames('')
    }
  } catch (error) {
    message.error(getApiErrorMessage(error, '加载帖子失败'))
    void router.replace('/posts')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  if (!isAuthenticated.value) {
    message.warning('登录后才能发帖')
    void router.replace('/login')
    return
  }

  if (props.presetGalgameId) {
    formState.galgame_id = props.presetGalgameId
  }

  void searchGalgames('')
  void loadPost()
})

async function submit(): Promise<void> {
  submitting.value = true

  try {
    const payload = {
      title: formState.title.trim(),
      content: formState.content,
      editor_mode: formState.editor_mode,
      galgame_id: formState.galgame_id
    }
    const response = props.postId
      ? await updatePost(props.postId, payload)
      : await createPost(payload)
    const data = unwrapApiData(response, '保存失败')
    message.success(props.postId ? '帖子已更新' : '发布成功')
    emit('submitted', data)
    void router.push(`/posts/${data.id}`)
  } catch (error) {
    message.error(getApiErrorMessage(error, '保存失败'))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <KunCard padding="lg">
    <a-spin :spinning="loading">
      <a-form
        layout="vertical"
        :model="formState"
        :rules="rules"
        @finish="submit"
      >
        <a-form-item label="标题" name="title">
          <a-input
            v-model:value="formState.title"
            placeholder="帖子标题"
            :maxlength="255"
            show-count
          />
        </a-form-item>

        <a-form-item label="关联 Galgame">
          <a-select
            v-model:value="formState.galgame_id"
            :options="galgameOptions"
            placeholder="搜索并选择 Galgame（可选）"
            show-search
            allow-clear
            :filter-option="false"
            :loading="searching"
            @search="searchGalgames"
          />
        </a-form-item>

        <a-form-item label="编辑模式">
          <a-segmented
            v-model:value="formState.editor_mode"
            :options="editorModeOptions"
            :disabled="submitting"
          />
        </a-form-item>

        <a-form-item label="内容" name="content">
          <PostEditor
            v-model="formState.content"
            :mode="formState.editor_mode"
            :disabled="submitting"
          />
        </a-form-item>

        <a-form-item>
          <div class="form-actions">
            <a-button
              type="primary"
              size="large"
              html-type="submit"
              :loading="submitting"
            >
              {{ postId ? '保存修改' : '发布帖子' }}
            </a-button>
            <KunButton color="default" variant="bordered" href="/posts">
              返回列表
            </KunButton>
          </div>
        </a-form-item>
      </a-form>
    </a-spin>
  </KunCard>
</template>

<style scoped>
.form-actions {
  display: flex;
  gap: 10px;
  margin-bottom: 0;
}
</style>
