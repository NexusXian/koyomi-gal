<script setup lang="ts">
import { message } from 'ant-design-vue'
import { storeToRefs } from 'pinia'
import { checkin, getCheckinStatus } from '~/api/generated/me/me'
import type { LeveldtoCheckinResultData } from '~/api/generated/models'

const emit = defineEmits<{ checkedIn: [] }>()

const { isAuthenticated } = storeToRefs(useUserStore())
const checkedInToday = ref(false)
const consecutiveDays = ref(0)
const loading = ref(false)
const submitting = ref(false)
const lastResult = ref<LeveldtoCheckinResultData | null>(null)

async function loadStatus(): Promise<void> {
  if (!isAuthenticated.value) {
    return
  }
  loading.value = true
  try {
    const data = unwrapApiData(await getCheckinStatus(), '查询签到状态失败')
    checkedInToday.value = data.checked_in_today ?? false
    consecutiveDays.value = data.consecutive_days ?? 0
  } catch {
  } finally {
    loading.value = false
  }
}

async function submitCheckin(): Promise<void> {
  if (submitting.value || checkedInToday.value) {
    return
  }
  submitting.value = true
  try {
    const data = unwrapApiData(await checkin(), '签到失败')
    lastResult.value = data
    checkedInToday.value = true
    consecutiveDays.value = data.consecutive_days ?? consecutiveDays.value
    message.success(`签到成功，经验 +${data.exp_gained ?? 0}`)
    emit('checkedIn')
  } catch (error) {
    message.error(getApiErrorMessage(error, '签到失败'))
  } finally {
    submitting.value = false
  }
}

watch(isAuthenticated, () => void loadStatus(), { immediate: true })
</script>

<template>
  <KunCard padding="md" class-name="user-checkin-card">
    <div class="user-checkin-head">
      <span class="user-checkin-title">
        <KunIcon name="lucide:calendar-check" />
        每日签到
      </span>
      <span class="user-checkin-streak">已连续签到 {{ consecutiveDays }} 天</span>
    </div>
    <KunButton
      color="primary"
      :variant="checkedInToday ? 'bordered' : 'solid'"
      :disabled="checkedInToday"
      :loading="submitting || loading"
      class-name="user-checkin-button"
      @click="submitCheckin"
    >
      <KunIcon :name="checkedInToday ? 'lucide:check-circle' : 'lucide:sparkles'" />
      {{ checkedInToday ? '今日已签到' : '立即签到' }}
    </KunButton>
    <p v-if="lastResult" class="user-checkin-result">
      获得 {{ lastResult.exp_gained ?? 0 }} EXP · 当前 LV{{ lastResult.level ?? 1 }}
      {{ lastResult.level_name ?? '' }}
    </p>
  </KunCard>
</template>

<style scoped>
.user-checkin-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.user-checkin-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.user-checkin-title {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-weight: 700;
}

.user-checkin-streak {
  color: var(--color-foreground-500);
  font-size: 13px;
}

.user-checkin-button {
  width: 100%;
}

.user-checkin-result {
  margin: 0;
  color: var(--color-foreground-500);
  font-size: 12px;
}
</style>
