<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { Rule } from 'ant-design-vue/es/form'
import {
  AGE_RATINGS,
  GALGAME_STATUS
} from '~/constants/domain'
import { listDevelopers } from '~/api/generated/developers/developers'
import { listTags } from '~/api/generated/tags/tags'
import {
  createGalgame,
  getGalgame,
  updateGalgame
} from '~/api/generated/galgames/galgames'
import { getAdminGalgame } from '~/api/generated/admin/admin'
import type {
  DtoDeveloperSummary,
  DtoTagSummary
} from '~/api/generated/models'
import type {
  GameDescriptions,
  GalgameDetailData,
  GalgameUpdatePayload
} from '~/types/galgame'
import type { ImageAsset } from '~/types/image'
import {
  DESCRIPTION_LOCALES,
  type DescriptionLocale
} from '~/utils/gameDescriptions'

// Cherry Markdown is heavy; only pull its chunk when the form mounts.
const MarkdownEditor = defineAsyncComponent(
  () => import('~/components/post/MarkdownEditor.vue')
)

const props = defineProps<{
  galgameId?: number
}>()

const emit = defineEmits<{
  submitted: [galgame: GalgameDetailData]
}>()

const router = useRouter()
const { has } = usePermissions()
const { initialize } = useAuth()
const { isAuthenticated } = storeToRefs(useUserStore())

const formState = reactive({
  title: '',
  slug: '',
  romaji_title: '',
  original_title: '',
  developer_id: undefined as number | undefined,
  release_date: '' as string,
  age_rating: 0,
  cover_sensitive: false,
  status: 0,
  tag_ids: [] as number[],
  aliases: [] as string[],
  cover_url: '',
  banner_url: ''
})

// Per-language description editor state; languages are fully independent.
interface DescriptionDraft {
  content: string
  source_type: string
  source_name: string
  source_url: string
  is_official: boolean
}

const LOCALE_TABS: { locale: DescriptionLocale; label: string }[] = [
  { locale: 'zh-CN', label: '中文' },
  { locale: 'en-US', label: 'English' },
  { locale: 'ja-JP', label: '日本語' }
]

const SOURCE_TYPE_OPTIONS: { value: string; label: string }[] = [
  { value: 'nextmoe', label: 'NextMoe' },
  { value: 'vndb', label: 'VNDB' },
  { value: 'official', label: '游戏官网' },
  { value: 'bangumi', label: 'Bangumi' },
  { value: 'steam', label: 'Steam' },
  { value: 'manual', label: '手动录入' },
  { value: 'unknown', label: '其他' }
]

const SOURCE_TYPE_DEFAULT_NAMES: Record<string, string> = {
  nextmoe: 'NextMoe 资料库',
  vndb: 'VNDB',
  official: '游戏官网',
  bangumi: 'Bangumi',
  steam: 'Steam',
  manual: '手动录入',
  unknown: '未知来源'
}

function defaultDraft(locale: DescriptionLocale): DescriptionDraft {
  switch (locale) {
    case 'zh-CN':
      return {
        content: '',
        source_type: 'nextmoe',
        source_name: 'NextMoe 资料库',
        source_url: '',
        is_official: false
      }
    case 'en-US':
      return {
        content: '',
        source_type: 'vndb',
        source_name: 'VNDB',
        source_url: '',
        is_official: false
      }
    default:
      return {
        content: '',
        source_type: 'official',
        source_name: '游戏官网',
        source_url: '',
        is_official: true
      }
  }
}

interface DescriptionDrafts {
  'zh-CN': DescriptionDraft
  'en-US': DescriptionDraft
  'ja-JP': DescriptionDraft
}

const descriptionDrafts = reactive<DescriptionDrafts>({
  'zh-CN': defaultDraft('zh-CN'),
  'en-US': defaultDraft('en-US'),
  'ja-JP': defaultDraft('ja-JP')
})
const activeDescriptionTab = ref<DescriptionLocale>('zh-CN')

function applyDescriptionSourceType(
  locale: DescriptionLocale,
  sourceType: string
): void {
  const draft = descriptionDrafts[locale]
  draft.source_type = sourceType
  draft.source_name = SOURCE_TYPE_DEFAULT_NAMES[sourceType] ?? ''
  draft.is_official = sourceType === 'official'
}

function loadDescriptionDrafts(descriptions?: GameDescriptions): void {
  for (const locale of DESCRIPTION_LOCALES) {
    const stored = descriptions?.[locale]
    const draft = defaultDraft(locale)
    if (stored) {
      draft.content = stored.content ?? ''
      if (stored.source?.type) {
        draft.source_type = stored.source.type
        draft.source_name =
          stored.source.name ?? SOURCE_TYPE_DEFAULT_NAMES[stored.source.type] ?? ''
        draft.source_url = stored.source.url ?? ''
        draft.is_official = stored.source.official ?? false
      }
    }
    Object.assign(descriptionDrafts[locale], draft)
  }
}

