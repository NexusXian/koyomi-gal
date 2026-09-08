<script setup lang="ts">
import { message as antMessage } from 'ant-design-vue'

const props = withDefaults(
  defineProps<{
    conversationId: number | null
    disabled?: boolean
    disabledHint?: string
  }>(),
  { disabled: false, disabledHint: '' }
)

const messageStore = useMessageStore()
const content = ref('')
const sending = ref(false)
const maxChars = 2000

const charCount = computed(() => content.value.length)
const overLimit = computed(() => charCount.value > maxChars)
const canSend = computed(
  () =>
    !props.disabled &&
    !sending.value &&
    !overLimit.value &&
    content.value.trim().length > 0 &&
    props.conversationId !== null
)

async function send(): Promise<void> {
  if (!canSend.value || props.conversationId === null) return
  const text = content.value
  sending.value = true
  try {
    await messageStore.sendMessage(props.conversationId, text)
    content.value = ''
  } catch (error) {
    antMessage.error(getApiErrorMessage(error, '发送失败，请重试'))
  } finally {
    sending.value = false
  }
}

function handleKeydown(event: KeyboardEvent): void {
  if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
    event.preventDefault()
    void send()
  }
}
</script>

<template>
  <div class="message-composer">
    <p v-if="disabled && disabledHint" class="composer-hint">
      <KunIcon name="lucide:ban" />{{ disabledHint }}
    </p>
    <div class="composer-row">
      <textarea
        v-model="content"
        class="composer-input"
        :disabled="disabled"
        rows="2"
        maxlength="2100"
        placeholder="输入私信内容..."
        aria-label="私信内容"
        @keydown="handleKeydown"
      />
      <div class="composer-side">
        <span class="composer-count" :class="{ over: overLimit }">
          {{ charCount }} / {{ maxChars }}
        </span>
        <KunButton
          color="primary"
          variant="solid"
          size="sm"
          :disabled="!canSend"
          :aria-label="'发送'"
          @click="send"
        >
          <KunIcon name="lucide:send" />
          发送
        </KunButton>
      </div>
    </div>
  </div>
</template>

<style scoped>
.message-composer {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 14px 12px;
  border-top: 1px solid var(--app-glass-border);
}

.composer-hint {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--color-default-500);
  font-size: 13px;
}

.composer-row {
  display: flex;
  gap: 10px;
  align-items: flex-end;
}

.composer-input {
  flex: 1;
  resize: none;
  padding: 9px 12px;
  border: 1px solid var(--app-glass-border);
  border-radius: var(--radius-kun-md);
  background: var(--color-content1);
  color: var(--color-default-800);
  font-size: 14px;
  font-family: inherit;
  line-height: 1.6;
}

.composer-input:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--color-primary) 60%, transparent);
  outline-offset: 1px;
}

.composer-input:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.composer-side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
}

.composer-count {
  color: var(--color-default-400);
  font-size: 11px;
}

.composer-count.over {
  color: var(--color-danger, #ef4444);
  font-weight: 700;
}
</style>
