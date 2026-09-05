<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { Rule } from 'ant-design-vue/es/form'
import {
  AGE_RATINGS,
  NOVEL_RELEASE_STATUS,
  NOVEL_STATUS
} from '~/constants/domain'
import { listTags } from '~/api/generated/tags/tags'
import {
  createNovel,
  getNovel,
  updateNovel
} from '~/api/generated/novels/novels'
import { getAdminNovel } from '~/api/generated/admin/admin'
import type {
  DtoCreateNovelRequestReleaseStatus,
  DtoNovelResponse,
  DtoTagSummary,
  DtoUpdateNovelRequestReleaseStatus
} from '~/api/generated/models'
import type { ImageAsset } from '~/types/image'

// Cherry Markdown is heavy; only pull its chunk when the form mounts.
const MarkdownEditor = defineAsyncComponent(
  () => import('~/components/post/MarkdownEditor.vue')
)

const props = defineProps<{
  novelId?: number
}>()

const emit = defineEmits<{
  submitted: [novel: DtoNovelResponse]
}>()

const router = useRouter()
const { has } = usePermissions()
const { initialize } = useAuth()
const { isAuthenticated } = storeToRefs(useUserStore())

const formState = reactive({
  title: '',
  slug: '',
  original_title: '',
  author: '',
  illustrator: '',
  publisher: '',
  label: '',
  language: '',
  region: '',
  release_status: 'unknown' as DtoUpdateNovelRequestReleaseStatus,
  first_release_date: '' as string,
  age_rating: 0,
  is_cover_sensitive: false,
  official_website: '',
  status: 0,
  tag_ids: [] as number[],
  cover_url: '',
  summary: ''
})

const tags = ref<DtoTagSummary[]>([])
const submitting = ref(false)
const loading = ref(false)

const rules: Record<string, Rule[]> = {
  title: [{ required: true, message: '请输入标题' }],
  slug: [
    { required: true, message: '请输入 Slug' },
    {
      pattern: /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
      message: 'Slug 只能包含小写字母、数字和中划线'
    }
  ],
  cover_url: [
    {
      validator: (_rule: Rule, value: string) =>
        !value || /^https?:\/\//.test(value)
          ? Promise.resolve()
          : Promise.reject('封面 URL 必须以 http:// 或 https:// 开头')
    }
  ],
  official_website: [
    {
      validator: (_rule: Rule, value: string) =>
        !value || /^https?:\/\//.test(value)
          ? Promise.resolve()
          : Promise.reject('官方网站 URL 必须以 http:// 或 https:// 开头')
    }
  ]
}

const canReview = computed(() => has('novel:review'))
const canUploadImages = computed(() => has('image:manage'))

function onCoverUploaded(asset: ImageAsset): void {
  formState.cover_url = asset.url
  message.success('封面已上传')
}

function slugify(value: string): string {
  return value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9\s-]/g, '')
    .replace(/[\s_]+/g, '-')
    .replace(/-+/g, '-')
}

watch(
  () => formState.title,
  (title) => {
    if (!props.novelId && !formState.slug) {
      formState.slug = slugify(title)
    }
  }
)

async function loadOptions(): Promise<void> {
  try {
    const tagData = await listTags()
    tags.value =
      (unwrapApiData(tagData) ?? []).map((item) => ({
        id: item.id,
        name: item.name
      })) ?? []
  } catch {
    message.warning('Tag 列表加载失败，可稍后刷新重试')
  }
}

async function loadNovel(): Promise<void> {
  if (!props.novelId) {
    return
  }

  loading.value = true
  try {
    let data: DtoNovelResponse
    try {
      data = unwrapApiData(await getNovel(props.novelId))
    } catch {
      // 待审核等未发布条目对公开接口返回 404，回退到管理端接口（需 novel:review）
      data = unwrapApiData(await getAdminNovel(props.novelId))
    }
    Object.assign(formState, {
      title: data.title ?? '',
      slug: data.slug ?? '',
      original_title: data.original_title ?? '',
      author: data.author ?? '',
      illustrator: data.illustrator ?? '',
      publisher: data.publisher ?? '',
      label: data.label ?? '',
      language: data.language ?? '',
      region: data.region ?? '',
      release_status: data.release_status ?? 'unknown',
      first_release_date: data.first_release_date
        ? data.first_release_date.slice(0, 10)
        : '',
      age_rating: data.age_rating ?? 0,
      is_cover_sensitive: data.is_cover_sensitive ?? false,
      official_website: data.official_website ?? '',
      status: data.status ?? 0,
      tag_ids: (data.tags ?? []).map((tag) => tag.id).filter(Boolean),
      cover_url: data.cover_url ?? '',
      summary: data.summary ?? ''
    })
  } catch (error) {
    message.error(getApiErrorMessage(error, '加载小说失败'))
    void router.replace('/novels')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  // Wait for the in-flight session refresh (auth-init plugin) so a full page
  // load is not misread as "not logged in".
  await initialize()

  if (!isAuthenticated.value) {
    message.warning('登录后才能创建或编辑小说')
    void router.replace('/login')
    return
  }

  void loadOptions()
  void loadNovel()
})