const developers = ref<DtoDeveloperSummary[]>([])
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
  ]
}

const canReview = computed(() => has('galgame:review'))
const canUploadImages = computed(() => has('image:manage'))

function onCoverUploaded(asset: ImageAsset): void {
  formState.cover_url = asset.url
  message.success('封面已上传')
}

function onBannerUploaded(asset: ImageAsset): void {
  formState.banner_url = asset.url
  message.success('横幅已上传')
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
    if (!props.galgameId && !formState.slug) {
      formState.slug = slugify(title)
    }
  }
)

async function loadOptions(): Promise<void> {
  try {
    const [developerData, tagData] = await Promise.all([
      listDevelopers(),
      listTags()
    ])
    developers.value =
      (unwrapApiData(developerData) ?? []).map((item) => ({
        id: item.id,
        name: item.name
      })) ?? []
    tags.value =
      (unwrapApiData(tagData) ?? []).map((item) => ({
        id: item.id,
        name: item.name
      })) ?? []
  } catch {
    message.warning('开发商或 Tag 列表加载失败，可稍后刷新重试')
  }
}

async function loadGalgame(): Promise<void> {
  if (!props.galgameId) {
    return
  }

  loading.value = true
  try {
    let data: GalgameDetailData
    try {
      data = unwrapApiData(await getGalgame(props.galgameId))
    } catch {
      // 待审核等未发布条目对公开接口返回 404，回退到管理端接口（需 galgame:review）
      data = unwrapApiData(await getAdminGalgame(props.galgameId))
    }
    loadDescriptionDrafts(data.descriptions)
    Object.assign(formState, {
      title: data.title ?? '',
      slug: data.slug ?? '',
      romaji_title: data.romaji_title ?? '',
      original_title: data.original_title ?? '',
      developer_id: data.developer?.id,
      release_date: data.release_date ? data.release_date.slice(0, 10) : '',
      age_rating: data.age_rating ?? 0,
      cover_sensitive: data.cover_sensitive ?? false,
      status: data.status ?? 0,
      tag_ids: (data.tags ?? []).map((tag) => tag.id).filter(Boolean),
      aliases: data.aliases ?? [],
      cover_url: data.cover_url ?? '',
      banner_url: data.banner_url ?? ''
    })
  } catch (error) {
    message.error(getApiErrorMessage(error, '加载 Galgame 失败'))
    void router.replace('/galgames')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  // Wait for the in-flight session refresh (auth-init plugin) so a full page
  // load is not misread as "not logged in".
  await initialize()

  if (!isAuthenticated.value) {
    message.warning('登录后才能创建或编辑 Galgame')
    void router.replace('/login')
    return
  }

  void loadOptions()
  void loadGalgame()
})

