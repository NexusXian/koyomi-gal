<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { message } from 'ant-design-vue'
import type { BackgroundSize } from '~/types/background'

const open = defineModel<boolean>({ default: false })
const backgroundStore = useBackgroundStore()
const { settings, presets } = storeToRefs(backgroundStore)
const resetting = ref(false)

const opacityPercent = computed({
  get: () => Math.round(settings.value.opacity * 100),
  set: (value: number) => backgroundStore.setOpacity(value / 100)
})

const backgroundSize = computed({
  get: () => settings.value.size,
  set: (value: BackgroundSize) => backgroundStore.setSize(value)
})

async function resetBackground(): Promise<void> {
  resetting.value = true
  try {
    await backgroundStore.reset()
    message.success('背景设置已恢复默认')
  } catch (error) {
    message.error(error instanceof Error ? error.message : '背景设置重置失败')
  } finally {
    resetting.value = false
  }
}
</script>

<template>
  <a-drawer
    v-model:open="open"
    title="个性化背景"
    placement="right"
    width="min(420px, 100vw)"
  >
    <div class="settings-content">
      <section class="settings-section">
        <h3>背景</h3>
        <button
          type="button"
          class="none-option"
          :class="{ 'none-option-selected': settings.source === 'none' }"
          :aria-pressed="settings.source === 'none'"
          @click="backgroundStore.disableBackground"
        >
          <KunIcon name="lucide:ban" />
          无背景
          <KunIcon
            v-if="settings.source === 'none'"
            class="option-check"
            name="lucide:check"
          />
        </button>
      </section>

      <section class="settings-section">
        <h3>预置背景</h3>
        <p v-if="!presets.length" class="preset-empty">暂无预置背景</p>
        <div v-else class="preset-grid">
          <BackgroundPresetCard
            v-for="preset in presets"
            :key="preset.id"
            :preset="preset"
            :selected="settings.source === 'preset' && settings.presetId === preset.id"
            @select="backgroundStore.selectPreset(preset.id)"
          />
        </div>
      </section>

      <section class="settings-section">
        <h3>自定义背景</h3>
        <BackgroundUploader />
      </section>

      <section class="settings-section">
        <div class="section-heading">
          <h3>背景透明度</h3>
          <span>{{ opacityPercent }}%</span>
        </div>
        <a-slider v-model:value="opacityPercent" :min="0" :max="100" :step="5" />
      </section>

      <section class="settings-section">
        <h3>显示模式</h3>
        <a-radio-group v-model:value="backgroundSize" class="size-options">
          <a-radio value="cover">填充 Cover</a-radio>
          <a-radio value="contain">完整 Contain</a-radio>
        </a-radio-group>
      </section>

      <a-button block :loading="resetting" @click="resetBackground">
        <template #icon>
          <KunIcon name="lucide:rotate-ccw" />
        </template>
        恢复默认
      </a-button>
    </div>
  </a-drawer>
</template>

<style scoped>
.settings-content {
  display: flex;
  flex-direction: column;
  gap: 26px;
}

.settings-section h3 {
  margin: 0 0 12px;
  color: var(--color-foreground);
  font-size: 14px;
  font-weight: 650;
}

.none-option {
  position: relative;
  display: flex;
  width: 100%;
  align-items: center;
  gap: 8px;
  padding: 11px 12px;
  border: 1px solid var(--app-glass-border);
  border-radius: var(--radius-kun-md);
  background: var(--color-content2);
  color: var(--color-foreground);
  cursor: pointer;
  text-align: left;
}

.none-option:hover,
.none-option-selected {
  border-color: var(--color-primary);
}

.none-option:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

.option-check {
  margin-left: auto;
  color: var(--color-primary);
}

.preset-empty {
  margin: 0;
  color: var(--color-default-500);
  font-size: 12px;
}

.preset-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.section-heading span {
  color: var(--color-primary);
  font-size: 13px;
  font-weight: 650;
}

.size-options {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
</style>
