<script setup lang="ts">
import { message, Modal } from 'ant-design-vue'
import type { TableColumnsType } from 'ant-design-vue'
import { createAppReleaseService } from '~/services/appRelease'
import type {
  AppRelease,
  AppReleasePayload,
  AppReleasePlatform,
  AppReleaseStatus,
  GitHubRelease
} from '~/types/appRelease'

useSeoMeta({ title: '应用版本管理 - Koyomi' })

const releaseService = createAppReleaseService(useNuxtApp().$api)
const { has } = usePermissions()
const items = ref<AppRelease[]>([])
const total = ref(0)
const page = ref(1)
const limit = 20
const loading = ref(false)
const saving = ref(false)
const actionKey = ref('')
const modalOpen = ref(false)
const editing = ref<AppRelease | null>(null)

const githubReleases = ref<GitHubRelease[]>([])
const githubLoading = ref(false)
const githubError = ref('')
const githubTag = ref<string | undefined>(undefined)

const githubOptions = computed(() =>
  githubReleases.value
    .filter((release) => release.apk)
    .map((release) => ({
      label: release.prerelease ? `${release.tag}（预发布）` : release.tag,
      value: release.tag
    }))
)

const APK_NAME_PATTERN = /-(\d+(?:\.\d+){1,3})-(\d+)-[0-9a-f]{7,40}\.apk$/i

const platformOptions: { label: string; value: AppReleasePlatform }[] = [
  { label: 'Android', value: 'android' },
  { label: 'iOS', value: 'ios' },
  { label: 'Windows', value: 'windows' },
  { label: 'macOS', value: 'macos' },
  { label: 'Linux', value: 'linux' }
]
const statusOptions: { label: string; value: AppReleaseStatus }[] = [
  { label: '草稿', value: 'draft' },
  { label: '立即发布', value: 'published' },
  { label: '停用', value: 'disabled' }
]

const emptyForm = () => ({
  platform: 'android' as AppReleasePlatform,
  versionName: '',
  versionCode: 1,
  title: '',
  changelog: '',
  downloadUrl: '',
  fileSize: null as number | null,
  sha256: '',
  minimumVersionCode: 0,
  forceUpdate: false,
  status: 'draft' as AppReleaseStatus,
  publishedAt: '',
  createAnnouncement: false
})
const formState = reactive(emptyForm())

function platformLabel(value: AppReleasePlatform): string {
  return platformOptions.find((option) => option.value === value)?.label ?? value
}

function formatDate(value?: string | null): string {
  return value
    ? new Date(value).toLocaleString('zh-CN', { hour12: false })
    : '-'
}

function toDateInput(value?: string | null): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000)
  return local.toISOString().slice(0, 16)
}

function toIsoDate(value: string): string | null {
  if (!value) return null
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? null : date.toISOString()
}

function statusLabel(status: string): string {
  if (status === 'published') return '已发布'
  if (status === 'disabled') return '已停用'
  return '草稿'
}

function statusColor(status: string): string {
  if (status === 'published') return 'success'
  if (status === 'disabled') return 'warning'
  return 'default'
}

