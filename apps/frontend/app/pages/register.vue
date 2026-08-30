<script setup lang="ts">
definePageMeta({
  layout: 'auth'
})

useSeoMeta({
  title: '注册',
  description: '注册 Koyomi Gal 账号'
})

const { register, sendRegistrationCode } = useAuth()
const credentials = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  verificationCode: ''
})
const errorMessage = ref('')
const codeMessage = ref('')
const isSubmitting = ref(false)
const isSendingCode = ref(false)
const sendCooldown = ref(0)
const verificationEmail = ref('')
let cooldownTimer: ReturnType<typeof setInterval> | undefined

const canSendCode = computed(
  () =>
    Boolean(credentials.email.trim()) &&
    !isSendingCode.value &&
    sendCooldown.value === 0
)

const sendCodeLabel = computed(() => {
  if (isSendingCode.value) {
    return '发送中...'
  }

  return sendCooldown.value > 0
    ? `${sendCooldown.value} 秒后重试`
    : '获取验证码'
})

function getRegistrationErrorMessage(error: unknown): string {
  const requestError = error as {
    data?: { msg?: unknown }
    name?: unknown
  }

  if (
    typeof requestError.data?.msg === 'string' &&
    requestError.data.msg.trim()
  ) {
    return requestError.data.msg
  }

  if (
    error instanceof Error &&
    error.name !== 'FetchError' &&
    error.message !== 'Request failed' &&
    error.message
  ) {
    return error.message
  }

  return '请求失败，请稍后重试'
}

function stopSendCooldown() {
  if (cooldownTimer) {
    clearInterval(cooldownTimer)
    cooldownTimer = undefined
  }

  sendCooldown.value = 0
}

function startSendCooldown() {
  stopSendCooldown()
  sendCooldown.value = 60

  cooldownTimer = setInterval(() => {
    if (sendCooldown.value <= 1) {
      stopSendCooldown()
      return
    }

    sendCooldown.value -= 1
  }, 1000)
}

async function sendCode() {
  if (!canSendCode.value) {
    return
  }

  const email = credentials.email.trim().toLowerCase()
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    errorMessage.value = '请输入有效的邮箱地址'
    return
  }

  errorMessage.value = ''
  codeMessage.value = ''
  isSendingCode.value = true

  try {
    codeMessage.value = await sendRegistrationCode(email)
    credentials.email = email
    verificationEmail.value = email
    startSendCooldown()
  } catch (error: unknown) {
    errorMessage.value = getRegistrationErrorMessage(error)
  } finally {
    isSendingCode.value = false
  }
}

watch(
  () => credentials.email,
  (email) => {
    if (
      verificationEmail.value &&
      email.trim().toLowerCase() !== verificationEmail.value
    ) {
      verificationEmail.value = ''
      codeMessage.value = ''
      stopSendCooldown()
    }
  }
)

async function submitRegistration() {
  if (isSubmitting.value) {
    return
  }

  errorMessage.value = ''

  if (!credentials.username.trim()) {
    errorMessage.value = '请输入用户名'
    return
  }

  if (credentials.password !== credentials.confirmPassword) {
    errorMessage.value = '两次输入的密码不一致'
    return
  }

  if (!/^\d{6}$/.test(credentials.verificationCode)) {
    errorMessage.value = '请输入 6 位数字验证码'
    return
  }

  isSubmitting.value = true

  try {
    await register({
      username: credentials.username.trim(),
      email: credentials.email.trim().toLowerCase(),
      password: credentials.password,
      confirmPassword: credentials.confirmPassword,
      verificationCode: credentials.verificationCode
    })
    await navigateTo({ path: '/login', query: { registered: '1' } })
  } catch (error: unknown) {
    errorMessage.value = getRegistrationErrorMessage(error)
  } finally {
    isSubmitting.value = false
  }
}

onScopeDispose(() => {
  stopSendCooldown()
})
</script>