async function submit(): Promise<void> {
  submitting.value = true

  const descriptions = DESCRIPTION_LOCALES.map((locale) => {
    const draft = descriptionDrafts[locale]
    return {
      language: locale,
      content: draft.content.trim(),
      source_type: draft.source_type,
      source_name: draft.source_name.trim(),
      source_url: draft.source_url.trim(),
      is_official: draft.is_official
    }
  })

  const payload: GalgameUpdatePayload = {
    title: formState.title.trim(),
    slug: formState.slug.trim(),
    romaji_title: formState.romaji_title.trim() || undefined,
    original_title: formState.original_title.trim() || undefined,
    developer_id: formState.developer_id,
    release_date: formState.release_date || undefined,
    age_rating: formState.age_rating as 0 | 1 | 2 | 3 | 4 | 5,
    cover_sensitive: formState.cover_sensitive,
    status: formState.status as 0 | 1 | 2 | 3,
    tag_ids: formState.tag_ids,
    aliases: formState.aliases,
    cover_url: formState.cover_url.trim() || undefined,
    banner_url: formState.banner_url.trim() || undefined,
    // 三种语言相互独立，同时保留旧字段兼容旧版后端。
    descriptions,
    description: descriptionDrafts['zh-CN'].content.trim() || undefined
  }

  try {
    const response = props.galgameId
      ? await updateGalgame(props.galgameId, payload)
      : await createGalgame(payload)
    const data = unwrapApiData(response, '保存失败')
    message.success(props.galgameId ? 'Galgame 已更新' : 'Galgame 已创建')
    emit('submitted', data)
    if (data.status === 1) {
      void router.push(`/galgames/${data.id}`)
    } else {
      void router.push('/')
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
              placeholder="例如：CLANNAD"
              :maxlength="255"
            />
          </a-form-item>

          <a-form-item label="Slug" name="slug">
            <a-input
              v-model:value="formState.slug"
              placeholder="例如：clannad"
              :maxlength="255"
            />
          </a-form-item>

          <a-form-item label="罗马音标题">
            <a-input v-model:value="formState.romaji_title" :maxlength="255" />
          </a-form-item>

          <a-form-item label="原名">
            <a-input
              v-model:value="formState.original_title"
              :maxlength="255"
            />
          </a-form-item>

          <a-form-item label="开发商">
            <a-select
              v-model:value="formState.developer_id"
              :options="
                developers.map((item) => ({
                  value: item.id,
                  label: item.name
                }))
              "
              placeholder="选择开发商"
              allow-clear
              show-search
              option-filter-prop="label"
            />
          </a-form-item>

          <a-form-item label="发行日期">
            <a-input
              v-model:value="formState.release_date"
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
              <a-switch v-model:checked="formState.cover_sensitive" />
              <span class="cover-sensitive-text">
                此封面可能不适合在公共环境展示
              </span>
            </div>
            <span class="cover-sensitive-help">
              此设置与游戏年龄等级无关。启用后，前台默认对该游戏封面进行模糊处理。
            </span>
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

          <a-form-item label="封面">
            <div class="image-field">
              <ImageUploader
                v-if="canUploadImages"
                category="galgames"
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

          <a-form-item label="横幅">
            <div class="image-field">
              <ImageUploader
                v-if="canUploadImages"
                category="galgames"
                :preview-url="formState.banner_url || null"
                width="112px"
                height="84px"
                @success="onBannerUploaded"
                @remove="formState.banner_url = ''"
              />
              <a-input
                v-model:value="formState.banner_url"
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
                GALGAME_STATUS.map((item) => ({
                  value: item.value,
                  label: item.label
                }))
              "
            />
          </a-form-item>
        </div>

        <a-form-item label="别名">
          <KunTagInput
            v-model="formState.aliases"
            placeholder="输入别名后回车"
            :max-tags="100"
          />
        </a-form-item>

        <a-form-item label="多语言简介（支持 Markdown）">
          <a-tabs v-model:active-key="activeDescriptionTab">
            <a-tab-pane
              v-for="tab in LOCALE_TABS"
              :key="tab.locale"
              :tab="tab.label"
            >
              <ClientOnly>
                <MarkdownEditor
                  v-model="descriptionDrafts[tab.locale].content"
                  upload-category="galgames"
                />
                <template #fallback>
                  <a-textarea
                    v-model:value="descriptionDrafts[tab.locale].content"
                    :rows="8"
                    placeholder="填写简介，支持 Markdown"
                  />
                </template>
              </ClientOnly>

              <div class="description-source-form">
                <a-form-item label="来源类型" class="description-source-field">
                  <a-select
                    :value="descriptionDrafts[tab.locale].source_type"
                    :options="
                      SOURCE_TYPE_OPTIONS.map((item) => ({
                        value: item.value,
                        label: item.label
                      }))
                    "
                    @change="
                      (value: string) =>
                        applyDescriptionSourceType(tab.locale, value)
                    "
                  />
                </a-form-item>
                <a-form-item label="来源名称" class="description-source-field">
                  <a-input
                    v-model:value="descriptionDrafts[tab.locale].source_name"
                    :maxlength="128"
                  />
                </a-form-item>
                <a-form-item label="来源链接" class="description-source-field-wide">
                  <a-input
                    v-model:value="descriptionDrafts[tab.locale].source_url"
                    placeholder="https://"
                  />
                </a-form-item>
                <a-form-item label="官方来源" class="description-source-field">
                  <a-switch
                    v-model:checked="descriptionDrafts[tab.locale].is_official"
                  />
                </a-form-item>
              </div>
              <span class="description-source-help">
                清空内容会保留记录与来源信息；三种语言的简介互不影响。
              </span>
            </a-tab-pane>
          </a-tabs>
        </a-form-item>

        <a-form-item>
          <div class="form-actions">
            <a-button
              type="primary"
              size="large"
              html-type="submit"
              :loading="submitting"
            >
              {{ galgameId ? '保存修改' : '创建 Galgame' }}
            </a-button>
            <KunButton
              color="default"
              variant="bordered"
              href="/galgames"
            >
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

.description-source-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 16px;
  margin-top: 12px;
}

.description-source-field :deep(.ant-form-item-label > label) {
  font-size: 13px;
}

.description-source-help {
  display: block;
  margin-top: 4px;
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
