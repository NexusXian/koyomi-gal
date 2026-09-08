<script setup lang="ts">
import {
  forgotPasswordCode,
  forgotPasswordReset,
  forgotPasswordVerify
} from '~/api/generated/auth/auth'

definePageMeta({
  layout: 'auth'
})

useSeoMeta({
  title: '找回密码',
  description: '通过邮箱验证码找回 Koyomi Gal 账号密码'
})

type ForgotStep = 'email' | 'code' | 'password' | 'done'

const step = ref<ForgotStep>('email')
const email = ref('')
const code = ref('')
const password = ref('')
const confirmPassword = ref('')
const resetToken = ref('')
const errorMessage = ref('')
const statusMessage = ref('')
const isSubmitting = ref(false)
const isSendingCode = ref(false)
const { cooldown, start: startCooldown } = useSendCooldown(60)

const stepTitles: Record<ForgotStep, string> = {
  email: '验证邮箱',
  code: '输入验证码',
  password: '设置新密码',
  done: '重置成功'
}

const maskedEmail = computed(() => {
  const local = email.value.split('@')[0] ?? ''
  if (!local) {
    return ''
  }
  const visible = local.length < 3 ? 1 : 2
  return `${local.slice(0, visible)}***@${email.value.split('@')[1] ?? ''}`
})

const sendCodeLabel = computed(() => {
  if (isSendingCode.value) {
    return '发送中...'
  }
  return cooldown.value > 0 ? `${cooldown.value} 秒后重试` : '重新发送'
})

function getErrorMessage(error: unknown, fallback: string): string {
  return getApiErrorMessage(error, fallback)
}

async function sendCode(): Promise<void> {
  if (isSendingCode.value || cooldown.value > 0) {
    return
  }

  const normalized = email.value.trim().toLowerCase()
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalized)) {
    errorMessage.value = '请输入有效的邮箱地址'
    return
  }

  errorMessage.value = ''
  statusMessage.value = ''
  isSendingCode.value = true

  try {
    await forgotPasswordCode(
      { email: normalized },
      { skipAuth: true, skipRefresh: true }
    )
    email.value = normalized
    statusMessage.value = '如果该邮箱已注册，验证码将发送到邮箱'
    step.value = 'code'
    startCooldown()
  } catch (error: unknown) {
    errorMessage.value = getErrorMessage(error, '验证码发送失败，请稍后重试')
  } finally {
    isSendingCode.value = false
  }
}

async function verifyCode(): Promise<void> {
  if (isSubmitting.value) {
    return
  }

  if (!/^\d{6}$/.test(code.value)) {
    errorMessage.value = '请输入 6 位数字验证码'
    return
  }

  errorMessage.value = ''
  isSubmitting.value = true

  try {
    const data = unwrapApiData(
      await forgotPasswordVerify(
        { email: email.value, code: code.value },
        { skipAuth: true, skipRefresh: true }
      ),
      '验证码错误或已过期'
    )
    resetToken.value = data.reset_token ?? ''
    statusMessage.value = ''
    step.value = 'password'
  } catch (error: unknown) {
    errorMessage.value = getErrorMessage(error, '验证码错误或已过期')
  } finally {
    isSubmitting.value = false
  }
}