function isHttpUrl(value: string): boolean {
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

function payload(): AppReleasePayload {
  return {
    platform: formState.platform,
    versionName: formState.versionName.trim(),
    versionCode: formState.versionCode,
    title: formState.title.trim(),
    changelog: formState.changelog.trim(),
    downloadUrl: formState.downloadUrl.trim(),
    fileSize: formState.fileSize,
    sha256: formState.sha256.trim() || null,
    minimumVersionCode: formState.minimumVersionCode,
    forceUpdate: formState.forceUpdate,
    createAnnouncement:
      formState.status === 'published' && formState.createAnnouncement,
    status: formState.status,
    publishedAt: toIsoDate(formState.publishedAt),
    announcementId: editing.value?.announcementId ?? null
  }
}

async function load(): Promise<void> {
  loading.value = true
  try {
    const data = await releaseService.listAdmin({ page: page.value, limit })
    items.value = data.items
    total.value = data.total
  } catch (error) {
    message.error(getApiErrorMessage(error, '版本列表加载失败'))
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())
watch(page, () => void load())

function openCreate(): void {
  if (!has('app_release:create')) return
  editing.value = null
  Object.assign(formState, emptyForm())
  githubTag.value = undefined
  void loadGitHubReleases()
  modalOpen.value = true
}

async function openEdit(item: AppRelease): Promise<void> {
  if (!has('app_release:update')) return
  actionKey.value = `edit:${item.id}`
  try {
    const detail = await releaseService.getAdmin(item.id)
    editing.value = detail
    Object.assign(formState, {
      platform: detail.platform,
      versionName: detail.versionName,
      versionCode: detail.versionCode,
      title: detail.title,
      changelog: detail.changelog,
      downloadUrl: detail.downloadUrl,
      fileSize: detail.fileSize ?? null,
      sha256: detail.sha256 ?? '',
      minimumVersionCode: detail.minimumVersionCode,
      forceUpdate: detail.forceUpdate,
      status: detail.status,
      publishedAt: toDateInput(detail.publishedAt),
      createAnnouncement: false
    })
    githubTag.value = undefined
    void loadGitHubReleases()
    modalOpen.value = true
  } catch (error) {
    message.error(getApiErrorMessage(error, '版本加载失败'))
  } finally {
    actionKey.value = ''
  }
}

async function loadGitHubReleases(): Promise<void> {
  if (githubLoading.value || githubReleases.value.length > 0 || githubError.value) {
    return
  }
  githubLoading.value = true
  try {
    githubReleases.value = await releaseService.listGitHubReleases()
  } catch (error) {
    githubError.value = getApiErrorMessage(error, 'GitHub Release 列表加载失败')
  } finally {
    githubLoading.value = false
  }
}

function importGitHubRelease(tag: string | undefined): void {
  const release = githubReleases.value.find((item) => item.tag === tag)
  if (!release?.apk) return
  formState.downloadUrl = release.apk.downloadUrl
  formState.fileSize = release.apk.size
  formState.sha256 = release.apk.sha256 ?? ''
  const match = release.apk.name.match(APK_NAME_PATTERN)
  const [, versionName, build] = match ?? []
  if (versionName && build) {
    formState.versionName = versionName
    formState.versionCode = Number.parseInt(build, 10)
  }
  if (!formState.title.trim()) formState.title = release.name
  if (!formState.changelog.trim() && release.body) {
    formState.changelog = release.body
  }
}

async function submit(): Promise<void> {
  if (
    (editing.value && !has('app_release:update')) ||
    (!editing.value && !has('app_release:create'))
  ) {
    return
  }
  if (
    !formState.versionName.trim() ||
    !formState.title.trim() ||
    !formState.changelog.trim() ||
    !formState.downloadUrl.trim()
  ) {
    message.warning('请填写版本名称、标题、更新内容和下载地址')
    return
  }
  if (
    !Number.isInteger(formState.versionCode) ||
    formState.versionCode < 0 ||
    !Number.isInteger(formState.minimumVersionCode) ||
    formState.minimumVersionCode < 0 ||
    formState.minimumVersionCode > formState.versionCode ||
    (formState.fileSize !== null &&
      (!Number.isInteger(formState.fileSize) || formState.fileSize < 0))
  ) {
    message.warning('版本号和文件大小必须有效，最低版本号不能高于当前版本号')
    return
  }
  if (!isHttpUrl(formState.downloadUrl.trim())) {
    message.warning('下载地址必须是有效的 HTTP/HTTPS URL')
    return
  }
  if (
    formState.sha256.trim() &&
    !/^[0-9a-f]{64}$/.test(formState.sha256.trim())
  ) {
    message.warning('SHA-256 必须是 64 位小写十六进制字符串')
    return
  }

  saving.value = true
  try {
    if (editing.value) {
      await releaseService.update(editing.value.id, payload())
    } else {
      await releaseService.create(payload())
    }
    message.success(editing.value ? '版本已更新' : '版本已创建')
    modalOpen.value = false
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '版本保存失败'))
  } finally {
    saving.value = false
  }
}

