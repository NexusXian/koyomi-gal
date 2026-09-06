<script setup lang="ts">
import { listMyExperienceLogs } from '~/api/generated/me/me'
import type { LeveldtoExperienceLogData } from '~/api/generated/models'

const props = withDefaults(
  defineProps<{
    refreshKey?: number
  }>(),
  { refreshKey: 0 }
)

const loading = ref(true)
const errorMessage = ref('')
const items = ref<LeveldtoExperienceLogData[]>([])
const total = ref(0)
const page = ref(1)
const limit = 10
let requestSequence = 0

const totalPage = computed(() => Math.max(1, Math.ceil(total.value / limit)))

async function load(): Promise<void> {
  const sequence = ++requestSequence
  loading.value = true
  errorMessage.value = ''
  try {
    const data = unwrapApiData(
      await listMyExperienceLogs({ page: page.value, limit }),
      '查询经验记录失败'
    )
    if (sequence === requestSequence) {
      items.value = data.items ?? []
      total.value = data.total ?? 0
    }
  } catch (error) {
    if (sequence === requestSequence) {
      items.value = []
      total.value = 0
      errorMessage.value = getApiErrorMessage(error, '查询经验记录失败')
    }
  } finally {
    if (sequence === requestSequence) {
      loading.value = false
    }
  }
}

function updatePage(next: number): void {
  page.value = next
  void load()
}

function formatDelta(delta: number | undefined): string {
  const value = delta ?? 0
  return value > 0 ? `+${value}` : `${value}`
}

watch(
  () => [page.value, props.refreshKey] as const,
  () => void load(),
  { immediate: true }
)

defineExpose({ reload: load })
</script>

<template>
  <div class="user-experience-history">
    <a-alert
      v-if="errorMessage"
      type="error"
      show-icon
      :message="errorMessage"
    >
      <template #action>
        <KunButton size="sm" color="danger" variant="light" @click="load">
          重试
        </KunButton>
      </template>
    </a-alert>

    <a-alert
      v-else-if="!loading && items.length === 0"
      type="info"
      show-icon
      message="还没有经验记录"
      description="签到、评论、点赞和贡献审核都会在这里产生经验记录。"
    />

    <template v-else>
      <ul class="user-experience-list">
        <li v-for="item in items" :key="item.id" class="user-experience-item">
          <span class="user-experience-delta" :class="{ negative: (item.exp_delta ?? 0) < 0 }">
            {{ formatDelta(item.exp_delta) }}
          </span>
          <span class="user-experience-description">
            {{ item.description || item.event_type }}
          </span>
          <span class="user-experience-time">
            {{ new Date(item.created_at ?? '').toLocaleString('zh-CN') }}
          </span>
        </li>
      </ul>
      <div v-if="totalPage > 1" class="user-experience-pagination">
        <KunPagination
          :current-page="page"
          :total-page="totalPage"
          :is-loading="loading"
          @update:current-page="updatePage"
        />
      </div>
    </template>
  </div>
</template>

<style scoped>
.user-experience-history {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.user-experience-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.user-experience-item {
  display: grid;
  grid-template-columns: 64px 1fr auto;
  align-items: center;
  gap: 12px;
  padding: 9px 4px;
  border-bottom: 1px dashed var(--app-glass-border);
  font-size: 13px;
}

.user-experience-item:last-child {
  border-bottom: 0;
}

.user-experience-delta {
  color: var(--color-success-600, #16a34a);
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.user-experience-delta.negative {
  color: var(--color-danger-600, #dc2626);
}

.user-experience-description {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-experience-time {
  color: var(--color-foreground-500);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.user-experience-pagination {
  display: flex;
  justify-content: center;
}
</style>
