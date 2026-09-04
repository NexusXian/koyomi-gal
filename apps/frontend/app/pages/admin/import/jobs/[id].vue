<script setup lang="ts">
import { message } from 'ant-design-vue'
import { getImportBatch } from '~/api/generated/admin-import/admin-import'
import type { DtoImportJobData } from '~/api/generated/models'

useSeoMeta({ title: '导入任务详情 - Koyomi' })

const route = useRoute()
const jobId = computed(() => Number(route.params.id))

const job = ref<DtoImportJobData | null>(null)
const loading = ref(false)
const notFound = ref(false)
const requestSequence = ref(0)

let timer: ReturnType<typeof setInterval> | null = null

const isTerminal = computed(() => {
  const status = job.value?.status
  return status !== 0 && status !== 1
})

async function load(): Promise<void> {
  if (!Number.isFinite(jobId.value) || jobId.value <= 0) {
    notFound.value = true
    return
  }
  const sequence = ++requestSequence.value
  loading.value = true
  try {
    const data = unwrapApiData(await getImportBatch(jobId.value))
    if (sequence !== requestSequence.value) {
      return
    }
    job.value = data
  } catch (error) {
    if (sequence !== requestSequence.value) {
      return
    }
    notFound.value = true
    message.error(getApiErrorMessage(error, '查询导入任务失败'))
  } finally {
    if (sequence === requestSequence.value) {
      loading.value = false
    }
  }
}

function syncTimer(): void {
  if (!isTerminal.value && timer === null) {
    timer = setInterval(() => {
      void load()
    }, 3000)
  } else if (isTerminal.value && timer !== null) {
    clearInterval(timer)
    timer = null
  }
}

watch(isTerminal, () => syncTimer(), { immediate: true })

onMounted(() => {
  void load()
})

onUnmounted(() => {
  if (timer !== null) {
    clearInterval(timer)
    timer = null
  }
})

function formatDateTime(value?: string): string {
  return value ? value.replace('T', ' ').slice(0, 19) : '-'
}

const paramsRows = computed(() => {
  const params = job.value?.params
  if (!params) {
    return []
  }
  const rows: { key: string; label: string; value: string }[] = []
  rows.push({ key: 'provider', label: '数据源', value: (job.value?.provider ?? '').toUpperCase() })
  rows.push({
    key: 'min_rating',
    label: '最低评分',
    value: params.min_rating != null ? String(params.min_rating) : '未设置'
  })
  rows.push({
    key: 'min_vote_count',
    label: '最低投票数',
    value: params.min_vote_count != null ? String(params.min_vote_count) : '未设置'
  })
  rows.push({
    key: 'from_year',
    label: '起始年份',
    value: params.from_year != null ? String(params.from_year) : '未设置'
  })
  rows.push({
    key: 'to_year',
    label: '结束年份',
    value: params.to_year != null ? String(params.to_year) : '未设置'
  })
  rows.push({
    key: 'original_language',
    label: '原始语言',
    value: params.original_language || '全部'
  })
  rows.push({
    key: 'limit',
    label: '最大数量',
    value: String(params.limit ?? '-')
  })
  return rows
})
</script>

<template>
  <div class="import-job-detail">
    <div class="import-job-detail-head">
      <a-button @click="navigateTo('/admin/import')">
        返回导入首页
      </a-button>
      <a-button :loading="loading" @click="load">刷新</a-button>
    </div>

    <KunNull v-if="notFound" text="导入任务不存在" />

    <template v-else-if="job">
      <KunCard padding="md" class="import-job-detail-section">
        <h3 class="import-job-detail-title">
          任务 #{{ job.id }}（{{ (job.provider ?? '').toUpperCase() }} 批量导入）
        </h3>
        <ImportJobProgress :job="job" />
        <a-alert
          v-if="job.status === 3 && job.error_message"
          class="import-job-detail-error"
          type="error"
          show-icon
          :message="job.error_message"
        />
      </KunCard>

      <KunCard padding="md" class="import-job-detail-section">
        <h3 class="import-job-detail-title">任务参数</h3>
        <a-descriptions :column="1" size="small" bordered>
          <a-descriptions-item
            v-for="row in paramsRows"
            :key="row.key"
            :label="row.label"
          >
            {{ row.value }}
          </a-descriptions-item>
          <a-descriptions-item label="创建者">
            {{ job.created_by ? `用户 #${job.created_by}` : '系统' }}
          </a-descriptions-item>
          <a-descriptions-item label="创建时间">
            {{ formatDateTime(job.created_at) }}
          </a-descriptions-item>
          <a-descriptions-item label="开始时间">
            {{ formatDateTime(job.started_at) }}
          </a-descriptions-item>
          <a-descriptions-item label="结束时间">
            {{ formatDateTime(job.finished_at) }}
          </a-descriptions-item>
        </a-descriptions>
      </KunCard>
    </template>

    <a-spin v-else :spinning="loading" class="import-job-detail-loading">
      <KunNull text="加载中" />
    </a-spin>
  </div>
</template>

<style scoped>
.import-job-detail-head {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}

.import-job-detail-section {
  margin-bottom: 18px;
}

.import-job-detail-title {
  margin: 0 0 14px;
  font-size: 16px;
}

.import-job-detail-error {
  margin-top: 14px;
}

.import-job-detail-loading {
  display: block;
  padding: 40px 0;
}
</style>