async function changeStatus(
  item: AppRelease,
  action: 'publish' | 'disable'
): Promise<void> {
  if (!has('app_release:publish')) return

  if (action === 'publish') {
    Modal.confirm({
      title: '发布版本',
      content: '是否同时创建关联公告？',
      okText: '创建公告并发布',
      cancelText: '仅发布',
      async onOk() {
        actionKey.value = `publish:${item.id}`
        try {
          await releaseService.publish(item.id, true)
          message.success('版本已发布并创建关联公告')
          await load()
        } catch (error) {
          message.error(getApiErrorMessage(error, '版本状态更新失败'))
        } finally {
          actionKey.value = ''
        }
      },
      async onCancel() {
        actionKey.value = `publish:${item.id}`
        try {
          await releaseService.publish(item.id, false)
          message.success('版本已发布')
          await load()
        } catch (error) {
          message.error(getApiErrorMessage(error, '版本状态更新失败'))
        } finally {
          actionKey.value = ''
        }
      }
    })
    return
  }

  actionKey.value = `${action}:${item.id}`
  try {
    await releaseService[action](item.id)
    message.success('版本已停用')
    await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '版本状态更新失败'))
  } finally {
    actionKey.value = ''
  }
}

async function remove(item: AppRelease): Promise<void> {
  if (!has('app_release:delete')) return
  actionKey.value = `delete:${item.id}`
  try {
    await releaseService.remove(item.id)
    message.success('版本已删除')
    if (items.value.length === 1 && page.value > 1) page.value -= 1
    else await load()
  } catch (error) {
    message.error(getApiErrorMessage(error, '版本删除失败'))
  } finally {
    actionKey.value = ''
  }
}

const columns: TableColumnsType = [
  { title: '平台', dataIndex: 'platform', width: 100 },
  { title: '版本 / Build', key: 'version', width: 160 },
  { title: '标题', dataIndex: 'title', width: 200, ellipsis: true },
  { title: '发布时间', dataIndex: 'publishedAt', width: 180 },
  { title: '强制更新', dataIndex: 'forceUpdate', width: 90 },
  { title: '最低版本号', dataIndex: 'minimumVersionCode', width: 110 },
  { title: '下载地址', dataIndex: 'downloadUrl', width: 260, ellipsis: true },
  { title: '状态', dataIndex: 'status', width: 90 },
  { title: '操作', key: 'actions', width: 260, fixed: 'right' }
]
</script>