async function submit(): Promise<void> {
  submitting.value = true

  try {
    const response = props.novelId
      ? await updateNovel(props.novelId, {
          title: formState.title.trim(),
          slug: formState.slug.trim(),
          original_title: formState.original_title.trim() || undefined,
          author: formState.author.trim() || undefined,
          illustrator: formState.illustrator.trim() || undefined,
          publisher: formState.publisher.trim() || undefined,
          label: formState.label.trim() || undefined,
          language: formState.language.trim() || undefined,
          region: formState.region.trim() || undefined,
          release_status: formState.release_status as DtoUpdateNovelRequestReleaseStatus,
          first_release_date: formState.first_release_date || undefined,
          age_rating: formState.age_rating as 0 | 1 | 2 | 3 | 4 | 5,
          is_cover_sensitive: formState.is_cover_sensitive,
          official_website: formState.official_website.trim() || undefined,
          status: formState.status as 0 | 1 | 2 | 3,
          tag_ids: formState.tag_ids,
          cover_url: formState.cover_url.trim() || undefined,
          summary: formState.summary.trim() || undefined
        })
      : await createNovel({
          title: formState.title.trim(),
          slug: formState.slug.trim(),
          original_title: formState.original_title.trim() || undefined,
          author: formState.author.trim() || undefined,
          illustrator: formState.illustrator.trim() || undefined,
          publisher: formState.publisher.trim() || undefined,
          label: formState.label.trim() || undefined,
          language: formState.language.trim() || undefined,
          region: formState.region.trim() || undefined,
          release_status: formState.release_status as DtoCreateNovelRequestReleaseStatus,
          first_release_date: formState.first_release_date || undefined,
          age_rating: formState.age_rating as 0 | 1 | 2 | 3 | 4 | 5,
          is_cover_sensitive: formState.is_cover_sensitive,
          official_website: formState.official_website.trim() || undefined,
          status: formState.status as 0 | 1,
          tag_ids: formState.tag_ids,
          cover_url: formState.cover_url.trim() || undefined,
          summary: formState.summary.trim() || undefined
        })
    const data = unwrapApiData(response, '保存失败')
    message.success(props.novelId ? '小说已更新' : '小说已创建')
    emit('submitted', data)
    if (data.status === 1) {
      void router.push(`/novels/${data.id}`)
    } else {
      void router.push('/novels')
    }
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
        <div class="form-grid">
          <a-form-item label="标题" name="title">
            <a-input
              v-model:value="formState.title"
              placeholder="例如：青春猪头少年不会梦到兔女郎学姐"
              :maxlength="255"
            />
          </a-form-item>

          <a-form-item label="Slug" name="slug">
            <a-input
              v-model:value="formState.slug"
              placeholder="例如：seishun-buta-yarou"
              :maxlength="255"
            />
          </a-form-item>

          <a-form-item label="原文标题">
            <a-input
              v-model:value="formState.original_title"
              placeholder="例如：青春ブタ野郎はバニーガール先輩の夢を見ない"
              :maxlength="255"
            />
          </a-form-item>

          <a-form-item label="作者">
            <a-input
              v-model:value="formState.author"
              placeholder="例如：鸭志田一"
              :maxlength="255"
            />
          </a-form-item>

          <a-form-item label="插画师">
            <a-input
              v-model:value="formState.illustrator"
              placeholder="例如：沟口凯吉"
              :maxlength="255"
            />
          </a-form-item>

          <a-form-item label="出版社">
            <a-input
              v-model:value="formState.publisher"
              placeholder="例如：KADOKAWA"
              :maxlength="255"
            />
          </a-form-item>

          <a-form-item label="文库 / Label">
            <a-input
              v-model:value="formState.label"
              placeholder="例如：电击文库"
              :maxlength="255"
            />
          </a-form-item>

          <a-form-item label="语言">
            <a-select
              v-model:value="formState.language"
              :options="[
                { value: 'ja', label: '日语 (ja)' },
                { value: 'zh-CN', label: '简体中文 (zh-CN)' },
                { value: 'zh-TW', label: '繁体中文 (zh-TW)' },
                { value: 'en', label: '英语 (en)' },
                { value: 'ko', label: '韩语 (ko)' }
              ]"
              allow-clear
            />
          </a-form-item>

          <a-form-item label="地区">
            <a-select
              v-model:value="formState.region"
              :options="[
                { value: 'JP', label: '日本 (JP)' },
                { value: 'CN', label: '中国大陆 (CN)' },
                { value: 'TW', label: '中国台湾 (TW)' },
                { value: 'HK', label: '中国香港 (HK)' },
                { value: 'US', label: '美国 (US)' },
                { value: 'KR', label: '韩国 (KR)' }
              ]"
              allow-clear
            />
          </a-form-item>

          <a-form-item label="连载状态">
            <a-select
              v-model:value="formState.release_status"
              :options="
                NOVEL_RELEASE_STATUS.map((item) => ({
                  value: item.slug,
                  label: item.label
                }))
              "
            />
          </a-form-item>

          <a-form-item label="首次发售日期">
            <a-input
              v-model:value="formState.first_release_date"
              placeholder="YYYY-MM-DD"
            />
          </a-form-item>

          <a-form-item label="年龄等级">
            <a-select
              v-model:value="formState.age_rating"
              :options="
                AGE_RATINGS.map((item) => ({
                  value: item.value,
                  label: item.label
                }))
              "
            />
          </a-form-item>

          <a-form-item label="敏感封面">
            <div class="cover-sensitive-field">
              <a-switch v-model:checked="formState.is_cover_sensitive" />
              <span class="cover-sensitive-text">
                此封面可能不适合在公共环境展示
              </span>
            </div>
            <span class="cover-sensitive-help">
              此设置与年龄等级无关。启用后，前台默认对该小说封面进行模糊处理。
            </span>
          </a-form-item>

          <a-form-item label="官方网站" name="official_website">
            <a-input
              v-model:value="formState.official_website"
              placeholder="https://..."
            />
          </a-form-item>

          <a-form-item label="Tags">
            <a-select
              v-model:value="formState.tag_ids"
              :options="
                tags.map((item) => ({ value: item.id, label: item.name }))
              "
              mode="multiple"
              placeholder="选择 Tag"
              :max-tag-count="8"
              option-filter-prop="label"
              allow-clear
            />
          </a-form-item>

          <a-form-item label="封面" name="cover_url">
            <div class="image-field">
              <ImageUploader
                v-if="canUploadImages"
                category="novels"
                :preview-url="formState.cover_url || null"
                width="112px"
                height="84px"
                @success="onCoverUploaded"
                @remove="formState.cover_url = ''"
              />
              <a-input
                v-model:value="formState.cover_url"
                placeholder="https:// 或上传后自动填充"
              />
            </div>
          </a-form-item>

          <a-form-item
            v-if="canReview"
            label="状态（需要审核权限）"
            name="status"
          >
            <a-select
              v-model:value="formState.status"
              :options="
                NOVEL_STATUS.filter((item) => item.value === 0 || item.value === 1).map(
                  (item) => ({
                    value: item.value,
                    label: item.label
                  })
                )
              "
            />
          </a-form-item>
        </div>

        <a-form-item label="简介（支持 Markdown）">
          <ClientOnly>
            <MarkdownEditor
              v-model="formState.summary"
              upload-category="novels"
            />
            <template #fallback>
              <a-textarea
                :value="formState.summary"
                :rows="8"
                placeholder="填写小说简介，支持 Markdown"
                @update:value="
                  (value: string) => {
                    formState.summary = value
                  }
                "
              />
            </template>
          </ClientOnly>
        </a-form-item>

        <a-form-item>
          <div class="form-actions">
            <a-button
              type="primary"
              size="large"
              html-type="submit"
              :loading="submitting"
            >
              {{ novelId ? '保存修改' : '创建小说' }}
            </a-button>
            <KunButton color="default" variant="bordered" href="/novels">
              返回列表
            </KunButton>
          </div>
        </a-form-item>
      </a-form>
    </a-spin>
  </KunCard>
</template>

<style scoped>
.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 20px;
}

.image-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cover-sensitive-field {
  display: flex;
  align-items: center;
  gap: 10px;
}

.cover-sensitive-text {
  font-size: 14px;
}

.cover-sensitive-help {
  display: block;
  margin-top: 6px;
  color: var(--color-default-500);
  font-size: 12px;
  line-height: 1.6;
}

.form-actions {
  display: flex;
  gap: 10px;
  margin-bottom: 0;
}

@media (min-width: 768px) {
  .form-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
</style>