async function resetPassword(): Promise<void> {
  if (isSubmitting.value) {
    return
  }

  if (password.value.length < 8 || password.value.length > 72) {
    errorMessage.value = '密码长度必须为 8 到 72 个字符'
    return
  }
  if (password.value !== confirmPassword.value) {
    errorMessage.value = '两次输入的密码不一致'
    return
  }

  errorMessage.value = ''
  isSubmitting.value = true

  try {
    await forgotPasswordReset(
      {
        reset_token: resetToken.value,
        password: password.value,
        confirm_password: confirmPassword.value
      },
      { skipAuth: true, skipRefresh: true }
    )
    resetToken.value = ''
    statusMessage.value = ''
    step.value = 'done'
  } catch (error: unknown) {
    errorMessage.value = getErrorMessage(error, '密码重置失败，请稍后重试')
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <KunCard padding="none" class-name="forgot-card" content-class="forgot-card-content">
    <section class="form-panel" aria-labelledby="forgot-title">
      <div class="form-heading">
        <h1 id="forgot-title">找回密码</h1>
        <p v-if="step !== 'done'">
          第 {{ step === 'email' ? '1' : step === 'code' ? '2' : '3' }} 步 / 共 3 步：{{ stepTitles[step] }}
        </p>
        <p v-else>请使用新密码重新登录</p>
      </div>

      <p v-if="errorMessage" class="form-message form-error" role="alert">
        <KunIcon name="lucide:circle-alert" />
        <span>{{ errorMessage }}</span>
      </p>
      <p v-else-if="statusMessage" class="form-message form-success" role="status">
        <KunIcon name="lucide:circle-check" />
        <span>{{ statusMessage }}</span>
      </p>

      <form v-if="step === 'email'" @submit.prevent="sendCode">
        <div class="form-fields">
          <KunInput
            v-model="email"
            label="邮箱"
            type="email"
            placeholder="请输入注册邮箱"
            autocomplete="email"
            maxlength="254"
            :is-invalid="Boolean(errorMessage)"
            required
            autofocus
          />

          <KunButton
            type="submit"
            color="primary"
            size="lg"
            class-name="submit-button"
            :disabled="isSendingCode || cooldown > 0"
          >
            <KunIcon
              v-if="isSendingCode"
              name="lucide:loader-circle"
              class="loading-icon"
            />
            {{ isSendingCode ? '发送中...' : '发送验证码' }}
          </KunButton>
        </div>
      </form>

      <form v-else-if="step === 'code'" @submit.prevent="verifyCode">
        <div class="form-fields">
          <p class="step-hint">验证码已发送至：{{ maskedEmail }}</p>

          <KunInput
            v-model="code"
            label="验证码"
            type="text"
            placeholder="请输入 6 位验证码"
            autocomplete="one-time-code"
            inputmode="numeric"
            pattern="[0-9]{6}"
            minlength="6"
            maxlength="6"
            :is-invalid="Boolean(errorMessage)"
            required
            autofocus
          />

          <KunButton
            type="button"
            color="primary"
            variant="bordered"
            class-name="submit-button"
            :disabled="isSendingCode || cooldown > 0"
            @click="sendCode"
          >
            {{ sendCodeLabel }}
          </KunButton>

          <KunButton
            type="submit"
            color="primary"
            size="lg"
            class-name="submit-button"
            :disabled="isSubmitting"
          >
            <KunIcon
              v-if="isSubmitting"
              name="lucide:loader-circle"
              class="loading-icon"
            />
            {{ isSubmitting ? '验证中...' : '验证' }}
          </KunButton>
        </div>
      </form>

      <form v-else-if="step === 'password'" @submit.prevent="resetPassword">
        <div class="form-fields">
          <KunInput
            v-model="password"
            label="新密码"
            type="password"
            placeholder="请输入至少 8 位新密码"
            autocomplete="new-password"
            minlength="8"
            maxlength="72"
            :is-invalid="Boolean(errorMessage)"
            required
            autofocus
          />

          <KunInput
            v-model="confirmPassword"
            label="确认新密码"
            type="password"
            placeholder="请再次输入新密码"
            autocomplete="new-password"
            minlength="8"
            maxlength="72"
            :is-invalid="Boolean(errorMessage)"
            required
          />

          <KunButton
            type="submit"
            color="primary"
            size="lg"
            class-name="submit-button"
            :disabled="isSubmitting"
          >
            <KunIcon
              v-if="isSubmitting"
              name="lucide:loader-circle"
              class="loading-icon"
            />
            {{ isSubmitting ? '重置中...' : '重置密码' }}
          </KunButton>
        </div>
      </form>

      <div v-else class="done-panel">
        <KunIcon name="lucide:circle-check" class="done-icon" />
        <p class="done-title">密码重置成功</p>
        <p class="done-hint">请使用新密码重新登录</p>
        <KunButton
          color="primary"
          size="lg"
          class-name="submit-button"
          @click="navigateTo('/login')"
        >
          前往登录
        </KunButton>
      </div>

      <div class="form-footer">
        <NuxtLink class="back-link" to="/login">
          <KunIcon name="lucide:arrow-left" />
          返回登录
        </NuxtLink>
      </div>
    </section>
  </KunCard>
</template>

<style scoped>
.forgot-card {
  width: 100%;
  max-width: 448px;
  overflow: hidden;
}

.form-panel {
  padding: 32px;
}

.form-heading {
  margin-bottom: 28px;
}

.form-heading h1,
.form-heading p {
  margin: 0;
}

.form-heading h1 {
  color: var(--color-foreground);
  font-size: 24px;
  font-weight: 700;
  line-height: 32px;
}

.form-heading p {
  margin-top: 8px;
  color: var(--color-default-500);
  font-size: 14px;
}

.form-message {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: -10px 0 18px;
  padding: 10px 12px;
  border-radius: var(--radius-kun-md);
  font-size: 14px;
  line-height: 1.5;
}

.form-message :deep(svg) {
  flex: 0 0 auto;
  margin-top: 2px;
}

.form-error {
  background: color-mix(in srgb, var(--color-danger) 10%, transparent);
  color: var(--color-danger);
}

.form-success {
  background: color-mix(in srgb, var(--color-success) 10%, transparent);
  color: var(--color-success);
}

.form-fields {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.step-hint {
  margin: 0;
  color: var(--color-default-500);
  font-size: 14px;
}

.submit-button {
  width: 100%;
  margin-top: 4px;
}

.loading-icon {
  margin-right: 4px;
  animation: spin 0.8s linear infinite;
}

.done-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px 0 8px;
  text-align: center;
}

.done-icon {
  color: var(--color-success);
  font-size: 48px;
}

.done-title {
  margin: 8px 0 0;
  color: var(--color-foreground);
  font-size: 20px;
  font-weight: 700;
}

.done-hint {
  margin: 0 0 16px;
  color: var(--color-default-500);
  font-size: 14px;
}

.form-footer {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 28px;
  padding-top: 20px;
  border-top: 1px solid var(--color-default-200);
  color: var(--color-default-500);
  font-size: 14px;
}

.form-footer a {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--color-primary);
}

.form-footer a:hover {
  text-decoration: underline;
}

.form-footer .back-link {
  align-self: flex-start;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
