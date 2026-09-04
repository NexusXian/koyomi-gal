<script setup lang="ts">
import type { DtoImportJobData } from '~/api/generated/models'

const props = defineProps<{
  job: DtoImportJobData
  compact?: boolean
}>()

const STATUS_COLORS: Record<number, string> = {
  0: 'default',
  1: 'processing',
  2: 'success',
  3: 'error',
  4: 'warning'
}

const STATUS_LABELS: Record<number, string> = {
  0: '待处理',
  1: '运行中',
  2: '成功',
  3: '失败',
  4: '已取消'
}

const status = computed(() => props.job.status ?? 0)
const statusLabel = computed(
  () => STATUS_LABELS[status.value] ?? props.job.status_label ?? '未知'
)
const percent = computed(() => {
  const total = props.job.total_count ?? 0
  const processed = props.job.processed_count ?? 0
  if (total <= 0) {
    return 0
  }
  return Math.min(100, Math.round((processed / total) * 100))
})
const percentStatus = computed(() =>
  status.value === 3 ? 'exception' : status.value === 2 ? 'success' : 'active'
)
</script>

<template>
  <div class="import-job-progress">
    <div class="import-job-progress-head">
      <a-tag :color="STATUS_COLORS[status]">{{ statusLabel }}</a-tag>
      <span class="import-job-progress-count">
        {{ job.processed_count ?? 0 }} / {{ job.total_count ?? 0 }}
      </span>
    </div>
    <a-progress
      v-if="!compact"
      :percent="percent"
      :status="percentStatus"
      size="small"
    />
    <div class="import-job-progress-stats">
      <span class="import-job-stat import-job-stat-created">
        新增 {{ job.created_count ?? 0 }}
      </span>
      <span class="import-job-stat import-job-stat-skipped">
        跳过 {{ job.skipped_count ?? 0 }}
      </span>
      <span class="import-job-stat import-job-stat-failed">
        失败 {{ job.failed_count ?? 0 }}
      </span>
    </div>
  </div>
</template>

<style scoped>
.import-job-progress {
  min-width: 170px;
}

.import-job-progress-head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.import-job-progress-count {
  font-size: 13px;
}

.import-job-progress-stats {
  display: flex;
  gap: 10px;
  margin-top: 4px;
  font-size: 12px;
}

.import-job-stat-created {
  color: #52c41a;
}

.import-job-stat-skipped {
  color: #faad14;
}

.import-job-stat-failed {
  color: #ff4d4f;
}
</style>