<template>
  <div>
    <div class="table-toolbar">
      <KunHeader
        name="应用版本"
        description="管理客户端版本、下载信息和更新策略。"
        scale="h3"
      />
      <a-button
        v-if="has('app_release:create')"
        type="primary"
        @click="openCreate"
      >
        新建版本
      </a-button>
    </div>

    <a-table
      :columns="columns"
      :data-source="items"
      :loading="loading"
      :pagination="{
        current: page,
        pageSize: limit,
        total,
        showSizeChanger: false,
        showTotal: (count: number) => `共 ${count} 条`
      }"
      row-key="id"
      :scroll="{ x: 1500 }"
      @change="(pagination: { current?: number }) => { page = pagination.current ?? 1 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'platform'">
          <a-tag>{{ platformLabel(record.platform) }}</a-tag>
        </template>
        <template v-else-if="column.key === 'version'">
          {{ record.versionName }} / {{ record.versionCode }}
        </template>
        <template v-else-if="column.dataIndex === 'publishedAt'">
          {{ formatDate(record.publishedAt) }}
        </template>
        <template v-else-if="column.dataIndex === 'forceUpdate'">
          <a-tag :color="record.forceUpdate ? 'error' : 'default'">
            {{ record.forceUpdate ? '是' : '否' }}
          </a-tag>
        </template>
        <template v-else-if="column.dataIndex === 'downloadUrl'">
          <a-typography-text :title="record.downloadUrl" ellipsis>
            {{ record.downloadUrl }}
          </a-typography-text>
        </template>
        <template v-else-if="column.dataIndex === 'status'">
          <a-tag :color="statusColor(record.status)">
            {{ statusLabel(record.status) }}
          </a-tag>
        </template>
        <template v-else-if="column.key === 'actions'">
          <div class="table-actions">
            <a-button
              v-if="has('app_release:update')"
              size="small"
              :loading="actionKey === `edit:${record.id}`"
              @click="openEdit(record)"
            >
              编辑
            </a-button>
            <a-button
              v-if="record.status !== 'published' && has('app_release:publish')"
              size="small"
              type="primary"
              :loading="actionKey === `publish:${record.id}`"
              @click="changeStatus(record, 'publish')"
            >
              发布
            </a-button>
            <a-button
              v-if="record.status === 'published' && has('app_release:publish')"
              size="small"
              :loading="actionKey === `disable:${record.id}`"
              @click="changeStatus(record, 'disable')"
            >
              停用
            </a-button>
            <a-popconfirm
              v-if="has('app_release:delete')"
              :title="`确定删除版本「${record.versionName}」吗？`"
              ok-text="删除"
              cancel-text="取消"
              @confirm="remove(record)"
            >
              <a-button
                size="small"
                danger
                :loading="actionKey === `delete:${record.id}`"
              >
                删除
              </a-button>
            </a-popconfirm>
          </div>
        </template>
      </template>
    </a-table>

    <a-modal
      v-model:open="modalOpen"
      :title="editing ? '编辑版本' : '新建版本'"
      :confirm-loading="saving"
      width="900px"
      ok-text="保存"
      cancel-text="取消"
      destroy-on-close
      :body-style="{ maxHeight: '72vh', overflowY: 'auto' }"
      @ok="submit"
    >
      <a-form layout="vertical">
        <a-form-item
          label="从 GitHub Release 导入"
          :validate-status="githubError ? 'warning' : undefined"
          :help="githubError || '选择已发布的 GitHub Release，自动填充下载地址、文件大小和校验和'"
        >
          <a-select
            v-model:value="githubTag"
            :options="githubOptions"
            :loading="githubLoading"
            :placeholder="githubError || '选择 GitHub Release'"
            :disabled="!!githubError"
            allow-clear
            @change="(value: string | undefined) => importGitHubRelease(value)"
          />
        </a-form-item>
        <div class="form-grid form-grid-three">
          <a-form-item label="平台" required>
            <a-select
              v-model:value="formState.platform"
              :options="platformOptions"
            />
          </a-form-item>
          <a-form-item label="版本名称" required>
            <a-input
              v-model:value="formState.versionName"
              :maxlength="64"
              placeholder="例如 1.2.0"
            />
          </a-form-item>
          <a-form-item label="版本号（Build）" required>
            <a-input-number
              v-model:value="formState.versionCode"
              :min="0"
              :precision="0"
              class="full-width"
            />
          </a-form-item>
        </div>
        <div class="form-grid">
          <a-form-item label="状态" required>
            <a-select
              v-model:value="formState.status"
              :options="statusOptions"
            />
          </a-form-item>
          <a-form-item label="发布时间">
            <a-input
              v-model:value="formState.publishedAt"
              type="datetime-local"
              :disabled="formState.status !== 'published'"
            />
          </a-form-item>
        </div>
        <a-form-item label="标题" required>
          <a-input v-model:value="formState.title" :maxlength="255" />
        </a-form-item>
        <a-form-item label="更新内容" required>
          <PostEditor
            v-model="formState.changelog"
            mode="markdown"
            :disabled="saving"
          />
        </a-form-item>
        <a-form-item label="下载地址" required>
          <a-input
            v-model:value="formState.downloadUrl"
            :maxlength="2048"
            placeholder="https://example.com/app.apk"
          />
        </a-form-item>
        <div class="form-grid form-grid-three">
          <a-form-item label="文件大小（字节）">
            <a-input-number
              v-model:value="formState.fileSize"
              :min="0"
              :precision="0"
              class="full-width"
            />
          </a-form-item>
          <a-form-item label="最低版本号">
            <a-input-number
              v-model:value="formState.minimumVersionCode"
              :min="0"
              :precision="0"
              class="full-width"
            />
          </a-form-item>
          <a-form-item label="强制更新">
            <a-switch v-model:checked="formState.forceUpdate" />
          </a-form-item>
        </div>
        <a-form-item label="SHA-256">
          <a-input
            v-model:value="formState.sha256"
            :maxlength="64"
            placeholder="可选"
          />
        </a-form-item>
        <a-form-item label="同步创建公告">
          <a-checkbox
            v-model:checked="formState.createAnnouncement"
            :disabled="formState.status !== 'published'"
          >
            发布版本时创建关联公告
          </a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.table-toolbar {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.table-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.form-grid-three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.full-width {
  width: 100%;
}

@media (max-width: 639px) {
  .table-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .form-grid,
  .form-grid-three {
    grid-template-columns: 1fr;
    gap: 0;
  }
}
</style>