<template>
  <KunCard
    padding="none"
    class-name="register-card"
    content-class="register-card-content"
  >
    <section class="brand-panel" aria-labelledby="brand-title">
      <div class="brand-heading">
        <span class="brand-logo" aria-hidden="true">
          <KunIcon name="lucide:gamepad-2" />
        </span>
        <div class="brand-copy">
          <p id="brand-title" class="brand-title">Koyomi Gal 账号</p>
          <p class="brand-subtitle">一个账号，连接你的 Galgame 世界</p>
        </div>
      </div>

      <div class="brand-illustration" aria-hidden="true">
        <span class="illustration-base" />
        <span class="illustration-orbit" />
        <span class="illustration-dot illustration-dot-large" />
        <span class="illustration-dot illustration-dot-small" />
        <span class="illustration-mark">
          <KunIcon name="lucide:sparkles" />
        </span>
      </div>
    </section>

    <section class="form-panel" aria-labelledby="register-title">
      <div class="form-heading">
        <h1 id="register-title">创建账号</h1>
        <p>注册 Koyomi Gal，开启你的 Galgame 世界</p>
      </div>

      <p v-if="errorMessage" class="form-message form-error" role="alert">
        <KunIcon name="lucide:circle-alert" />
        <span>{{ errorMessage }}</span>
      </p>

      <p v-else-if="codeMessage" class="form-message form-success" role="status">
        <KunIcon name="lucide:circle-check" />
        <span>{{ codeMessage }}</span>
      </p>

      <form @submit.prevent="submitRegistration">
        <div class="form-fields">
          <KunInput
            v-model="credentials.username"
            label="用户名"
            type="text"
            placeholder="请输入用户名"
            autocomplete="username"
            minlength="1"
            maxlength="50"
            :is-invalid="Boolean(errorMessage)"
            required
            autofocus
          />

          <div class="verification-row">
            <KunInput
              v-model="credentials.email"
              label="邮箱"
              type="email"
              placeholder="请输入邮箱"
              autocomplete="email"
              maxlength="254"
              :is-invalid="Boolean(errorMessage)"
              required
            />
            <KunButton
              type="button"
              color="primary"
              variant="bordered"
              class-name="verification-button"
              :disabled="!canSendCode"
              @click="sendCode"
            >
              {{ sendCodeLabel }}
            </KunButton>
          </div>

          <KunInput
            v-model="credentials.verificationCode"
            label="邮箱验证码"
            type="text"
            placeholder="请输入 6 位验证码"
            autocomplete="one-time-code"
            inputmode="numeric"
            pattern="[0-9]{6}"
            minlength="6"
            maxlength="6"
            :is-invalid="Boolean(errorMessage)"
            required
          />

          <KunInput
            v-model="credentials.password"
            label="密码"
            type="password"
            placeholder="请输入至少 8 位密码"
            autocomplete="new-password"
            minlength="8"
            maxlength="255"
            :is-invalid="Boolean(errorMessage)"
            required
          />

          <KunInput
            v-model="credentials.confirmPassword"
            label="确认密码"
            type="password"
            placeholder="请再次输入密码"
            autocomplete="new-password"
            minlength="8"
            maxlength="255"
            :is-invalid="Boolean(errorMessage)"
            required
          />

          <KunButton
            type="submit"
            color="primary"
            size="lg"
            class-name="register-button"
            :disabled="isSubmitting"
            :aria-label="isSubmitting ? '注册中' : '注册'"
          >
            <KunIcon
              v-if="isSubmitting"
              name="lucide:loader-circle"
              class="loading-icon"
            />
            {{ isSubmitting ? '注册中...' : '注册' }}
          </KunButton>
        </div>
      </form>

      <div class="form-footer">
        <p>
          已有账号？
          <NuxtLink to="/login">立即登录</NuxtLink>
        </p>
        <NuxtLink class="back-link" to="/">
          <KunIcon name="lucide:arrow-left" />
          返回首页
        </NuxtLink>
      </div>
    </section>
  </KunCard>
</template>

<style scoped>
.register-card {
  width: 100%;
  max-width: 896px;
  overflow: hidden;
}

:deep(.register-card-content) {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 0;
}

.brand-panel {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 32px;
  padding: 24px;
  background: var(--color-primary-50);
}

.brand-heading {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand-logo {
  display: grid;
  width: 40px;
  height: 40px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 12px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-secondary));
  box-shadow: var(--shadow-kun-sm);
  color: var(--color-primary-foreground);
  font-size: 22px;
}

.brand-copy {
  min-width: 0;
}

.brand-title,
.brand-subtitle,
.form-heading h1,
.form-heading p,
.form-footer p {
  margin: 0;
}

.brand-title {
  color: var(--color-foreground);
  font-size: 18px;
  font-weight: 700;
}

.brand-subtitle {
  margin-top: 2px;
  color: var(--color-default-500);
  font-size: 14px;
}

.brand-illustration {
  position: relative;
  display: none;
  height: 208px;
  align-items: flex-end;
  justify-content: center;
}

.illustration-base,
.illustration-orbit,
.illustration-dot,
.illustration-mark {
  position: absolute;
}

.illustration-base {
  bottom: 0;
  left: 50%;
  width: 224px;
  height: 48px;
  border-radius: 999px;
  background: var(--color-primary-100);
  transform: translateX(-50%);
}

.illustration-orbit {
  bottom: 40px;
  left: 50%;
  width: 128px;
  height: 128px;
  border: 2px solid var(--color-primary-200);
  border-radius: 50%;
  transform: translateX(-50%);
}

.illustration-dot {
  border-radius: 50%;
  background: var(--color-primary-200);
}

.illustration-dot-large {
  bottom: 80px;
  left: 8px;
  width: 40px;
  height: 40px;
}

.illustration-dot-small {
  right: 16px;
  bottom: 32px;
  width: 20px;
  height: 20px;
}

.illustration-mark {
  bottom: 74px;
  left: 50%;
  display: grid;
  width: 60px;
  height: 60px;
  place-items: center;
  border: 1px solid var(--color-primary-200);
  border-radius: 20px;
  background: var(--color-content1);
  box-shadow: var(--shadow-kun-md);
  color: var(--color-primary);
  font-size: 28px;
  transform: translateX(-50%) rotate(5deg);
}

.form-panel {
  padding: 24px;
}

.form-heading {
  margin-bottom: 28px;
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

.verification-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: end;
  gap: 10px;
}

.verification-button {
  min-width: 112px;
}

.register-button {
  width: 100%;
  margin-top: 4px;
}

.loading-icon {
  margin-right: 4px;
  animation: spin 0.8s linear infinite;
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
  color: var(--color-primary);
}

.form-footer a:hover {
  text-decoration: underline;
}

.form-footer .back-link {
  display: inline-flex;
  align-items: center;
  align-self: flex-start;
  gap: 6px;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (min-width: 640px) {
  .form-panel {
    padding: 32px;
  }
}

@media (min-width: 1024px) {
  :deep(.register-card-content) {
    grid-template-columns: 5fr 6fr;
  }

  .brand-panel,
  .form-panel {
    padding: 40px;
  }

  .brand-heading {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }

  .brand-logo {
    width: 48px;
    height: 48px;
    font-size: 26px;
  }

  .brand-title {
    font-size: 20px;
  }

  .brand-illustration {
    display: flex;
  }
}

@media (max-width: 479px) {
  .verification-row {
    grid-template-columns: minmax(0, 1fr);
    align-items: stretch;
  }

  .verification-button {
    width: 100%;
  }
}
</style>
