<script setup lang="ts">
definePageMeta({
  layout: 'auth'
})

useSeoMeta({
  title: '登录',
  description: '登录 Koyomi Gal 账号'
})

const route = useRoute()
const { login } = useAuth()
const credentials = reactive({
  account: '',
  password: ''
})
const errorMessage = ref('')
const isSubmitting = ref(false)
const registrationComplete = computed(() => route.query.registered === '1')

function getLoginErrorMessage(error: unknown): string {
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

  return '登录失败，请检查账号和密码或稍后重试'
}

async function submitLogin() {
  if (isSubmitting.value) {
    return
  }

  errorMessage.value = ''
  isSubmitting.value = true

  try {
    await login({
      account: credentials.account.trim(),
      password: credentials.password
    })
    await navigateTo('/')
  } catch (error: unknown) {
    errorMessage.value = getLoginErrorMessage(error)
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <KunCard
    padding="none"
    class-name="login-card"
    content-class="login-card-content"
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

    <section class="form-panel" aria-labelledby="login-title">
      <div class="form-heading">
        <h1 id="login-title">欢迎回来</h1>
        <p>登录你的账号以继续使用 Koyomi Gal</p>
      </div>

      <p v-if="errorMessage" class="login-message login-error" role="alert">
        <KunIcon name="lucide:circle-alert" />
        <span>{{ errorMessage }}</span>
      </p>

      <p
        v-else-if="registrationComplete"
        class="login-message login-success"
        role="status"
      >
        <KunIcon name="lucide:circle-check" />
        <span>注册成功，请登录你的新账号</span>
      </p>

      <form @submit.prevent="submitLogin">
        <div class="form-fields">
          <KunInput
            v-model="credentials.account"
            label="使用邮箱或用户名登录"
            type="text"
            placeholder="请输入邮箱或用户名"
            autocomplete="username"
            :is-invalid="Boolean(errorMessage)"
            required
            autofocus
          />

          <KunInput
            v-model="credentials.password"
            label="密码"
            type="password"
            placeholder="请输入密码"
            autocomplete="current-password"
            :is-invalid="Boolean(errorMessage)"
            required
          />

          <KunButton
            type="submit"
            color="primary"
            size="lg"
            class-name="login-button"
            :disabled="isSubmitting"
            :aria-label="isSubmitting ? '登录中' : '登录'"
          >
            <KunIcon
              v-if="isSubmitting"
              name="lucide:loader-circle"
              class="loading-icon"
            />
            {{ isSubmitting ? '登录中...' : '登录' }}
          </KunButton>
        </div>
      </form>

      <div class="form-footer">
        <p>
          还没有账号？
          <NuxtLink to="/register">立即注册</NuxtLink>
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
.login-card {
  width: 100%;
  max-width: 896px;
  overflow: hidden;
}

:deep(.login-card-content) {
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
.form-heading p {
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
  margin-bottom: 32px;
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

.login-message {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: -12px 0 20px;
  padding: 10px 12px;
  border-radius: var(--radius-kun-md);
  font-size: 14px;
  line-height: 1.5;
}

.login-message :deep(svg) {
  flex: 0 0 auto;
  margin-top: 2px;
}

.login-error {
  background: color-mix(in srgb, var(--color-danger) 10%, transparent);
  color: var(--color-danger);
}

.login-success {
  background: color-mix(in srgb, var(--color-success) 10%, transparent);
  color: var(--color-success);
}

.form-fields {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.login-button {
  width: 100%;
}

.loading-icon {
  margin-right: 4px;
  animation: spin 0.8s linear infinite;
}

.form-footer {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 32px;
  padding-top: 24px;
  border-top: 1px solid var(--color-default-200);
  color: var(--color-default-500);
  font-size: 14px;
}

.form-footer p {
  margin: 0;
}

.form-footer a {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--color-primary);
}

.form-footer .back-link {
  align-self: flex-start;
}

.form-footer a:hover {
  text-decoration: underline;
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
  :deep(.login-card-content) {
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
</style>
